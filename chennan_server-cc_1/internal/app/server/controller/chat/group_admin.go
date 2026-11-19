package controller_chat

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/grouprolecache"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/logger"
	"gorm.io/gorm"
)

const (
	groupRoleOwner  = "Owner"
	groupRoleAdmin  = "Admin"
	groupRoleMember = "Member"

	groupAdminSourceTencent         = "tencent"
	groupAdminSourceTencentRoleSync = "tencent_role_sync"
	groupAdminSourceBusiness        = "business"
)

type GroupAdminMemberResult struct {
	UserID     string `json:"user_id"`
	Nickname   string `json:"nickname,omitempty"`
	AvatarURL  string `json:"avatar_url,omitempty"`
	Role       string `json:"role"`
	RoleSource string `json:"role_source"`
}

type GroupAdminsResult struct {
	Owner  *GroupAdminMemberResult  `json:"owner,omitempty"`
	Admins []GroupAdminMemberResult `json:"admins"`
}

func ListGroupAdmins(imUserID, groupID string) (*GroupAdminsResult, error) {
	return ListGroupAdminsContext(context.Background(), imUserID, groupID)
}

func ListGroupAdminsContext(ctx context.Context, imUserID, groupID string) (*GroupAdminsResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, errors.New("group_id不能为空")
	}
	// Authorization remains per caller, outside the shared display-only read.
	group, err := SearchGroup(ctx, imUserID, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, errors.New("群组不存在")
	}
	return groupAdminReads.load(ctx, groupID, func(readCtx context.Context) (*GroupAdminsResult, error) {
		started := time.Now()
		members, err := tencentim.NewClient().GroupGetAllMembersContext(readCtx, groupID)
		upstreamMS := time.Since(started).Milliseconds()
		if err != nil {
			logger.Warnf("group_admin_read stage=tencent elapsed_ms=%d", upstreamMS)
			return nil, err
		}
		db := datasource.Datasource.Mysql.GetClient().WithContext(readCtx)
		if err := syncTencentGroupAdminRolesWithDB(db, groupID, members, imUserID); err != nil {
			return nil, err
		}
		result, err := buildGroupAdminsResultWithDB(db, groupID, members)
		logger.Infof("group_admin_read members=%d tencent_ms=%d db_ms=%d total_ms=%d", len(members), upstreamMS, time.Since(started).Milliseconds()-upstreamMS, time.Since(started).Milliseconds())
		return result, err
	})
}

func AddGroupAdmin(imUserID, groupID, targetUserID string) (*GroupAdminMemberResult, error) {
	groupID = strings.TrimSpace(groupID)
	targetUserID = strings.TrimSpace(targetUserID)
	if groupID == "" {
		return nil, errors.New("group_id不能为空")
	}
	if targetUserID == "" {
		return nil, errors.New("user_id不能为空")
	}
	if err := ensureGroupVisibleToUser(imUserID, groupID); err != nil {
		return nil, err
	}

	client := tencentim.NewClient()
	members, err := client.GroupGetAllMembers(groupID)
	if err != nil {
		return nil, err
	}
	if err := syncTencentGroupAdminRoles(groupID, members, imUserID); err != nil {
		return nil, err
	}
	if !isTencentOwner(imUserID, members) {
		return nil, errors.New("仅群主可以设置群管理员")
	}

	targetMember, ok := findGroupMember(targetUserID, members)
	if !ok {
		return nil, errors.New("目标用户不是群成员")
	}
	if targetMember.Role == groupRoleOwner {
		return enrichGroupAdminMember(GroupAdminMemberResult{
			UserID:     targetUserID,
			Role:       groupRoleOwner,
			RoleSource: groupAdminSourceTencent,
		})
	}
	if targetMember.Role == groupRoleAdmin {
		if _, err := repository.NewGroupBusinessAdminRepository().UpsertActiveRole(
			groupID,
			targetUserID,
			imUserID,
			groupAdminSourceTencent,
			repository.GroupRoleTencentAdmin,
		); err != nil {
			return nil, err
		}
		grouprolecache.InvalidateGroup(context.Background(), groupID)
		return enrichGroupAdminMember(GroupAdminMemberResult{
			UserID:     targetUserID,
			Role:       groupRoleAdmin,
			RoleSource: groupAdminSourceTencent,
		})
	}

	if err := client.GroupSetMemberRole(groupID, targetUserID, groupRoleAdmin); err != nil {
		if !isTencentAdminLimitError(err) {
			return nil, err
		}
		if _, upsertErr := repository.NewGroupBusinessAdminRepository().UpsertActiveRole(
			groupID,
			targetUserID,
			imUserID,
			"tencent_overflow",
			repository.GroupRoleBusinessAdmin,
		); upsertErr != nil {
			return nil, upsertErr
		}
		grouprolecache.InvalidateGroup(context.Background(), groupID)
		return enrichGroupAdminMember(GroupAdminMemberResult{
			UserID:     targetUserID,
			Role:       groupRoleAdmin,
			RoleSource: groupAdminSourceBusiness,
		})
	}

	if _, err := repository.NewGroupBusinessAdminRepository().UpsertActiveRole(
		groupID,
		targetUserID,
		imUserID,
		groupAdminSourceTencent,
		repository.GroupRoleTencentAdmin,
	); err != nil {
		return nil, err
	}
	grouprolecache.InvalidateGroup(context.Background(), groupID)
	return enrichGroupAdminMember(GroupAdminMemberResult{
		UserID:     targetUserID,
		Role:       groupRoleAdmin,
		RoleSource: groupAdminSourceTencent,
	})
}

