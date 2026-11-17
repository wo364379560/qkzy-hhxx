package controller_admin

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	mathbig "math/big"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/internal/pkg/idgen"
	"github.com/ixayda/ix-Go/pkg/idgs"
)

// GroupResult 群组信息响应。
type GroupResult struct {
	ID            uint   `json:"id"`
	IMGroupID     string `json:"imGroupId"`
	Name          string `json:"name"`
	GroupType     string `json:"groupType"`
	OwnerID       uint   `json:"ownerID"`
	OwnerName     string `json:"ownerName"`
	OwnerNickname string `json:"ownerNickname"` // TIM 群主昵称
	MemberCount   int    `json:"memberCount"`
	IsMuted       bool   `json:"isMuted"` // 全员禁言
	Status        string `json:"status"`
	CreatedAt     string `json:"createdAt"`
}

func toGroupResult(g *model.Group) GroupResult {
	return GroupResult{
		ID:        g.ID,
		IMGroupID: g.IMGroupID,
		Name:      g.Name,
		GroupType: g.GroupType,
		OwnerID:   g.OwnerID,
		Status:    g.Status,
		CreatedAt: g.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ListGroupsReq 查询参数。
type ListGroupsReq struct {
	Page          int
	PageSize      int
	ScopeOwnerIDs []uint
	OwnerID       uint // 0 表示不限制（管理员）
	Keyword       string
	Status        string
	GroupType     string
}

// ListGroups 分页列出群组。
func ListGroups(req ListGroupsReq) (PageResult[GroupResult], error) {
	repo := repository.NewGroupRepository()
	groups, total, err := repo.ListPaged(req.Page, req.PageSize, req.ScopeOwnerIDs, req.OwnerID, req.Keyword, req.Status, req.GroupType)
	if err != nil {
		return PageResult[GroupResult]{}, err
	}

	// 批量查归属人名称（避免 N+1）
	ownerIDs := make([]uint, 0, len(groups))
	seen := map[uint]bool{}
	for _, g := range groups {
		if g.OwnerID > 0 && !seen[g.OwnerID] {
			ownerIDs = append(ownerIDs, g.OwnerID)
			seen[g.OwnerID] = true
		}
	}
	adminRepo := repository.NewAdminUserRepository()
	owners, _ := adminRepo.FindByIDs(ownerIDs)
	ownerMap := make(map[uint]string, len(owners))
	for _, o := range owners {
		ownerMap[o.ID] = o.Username
	}

	// 批量从腾讯 IM 获取群信息（成员数、群主、禁言状态，一次调用）
	type imGroupExtra struct {
		memberCount  int
		ownerAccount string
		isMuted      bool
	}
	imExtraMap := map[string]imGroupExtra{}
	imClient := tencentim.NewClient()
	if imClient.IsConfigured() && len(groups) > 0 {
		imIDs := make([]string, 0, len(groups))
		for _, g := range groups {
			if g.IMGroupID != "" {
				imIDs = append(imIDs, g.IMGroupID)
			}
		}
		if infoMap, err2 := imClient.GroupGetInfo(imIDs); err2 == nil {
			for id, info := range infoMap {
				imExtraMap[id] = imGroupExtra{
					memberCount:  info.MemberNum,
					ownerAccount: info.OwnerAccount,
					isMuted:      info.ShutUpAllMember,
				}
			}
		}
	}

	// 批量查群主昵称（通过 IM 账号反查本地用户）
	ownerAccountSet := make([]string, 0)
	seenAcc := map[string]bool{}
	for _, ex := range imExtraMap {
		if ex.ownerAccount != "" && !seenAcc[ex.ownerAccount] {
			ownerAccountSet = append(ownerAccountSet, ex.ownerAccount)
			seenAcc[ex.ownerAccount] = true
		}
	}
	userRepo := repository.NewUserRepository()
	ownerUsers, _ := userRepo.FindByIMUserIDs(ownerAccountSet)
	ownerNicknameMap := make(map[string]string, len(ownerUsers))
	for _, u := range ownerUsers {
		ownerNicknameMap[u.IMUserID] = u.Nickname
	}

	list := make([]GroupResult, 0, len(groups))
	for i := range groups {
		r := toGroupResult(&groups[i])
		r.OwnerName = ownerMap[groups[i].OwnerID]
		ex := imExtraMap[groups[i].IMGroupID]
		r.MemberCount = ex.memberCount
		r.IsMuted = ex.isMuted
		r.OwnerNickname = ownerNicknameMap[ex.ownerAccount]
		if r.OwnerNickname == "" {
			r.OwnerNickname = ex.ownerAccount // 降级显示 IM 账号
		}
		list = append(list, r)
	}
	return PageResult[GroupResult]{List: list, Total: total}, nil
}

// GetGroup 获取单个群组详情。
func GetGroup(groupID, adminID uint, role string) (*GroupResult, error) {
	group, err := findManagedGroup(adminID, groupID)
	if err != nil {
		return nil, err
	}

	result := toGroupResult(group)
	if group.OwnerID > 0 {
		if owner, ownerErr := repository.NewAdminUserRepository().FindByID(group.OwnerID); ownerErr == nil {
			result.OwnerName = owner.Username
		}
	}

	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() || strings.TrimSpace(group.IMGroupID) == "" {
		return &result, nil
	}

	infoMap, err := imClient.GroupGetInfo([]string{group.IMGroupID})
	if err != nil {
		return &result, nil
	}
	if info, ok := infoMap[group.IMGroupID]; ok {
		result.MemberCount = info.MemberNum
		result.IsMuted = info.ShutUpAllMember
		if strings.TrimSpace(info.OwnerAccount) != "" {
			if ownerUser, ownerErr := repository.NewUserRepository().FindByIMUserID(info.OwnerAccount); ownerErr == nil {
				result.OwnerNickname = ownerUser.Nickname
			}
			if result.OwnerNickname == "" {
				result.OwnerNickname = info.OwnerAccount
			}
		}
	}

	return &result, nil
}

// CreateGroupReq 创建群组参数。
type CreateGroupReq struct {
	Name          string
	GroupType     string
	OwnerID       uint // 群组归属的 admin/assistant ID，由 handler 在请求层决定
	CreatedBy     uint
	MemberIDs     []uint
	OwnerMemberID uint // 指定群主的本地用户 ID，0 = 不指定
}

// CreateGroup 创建群组并同步腾讯云 IM。
func CreateGroup(req CreateGroupReq) (*GroupResult, error) {
	if req.GroupType == "" {
		req.GroupType = "Work"
	}
	resolvedOwnerID, err := resolveManagedOwnerID(req.CreatedBy, true, req.OwnerID)
	if err != nil {
		return nil, err
	}
	req.OwnerID = resolvedOwnerID
	if err := ensureAnyCreateGroupAllowedNow(context.Background(), req.OwnerID, time.Now()); err != nil {
		return nil, err
	}

	// 校验成员归属：所有成员必须属于同一个 owner，且与 req.OwnerID 一致
	if len(req.MemberIDs) > 0 {
		userRepo := repository.NewUserRepository()

		checkIDs := make([]uint, 0, len(req.MemberIDs)+1)
		checkIDs = append(checkIDs, req.MemberIDs...)
		if req.OwnerMemberID > 0 {
			checkIDs = append(checkIDs, req.OwnerMemberID)
		}

		for _, uid := range checkIDs {
			u, err := userRepo.FindByID(uid)
			if err != nil {
				return nil, errors.New("成员用户不存在")
			}
			if req.OwnerID > 0 && u.OwnerID != req.OwnerID {
				return nil, errors.New("成员中包含不属于指定归属人的用户")
			}
		}
	}

	ctx := context.Background()
	imGroupID, err := idgen.GenGroupID(ctx, "ga")
	if err != nil {
		return nil, err
	}

	group := &model.Group{
		BizID:        idgs.Global.GroupGenerate().Int64(),
		IMGroupID:    imGroupID,
		Name:         req.Name,
		GroupType:    req.GroupType,
		OwnerID:      req.OwnerID,
		OwnerBinding: "independent",
		CreatedBy:    req.CreatedBy,
		Status:       "active",
		CreatedAt:    time.Now(),
	}
	repo := repository.NewGroupRepository()
	if err := repo.Create(group); err != nil {
		return nil, err
	}

	imClient := tencentim.NewClient()
	if imClient.IsConfigured() {
		ownerIMUserID := ""
		if req.OwnerMemberID > 0 {
			userRepo := repository.NewUserRepository()
			if u, err := userRepo.FindByID(req.OwnerMemberID); err == nil {
				ownerIMUserID = u.IMUserID
			}
		}

		timGroupID, err := imClient.GroupCreate(imGroupID, req.Name, ownerIMUserID, req.GroupType)
		if err != nil {
			_ = repo.Delete(group.ID)
			return nil, err
		}
		// TIM 可能返回不同的 ID（极少情况），以返回值为准
		if timGroupID != imGroupID {
			group.IMGroupID = timGroupID
			_ = repo.Update(group)
		}

		if len(req.MemberIDs) > 0 {
			if memberIMIDs := resolveIMUserIDs(req.MemberIDs); len(memberIMIDs) > 0 {
				_ = imClient.GroupAddMember(group.IMGroupID, memberIMIDs)
			}
		}
	}

	result := toGroupResult(group)
	return &result, nil
}

func ensureAnyCreateGroupAllowedNow(ctx context.Context, ownerID uint, now time.Time) error {
	policy, hasPolicy, err := resolveCreateGroupQuietHoursPolicy(ctx, ownerID)
	if err != nil {
		return err
	}
	if !hasPolicy {
		return nil
	}
	blocked, err := config.IsChatQuietHoursActionBlockedNow(
		policy,
		config.ChatQuietHoursActionCreateGroup,
		now,
	)
	if err != nil {
		return err
	}
	if blocked {
		return errors.New(
			config.ChatQuietHoursBlockedMessage(
				config.ChatQuietHoursActionCreateGroup,
				policy,
			),
		)
	}
	return nil
}

func resolveCreateGroupQuietHoursPolicy(ctx context.Context, ownerID uint) (config.ChatQuietHoursPolicy, bool, error) {
	if ownerID == 0 {
		policy, err := config.GetChatQuietHoursPolicy(ctx, datasource.Datasource.Mysql.GetClient())
		if err != nil {
			return config.ChatQuietHoursPolicy{}, false, err
		}
		return policy, true, nil
	}

	adminRepo := repository.NewAdminUserRepository()
	adminUser, err := adminRepo.FindByID(ownerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return config.ChatQuietHoursPolicy{}, false, nil
		}
		return config.ChatQuietHoursPolicy{}, false, err
	}

	if IsSuperAdminRole(adminUser.Role) {
		policy, err := config.GetChatQuietHoursPolicy(ctx, datasource.Datasource.Mysql.GetClient())
		if err != nil {
			return config.ChatQuietHoursPolicy{}, false, err
		}
		return policy, true, nil
	}

	policyRecord, err := repository.NewAdminChatQuietHoursPolicyRepository().FindByAdminUserID(ownerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			policy, globalErr := config.GetChatQuietHoursPolicy(ctx, datasource.Datasource.Mysql.GetClient())
			if globalErr != nil {
				return config.ChatQuietHoursPolicy{}, false, globalErr
			}
			return policy, true, nil
		}
		return config.ChatQuietHoursPolicy{}, false, err
	}

	policy := config.DefaultChatQuietHoursPolicy()
	policy.Enabled = policyRecord.Enabled
	policy.TimeZone = policyRecord.TimeZone
	policy.Start = policyRecord.Start
	policy.End = policyRecord.End
	policy.BlockRegister = policyRecord.BlockRegister
	policy.BlockSendMessage = policyRecord.BlockSendMessage
	policy.BlockCreateGroup = policyRecord.BlockCreateGroup
	policy.BlockAddFriend = policyRecord.BlockAddFriend
	if err := config.NormalizeChatQuietHoursPolicy(&policy); err != nil {
		return config.ChatQuietHoursPolicy{}, false, err
	}
	return policy, true, nil
}

