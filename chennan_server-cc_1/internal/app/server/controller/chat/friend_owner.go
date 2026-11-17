package controller_chat

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/messagelimit"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/pkg/logger"
)

// ErrInfoCrossOwnerFriendAdd 跨服务空间加好友时返回给客户端的提示。
const ErrInfoCrossOwnerFriendAdd = "当前账号属于其他服务空间无法直接添加。"

func ownerIsSuperAdmin(ownerID uint) bool {
	if ownerID == 0 {
		return false
	}
	admin, err := repository.NewAdminUserRepository().FindByID(ownerID)
	if err != nil || admin == nil {
		return false
	}
	return admin.Role == controller_admin.RoleSuperAdmin && admin.Status == "active"
}

// defaultRegisterOwnerID 无邀请码注册时挂到当前活跃超管。
func defaultRegisterOwnerID() uint {
	admins, err := repository.NewAdminUserRepository().ListByRole(controller_admin.RoleSuperAdmin)
	if err != nil {
		logger.Warnf("lookup default register owner failed: %v", err)
		return 0
	}
	if len(admins) == 0 {
		return 0
	}
	return admins[0].ID
}

// ShouldMigrateOwnerOnPrevFriendAdd 保留旧调用兼容；所有前置阶段都只校验。
// 即使允许直接添加，也必须等确认双向关系之后再分配。
func ShouldMigrateOwnerOnPrevFriendAdd(to *model.User) bool {
	return false
}

func userOwnerIsSuperAdmin(u *model.User) bool {
	if u == nil {
		return false
	}
	return ownerIsSuperAdmin(u.OwnerID)
}

// isFriendOwnerPool 未进入具体运营空间：owner_id=0，或挂在超管名下。
func isFriendOwnerPool(u *model.User) bool {
	return isFriendOwnerPoolWith(u, ownerIsSuperAdmin)
}

func isFriendOwnerPoolWith(u *model.User, superAdmin func(uint) bool) bool {
	if u == nil {
		return false
	}
	if u.OwnerID == 0 {
		return true
	}
	if superAdmin == nil {
		return false
	}
	return superAdmin(u.OwnerID)
}

// CanSearchUser 是否允许在加好友搜索里看到对方。
// 同归属可搜；超管/未归属池互搜可搜；池用户只能搜到已开全局搜索的助理/管理员账号；
// 已进助理/管理员的用户不能跨空间互搜。
func CanSearchUser(caller *model.User, target *model.User) bool {
	return model.DiscoverOwnerPair(caller, target, model.ClassifyOwner(caller, ownerForPolicy(caller)), model.ClassifyOwner(target, ownerForPolicy(target))) == ""
}

func canSearchUserWith(caller *model.User, target *model.User, superAdmin func(uint) bool) bool {
	if caller == nil || target == nil {
		return false
	}
	if caller.OwnerID == target.OwnerID {
		return true
	}
	callerPool := isFriendOwnerPoolWith(caller, superAdmin)
	targetPool := isFriendOwnerPoolWith(target, superAdmin)
	if callerPool && targetPool {
		return true
	}
	if callerPool && !targetPool {
		return target.GloballySearchable
	}
	if !callerPool && targetPool {
		return true
	}
	return false
}

// CanCrossOwnerFriendAdd 是否允许加好友。规则与可搜索范围一致。
func CanCrossOwnerFriendAdd(from *model.User, to *model.User) bool {
	return CanSearchUser(from, to)
}

func canCrossOwnerFriendAdd(from *model.User, to *model.User) bool {
	return canCrossOwnerFriendAddWith(from, to, ownerIsSuperAdmin)
}

func canCrossOwnerFriendAddWith(from *model.User, to *model.User, superAdmin func(uint) bool) bool {
	return canSearchUserWith(from, to, superAdmin)
}