func RemoveGroupAdmin(imUserID, groupID, targetUserID string) error {
	groupID = strings.TrimSpace(groupID)
	targetUserID = strings.TrimSpace(targetUserID)
	if groupID == "" {
		return errors.New("group_id不能为空")
	}
	if targetUserID == "" {
		return errors.New("user_id不能为空")
	}
	if err := ensureGroupVisibleToUser(imUserID, groupID); err != nil {
		return err
	}

	client := tencentim.NewClient()
	members, err := client.GroupGetAllMembers(groupID)
	if err != nil {
		return err
	}
	if err := syncTencentGroupAdminRoles(groupID, members, imUserID); err != nil {
		return err
	}
	if !isTencentOwner(imUserID, members) {
		return errors.New("仅群主可以移除群管理员")
	}
	if targetUserID == imUserID {
		return errors.New("不能移除自己的群主权限")
	}

	targetMember, ok := findGroupMember(targetUserID, members)
	if ok && targetMember.Role == groupRoleOwner {
		return errors.New("不能移除群主")
	}
	if ok && targetMember.Role == groupRoleAdmin {
		if err := client.GroupSetMemberRole(groupID, targetUserID, groupRoleMember); err != nil {
			return err
		}
	}
	if err := repository.NewGroupBusinessAdminRepository().MarkDeleted(groupID, targetUserID); err != nil {
		return err
	}
	grouprolecache.InvalidateGroup(context.Background(), groupID)
	return nil
}

func ensureGroupVisibleToUser(imUserID, groupID string) error {
	group, err := SearchGroup(context.Background(), imUserID, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return errors.New("群组不存在")
	}
	return nil
}