// DismissGroup 解散群组（status=dismissed + 同步腾讯云）。
func DismissGroup(adminID uint, isAdmin bool, groupID uint) error {
	repo := repository.NewGroupRepository()
	group, err := findManagedGroup(adminID, groupID)
	if err != nil {
		return err
	}
	if group.Status == "dismissed" {
		return errors.New("群组已解散")
	}
	if err = repo.UpdateStatus(groupID, "dismissed"); err != nil {
		return err
	}
	_ = tencentim.NewClient().GroupDestroy(group.IMGroupID)
	return nil
}

// GroupMemberResult 群成员信息响应。
type GroupMemberResult struct {
	IMUserID string `json:"imUserID"`
	Nickname string `json:"nickname"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`     // Owner / Admin / Member
	JoinTime int64  `json:"joinTime"` // Unix 时间戳
}

// GetGroupMembers 获取群成员列表（从腾讯 IM 拉取并与本地用户数据合并）。
func GetGroupMembers(adminID uint, isAdmin bool, groupID uint) ([]GroupMemberResult, error) {
	group, err := findManagedGroup(adminID, groupID)
	if err != nil {
		return nil, err
	}

	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return []GroupMemberResult{}, nil
	}

	members, err := imClient.GroupGetMembers(group.IMGroupID)
	if err != nil {
		return nil, err
	}

	// 批量查本地用户（昵称、手机）
	imIDs := make([]string, 0, len(members))
	for _, m := range members {
		imIDs = append(imIDs, m.Account)
	}
	userRepo := repository.NewUserRepository()
	users, _ := userRepo.FindByIMUserIDs(imIDs)
	userMap := make(map[string]*model.User, len(users))
	for i := range users {
		userMap[users[i].IMUserID] = &users[i]
	}

	result := make([]GroupMemberResult, 0, len(members))
	for _, m := range members {
		r := GroupMemberResult{
			IMUserID: m.Account,
			Role:     m.Role,
			JoinTime: m.JoinTime,
		}
		if u, ok := userMap[m.Account]; ok {
			r.Nickname = u.Nickname
			r.Phone = u.Phone
		}
		result = append(result, r)
	}
	return result, nil
}