func pickFriendAddMigrate(initiator, peer *model.User) (move *model.User, newOwnerID uint, ok bool) {
	return pickFriendAddMigrateWith(initiator, peer, ownerIsSuperAdmin)
}

func pickFriendAddMigrateWith(initiator, peer *model.User, superAdmin func(uint) bool) (move *model.User, newOwnerID uint, ok bool) {
	if initiator == nil || peer == nil {
		return nil, 0, false
	}
	initiatorPool := isFriendOwnerPoolWith(initiator, superAdmin)
	peerPool := isFriendOwnerPoolWith(peer, superAdmin)
	switch {
	case initiatorPool && !peerPool && peer.OwnerID > 0:
		return initiator, peer.OwnerID, true
	case peerPool && !initiatorPool && initiator.OwnerID > 0:
		return peer, initiator.OwnerID, true
	default:
		return nil, 0, false
	}
}

// FriendAddBlockedReason 若不允许添加，返回提示文案；允许则返回空字符串。
func FriendAddBlockedReason(from, to *model.User) string {
	code := model.DiscoverOwnerPair(from, to, model.ClassifyOwner(from, ownerForPolicy(from)), model.ClassifyOwner(to, ownerForPolicy(to)))
	if code == "" {
		return ""
	}
	return OwnerReasonMessage(code)
}

// MigrateOwnerByFriendAdd 双方成为好友后分配归属：
// 超管/未归属池用户和已进助理/管理员的用户成为好友后，把池用户划到对方运营方。
func MigrateOwnerByFriendAdd(initiatorIMUserID, peerIMUserID string) error {
	return migrateOwnerByFriendAdd(initiatorIMUserID, peerIMUserID, true)
}

// MaybeMigrateOwnerByPair 发消息/登录兜底：不打「无需迁移」日志，避免刷屏。
func MaybeMigrateOwnerByPair(a, b string) error {
	return migrateOwnerByFriendAdd(a, b, false)
}

func migrateOwnerByFriendAdd(a, b string, logSkip bool) error {
	_, err := QueueFriendOwnership(context.Background(), "internal", repository.OwnershipID(), a, b)
	return err
}

func ensureAssignedRealnameByFriendAdd(user *model.User) {
	if user == nil || user.OwnerID == 0 || userOwnerIsSuperAdmin(user) {
		return
	}
	if err := repository.NewUserRealNameRepository().UpsertAssignedVerified(user.ID); err != nil {
		logger.Warnf("friend-add auto realname failed: user=%s owner=%d err=%v", user.IMUserID, user.OwnerID, err)
		return
	}
	messagelimit.InvalidateRealNameVerified(context.Background(), user.ID)
}

func inviteCodeForNewOwner(ownerID uint, stay *model.User) string {
	if ownerID == 0 {
		return ""
	}
	return pickMigratedInviteCode(lookupActiveOwnerInviteCode(ownerID), stay)
}

func lookupActiveOwnerInviteCode(ownerID uint) string {
	ic, err := repository.NewInviteCodeRepository().FindActiveByOwner(ownerID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warnf("lookup owner invite code failed: owner=%d err=%v", ownerID, err)
		}
		return ""
	}
	if ic == nil {
		return ""
	}
	return strings.TrimSpace(ic.Code)
}

// ReconcileOwnerByFriends 未归属用户登录后，按 IM 好友列表补一次归属。
func ReconcileOwnerByFriends(imUserID string) {
	if strings.TrimSpace(imUserID) == "" {
		return
	}
	if _, err := QueueFriendOwnership(context.Background(), "reconcile", repository.OwnershipID(), imUserID, ""); err != nil {
		logger.Warnf("enqueue friend ownership reconciliation failed user=%s", imUserID)
	}
}

func pickMigratedInviteCode(ownerActiveCode string, stay *model.User) string {
	if code := strings.TrimSpace(ownerActiveCode); code != "" {
		return code
	}
	if stay != nil {
		return strings.TrimSpace(stay.InviteCode)
	}
	return ""
}