func buildGroupAdminsResult(groupID string, members []tencentim.GroupMember) (*GroupAdminsResult, error) {
	return buildGroupAdminsResultWithDB(datasource.Datasource.Mysql.GetClient(), groupID, members)
}
func buildGroupAdminsResultWithDB(db *gorm.DB, groupID string, members []tencentim.GroupMember) (*GroupAdminsResult, error) {
	memberMap := make(map[string]tencentim.GroupMember, len(members))
	adminIDs := make([]string, 0)
	enrichIDs := make([]string, 0, len(members))
	var ownerID string
	for _, member := range members {
		member.Account = strings.TrimSpace(member.Account)
		if member.Account == "" {
			continue
		}
		memberMap[member.Account] = member
		switch member.Role {
		case groupRoleOwner:
			ownerID = member.Account
			enrichIDs = append(enrichIDs, member.Account)
		case groupRoleAdmin:
			adminIDs = append(adminIDs, member.Account)
			enrichIDs = append(enrichIDs, member.Account)
		}
	}

	businessAdmins, err := repository.NewGroupBusinessAdminRepositoryWithDB(db).ListActiveByGroup(groupID)
	if err != nil {
		return nil, err
	}

	seenAdmin := make(map[string]struct{}, len(adminIDs)+len(businessAdmins))
	result := &GroupAdminsResult{
		Admins: make([]GroupAdminMemberResult, 0, len(adminIDs)+len(businessAdmins)),
	}
	if ownerID != "" {
		result.Owner = &GroupAdminMemberResult{
			UserID:     ownerID,
			Role:       groupRoleOwner,
			RoleSource: groupAdminSourceTencent,
		}
	}
	for _, adminID := range adminIDs {
		result.Admins = append(result.Admins, GroupAdminMemberResult{
			UserID:     adminID,
			Role:       groupRoleAdmin,
			RoleSource: groupAdminSourceTencent,
		})
		seenAdmin[adminID] = struct{}{}
	}
	for _, businessAdmin := range businessAdmins {
		userID := strings.TrimSpace(businessAdmin.UserID)
		if userID == "" || userID == ownerID {
			continue
		}
		switch strings.TrimSpace(businessAdmin.Role) {
		case repository.GroupRoleBusinessAdmin, "":
		default:
			continue
		}
		if _, exists := memberMap[userID]; !exists {
			continue
		}
		if _, exists := seenAdmin[userID]; exists {
			continue
		}
		result.Admins = append(result.Admins, GroupAdminMemberResult{
			UserID:     userID,
			Role:       groupRoleAdmin,
			RoleSource: groupAdminSourceBusiness,
		})
		seenAdmin[userID] = struct{}{}
		enrichIDs = append(enrichIDs, userID)
	}

	enrichGroupAdminMembersWithDB(db, result, enrichIDs)
	return result, nil
}

func syncTencentGroupAdminRoles(groupID string, members []tencentim.GroupMember, updatedBy string) error {
	return syncTencentGroupAdminRolesWithDB(datasource.Datasource.Mysql.GetClient(), groupID, members, updatedBy)
}
func syncTencentGroupAdminRolesWithDB(db *gorm.DB, groupID string, members []tencentim.GroupMember, updatedBy string) error {
	roles := make(map[string]string, len(members))
	for _, member := range members {
		account := strings.TrimSpace(member.Account)
		if account == "" {
			return errors.New("腾讯群成员身份数据不完整")
		}
		roles[account] = strings.TrimSpace(member.Role)
	}
	if err := repository.NewGroupBusinessAdminRepositoryWithDB(db).SyncTencentRoles(groupID, roles, updatedBy, true); err != nil {
		return err
	}
	grouprolecache.InvalidateGroup(context.Background(), groupID)
	return nil
}

func enrichGroupAdminMember(item GroupAdminMemberResult) (*GroupAdminMemberResult, error) {
	enriched := &item
	users, err := repository.NewUserRepository().FindByIMUserIDs([]string{item.UserID})
	if err != nil {
		return nil, err
	}
	if len(users) > 0 {
		enriched.Nickname = users[0].Nickname
		enriched.AvatarURL = users[0].AvatarURL
	}
	return enriched, nil
}

func enrichGroupAdminMembersWithDB(db *gorm.DB, result *GroupAdminsResult, userIDs []string) {
	users, err := repository.NewUserRepositoryWithDB(db).FindByIMUserIDs(userIDs)
	if err != nil {
		return
	}
	userMap := make(map[string]model.User, len(users))
	for _, user := range users {
		userMap[user.IMUserID] = user
	}
	apply := func(item *GroupAdminMemberResult) {
		if item == nil {
			return
		}
		if user, ok := userMap[item.UserID]; ok {
			item.Nickname = user.Nickname
			item.AvatarURL = user.AvatarURL
		}
	}
	apply(result.Owner)
	for i := range result.Admins {
		apply(&result.Admins[i])
	}
}

func isTencentOwner(imUserID string, members []tencentim.GroupMember) bool {
	member, ok := findGroupMember(imUserID, members)
	return ok && member.Role == groupRoleOwner
}

func findGroupMember(userID string, members []tencentim.GroupMember) (tencentim.GroupMember, bool) {
	userID = strings.TrimSpace(userID)
	for _, member := range members {
		if strings.TrimSpace(member.Account) == userID {
			return member, true
		}
	}
	return tencentim.GroupMember{}, false
}

func isTencentAdminLimitError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "number of admin reaches") ||
		strings.Contains(message, "admin reaches the limit")
}