// BatchMuteGroupDetail 表示批量静默单条结果。
type BatchMuteGroupDetail struct {
	GroupID   uint   `json:"groupID"`
	GroupName string `json:"groupName"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
}

// BatchMuteGroupResult 表示批量静默汇总结果。
type BatchMuteGroupResult struct {
	Success int                    `json:"success"`
	Failed  int                    `json:"failed"`
	Details []BatchMuteGroupDetail `json:"details"`
}

// AddGroupMembers 添加群成员。
func AddGroupMembers(adminID uint, isAdmin bool, groupID uint, memberIDs []uint) error {
	group, err := findManagedGroup(adminID, groupID)
	if err != nil {
		return err
	}
	memberIMIDs := resolveIMUserIDs(memberIDs)
	if len(memberIMIDs) == 0 {
		return errors.New("未找到有效用户")
	}
	return tencentim.NewClient().GroupAddMember(group.IMGroupID, memberIMIDs)
}

// RemoveGroupMembers 移除群成员。
func RemoveGroupMembers(adminID uint, isAdmin bool, groupID uint, memberIDs []uint) error {
	group, err := findManagedGroup(adminID, groupID)
	if err != nil {
		return err
	}
	memberIMIDs := resolveIMUserIDs(memberIDs)
	if len(memberIMIDs) == 0 {
		return errors.New("未找到有效用户")
	}
	return tencentim.NewClient().GroupDeleteMember(group.IMGroupID, memberIMIDs)
}

// resolveIMUserIDs 将本地用户 ID 列表转换为 IM UserID 列表。
func resolveIMUserIDs(userIDs []uint) []string {
	userRepo := repository.NewUserRepository()
	imIDs := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		u, err := userRepo.FindByID(id)
		if err == nil && u.IMUserID != "" {
			imIDs = append(imIDs, u.IMUserID)
		}
	}
	return imIDs
}

// randSuffix 生成指定长度的随机字符串（使用 charset 字符集）。
func randSuffix(n int) string {
	b := make([]byte, n)
	for i := range b {
		idx, err := cryptorand.Int(cryptorand.Reader, mathbig.NewInt(int64(len(charset))))
		if err != nil {
			b[i] = charset[0]
			continue
		}
		b[i] = charset[idx.Int64()]
	}
	return string(b)
}

// MuteGroup 开启/关闭全员禁言。
func MuteGroup(adminID uint, isAdmin bool, groupID uint, mute bool) error {
	group, err := findManagedGroup(adminID, groupID)
	if err != nil {
		return err
	}
	if group.Status == "dismissed" {
		return errors.New("群组已解散")
	}
	return tencentim.NewClient().GroupMuteAll(group.IMGroupID, mute)
}

// BatchMuteGroups 批量开启或关闭全员禁言。
func BatchMuteGroups(adminID uint, isAdmin bool, groupIDs []uint, mute bool) (BatchMuteGroupResult, error) {
	result := BatchMuteGroupResult{
		Details: make([]BatchMuteGroupDetail, 0, len(groupIDs)),
	}

	if len(groupIDs) == 0 {
		return result, errors.New("请选择至少一个群组")
	}

	groupRepo := repository.NewGroupRepository()
	seen := make(map[uint]bool, len(groupIDs))
	for _, groupID := range groupIDs {
		if seen[groupID] {
			continue
		}
		seen[groupID] = true

		group, err := groupRepo.FindByID(groupID)
		if err != nil {
			message := "群组不存在"
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				message = err.Error()
			}
			result.Failed++
			result.Details = append(result.Details, BatchMuteGroupDetail{
				GroupID: groupID,
				Success: false,
				Message: message,
			})
			continue
		}

		err = MuteGroup(adminID, isAdmin, groupID, mute)
		if err != nil {
			result.Failed++
			result.Details = append(result.Details, BatchMuteGroupDetail{
				GroupID:   group.ID,
				GroupName: group.Name,
				Success:   false,
				Message:   err.Error(),
			})
			continue
		}

		result.Success++
		result.Details = append(result.Details, BatchMuteGroupDetail{
			GroupID:   group.ID,
			GroupName: group.Name,
			Success:   true,
			Message: func() string {
				if mute {
					return "已静默"
				}
				return "已取消静默"
			}(),
		})
	}

	return result, nil
}

// RepairGroup 将本地群组补同步到腾讯云 IM（适用于历史孤立群）。
func RepairGroup(adminID uint, isAdmin bool, groupID uint) error {
	repo := repository.NewGroupRepository()
	group, err := findManagedGroup(adminID, groupID)
	if err != nil {
		return err
	}
	if group.Status == "dismissed" {
		return errors.New("群组已解散")
	}

	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return errors.New("IM 未配置")
	}

	newIMGroupID, err := idgen.GenGroupID(context.Background(), "ga")
	if err != nil {
		return err
	}
	timGroupID, err := imClient.GroupCreate(newIMGroupID, group.Name, "", group.GroupType)
	if err != nil {
		return err
	}
	group.IMGroupID = timGroupID
	return repo.Update(group)
}

// SyncGroupPreview 同步预览项。
type SyncGroupPreview struct {
	ImGroupID     string `json:"imGroupId"`
	Name          string `json:"name"`
	OwnerNickname string `json:"ownerNickname"`
	MemberCount   int    `json:"memberCount"`
	IsNew         bool   `json:"isNew"`
	LocalGroupID  uint   `json:"localGroupId,omitempty"`
	OwnerName     string `json:"ownerName"`
	OwnerID       uint   `json:"ownerId"`
	Action        string `json:"action"`
}

// SyncGroupsPreview 从腾讯 IM 拉取所有群组并生成同步预览列表。
func SyncGroupsPreview(adminID uint, isAdmin bool) ([]SyncGroupPreview, error) {
	if !isAdmin {
		return nil, errors.New("仅管理员可执行同步操作")
	}
	actor, err := BuildActorContext(adminID)
	if err != nil {
		return nil, err
	}

	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return nil, errors.New("IM 未配置")
	}

	// 从腾讯 IM 拉取所有群组
	imGroups, err := imClient.GroupGetAllGroups()
	if err != nil {
		return nil, err
	}

	// 查询本地所有群组
	groupRepo := repository.NewGroupRepository()
	var localGroups []model.Group
	if IsSuperAdminRole(actor.Role) {
		localGroups, err = groupRepo.FindAll()
	} else {
		localGroups, err = groupRepo.FindAllByOwners(actor.ScopeAdminIDs)
	}
	if err != nil {
		return nil, err
	}
	localGroupMap := make(map[string]*model.Group, len(localGroups))
	for i := range localGroups {
		localGroupMap[localGroups[i].IMGroupID] = &localGroups[i]
	}

	// 批量获取群组详细信息（成员数、群主等）
	imGroupIDs := make([]string, 0, len(imGroups))
	for _, g := range imGroups {
		imGroupIDs = append(imGroupIDs, g.GroupID)
	}
	imInfoMap, _ := imClient.GroupGetInfo(imGroupIDs)

	// 批量查询群主对应的本地用户（用于匹配归属）
	ownerAccounts := make([]string, 0)
	for _, info := range imInfoMap {
		if info.OwnerAccount != "" {
			ownerAccounts = append(ownerAccounts, info.OwnerAccount)
		}
	}
	userRepo := repository.NewUserRepository()
	ownerUsers, _ := userRepo.FindByIMUserIDs(ownerAccounts)
	ownerUserMap := make(map[string]*model.User, len(ownerUsers))
	for i := range ownerUsers {
		ownerUserMap[ownerUsers[i].IMUserID] = &ownerUsers[i]
	}

	// 批量查询归属人名称
	ownerIDs := make([]uint, 0)
	seenOwnerID := map[uint]bool{}
	for _, u := range ownerUsers {
		if u.OwnerID > 0 && !seenOwnerID[u.OwnerID] {
			ownerIDs = append(ownerIDs, u.OwnerID)
			seenOwnerID[u.OwnerID] = true
		}
	}
	// 添加本地已存在群组的归属人
	for _, lg := range localGroups {
		if lg.OwnerID > 0 && !seenOwnerID[lg.OwnerID] {
			ownerIDs = append(ownerIDs, lg.OwnerID)
			seenOwnerID[lg.OwnerID] = true
		}
	}
	adminRepo := repository.NewAdminUserRepository()
	owners, _ := adminRepo.FindByIDs(ownerIDs)
	ownerNameMap := make(map[uint]string, len(owners))
	for _, o := range owners {
		ownerNameMap[o.ID] = o.Username
	}

	// 生成预览列表
	previews := make([]SyncGroupPreview, 0, len(imGroups))
	for _, g := range imGroups {
		info := imInfoMap[g.GroupID]
		localGroup := localGroupMap[g.GroupID]
		isNew := localGroup == nil

		var ownerID uint
		var ownerName string
		var ownerNickname string

		if isNew {
			// 新群组：根据群主匹配归属
			if info.OwnerAccount != "" {
				if ownerUser := ownerUserMap[info.OwnerAccount]; ownerUser != nil {
					ownerID = ownerUser.OwnerID
					ownerNickname = ownerUser.Nickname
				}
			}
			// 如果找不到群主对应的本地用户，归属到当前管理员
			if ownerID == 0 {
				ownerID = adminID
			}
			if !CanManageOwnerID(actor, ownerID) {
				continue
			}
			ownerName = ownerNameMap[ownerID]
		} else {
			// 已存在群组：使用本地归属
			ownerID = localGroup.OwnerID
			if !CanManageOwnerID(actor, ownerID) {
				continue
			}
			ownerName = ownerNameMap[ownerID]
			if info.OwnerAccount != "" {
				if ownerUser := ownerUserMap[info.OwnerAccount]; ownerUser != nil {
					ownerNickname = ownerUser.Nickname
				}
			}
		}

		if ownerNickname == "" {
			ownerNickname = info.OwnerAccount
		}

		preview := SyncGroupPreview{
			ImGroupID:     g.GroupID,
			Name:          g.GroupName,
			OwnerNickname: ownerNickname,
			MemberCount:   info.MemberNum,
			IsNew:         isNew,
			OwnerName:     ownerName,
			OwnerID:       ownerID,
			Action:        "skip",
		}
		if !isNew {
			preview.LocalGroupID = localGroup.ID
		}
		previews = append(previews, preview)
	}

	return previews, nil
}

// SyncSelection 同步选择项。
type SyncSelection struct {
	ImGroupID string
	Action    string
}

// SyncGroupsConfirm 确认同步选中的群组。
func SyncGroupsConfirm(adminID uint, isAdmin bool, selections []SyncSelection) (map[string]interface{}, error) {
	if !isAdmin {
		return nil, errors.New("仅管理员可执行同步操作")
	}
	actor, err := BuildActorContext(adminID)
	if err != nil {
		return nil, err
	}

	imClient := tencentim.NewClient()
	if !imClient.IsConfigured() {
		return nil, errors.New("IM 未配置")
	}

	groupRepo := repository.NewGroupRepository()
	userRepo := repository.NewUserRepository()

	success := 0
	failed := 0

	for _, sel := range selections {
		if sel.Action == "skip" {
			continue
		}

		// 获取 IM 群组信息
		infoMap, err := imClient.GroupGetInfo([]string{sel.ImGroupID})
		if err != nil || len(infoMap) == 0 {
			failed++
			continue
		}
		info := infoMap[sel.ImGroupID]

		// 确定归属人
		var ownerID uint = adminID
		if info.OwnerAccount != "" {
			if ownerUser, err := userRepo.FindByIMUserID(info.OwnerAccount); err == nil {
				ownerID = ownerUser.OwnerID
			}
		}

		if sel.Action == "import" {
			groupType := info.Type
			if groupType == "" {
				groupType = "Work"
			}
			// 导入新群组
			group := &model.Group{
				BizID:        idgs.Global.GroupGenerate().Int64(),
				IMGroupID:    sel.ImGroupID,
				Name:         info.Name,
				GroupType:    groupType,
				OwnerID:      ownerID,
				OwnerBinding: "independent",
				CreatedBy:    adminID,
				Status:       "active",
				CreatedAt:    time.Now(),
			}
			if !CanManageOwnerID(actor, ownerID) {
				failed++
				continue
			}
			if err := groupRepo.Create(group); err != nil {
				failed++
				continue
			}
			success++
		} else if sel.Action == "update" {
			// 更新已存在群组
			localGroup, err := groupRepo.FindByIMGroupID(sel.ImGroupID)
			if err != nil {
				failed++
				continue
			}
			if !CanManageOwnerID(actor, localGroup.OwnerID) {
				failed++
				continue
			}
			localGroup.Name = info.Name
			if info.Type == "" {
				localGroup.GroupType = "Work"
			} else {
				localGroup.GroupType = info.Type
			}
			if err := groupRepo.Update(localGroup); err != nil {
				failed++
				continue
			}
			success++
		}
	}

	return map[string]interface{}{
		"success": success,
		"failed":  failed,
	}, nil
}
