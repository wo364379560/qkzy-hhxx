package controller_chat

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/grouprolecache"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
)

const defaultGroupMemberMuteDuration = 45 * 24 * time.Hour
const groupAddFailedReasonOwnerMismatch = "不能邀请不同归属用户入群"

type FilterGroupMembersResult struct {
	InGroup    []string `json:"in_group"`
	NotInGroup []string `json:"not_in_group"`
}

type AddGroupMembersResult struct {
	Added          []string                             `json:"added"`
	AlreadyMembers []string                             `json:"already_members"`
	Failed         []tencentim.GroupAddMemberItemResult `json:"failed"`
}

func FilterGroupMembers(imUserID, groupID string, targetUserIDs []string) (FilterGroupMembersResult, error) {
	groupID = strings.TrimSpace(groupID)
	targetUserIDs = normalizeGroupUserIDs(targetUserIDs)
	if groupID == "" {
		return FilterGroupMembersResult{}, errors.New("group_id不能为空")
	}
	if len(targetUserIDs) == 0 {
		return FilterGroupMembersResult{}, nil
	}
	if err := ensureGroupVisibleToUser(imUserID, groupID); err != nil {
		return FilterGroupMembersResult{}, err
	}

	permission, err := canManageRegularMembersFromDB(groupID, imUserID)
	if err != nil {
		return FilterGroupMembersResult{}, err
	}
	if permission == nil {
		return FilterGroupMembersResult{}, errors.New("没有群管理权限")
	}

	// Only the candidate accounts matter here. Paging the whole group while
	// membership changes can repeat rows and cannot establish a stable snapshot.
	client := tencentim.NewClient()
	result := FilterGroupMembersResult{InGroup: []string{}, NotInGroup: []string{}}
	const batchSize = 100
	for start := 0; start < len(targetUserIDs); start += batchSize {
		batch := targetUserIDs[start:min(start+batchSize, len(targetUserIDs))]
		roles, err := client.GroupGetRoles(groupID, batch)
		if err != nil {
			return FilterGroupMembersResult{}, errors.New("查询好友入群状态失败，请稍后重试")
		}
		for _, id := range batch {
			switch roles[id] {
			case "NotMember":
				result.NotInGroup = append(result.NotInGroup, id)
			case groupRoleOwner, groupRoleAdmin, groupRoleMember:
				result.InGroup = append(result.InGroup, id)
			default:
				return FilterGroupMembersResult{}, errors.New("好友入群状态未确认，请重试")
			}
		}
	}
	return result, nil
}

func AddGroupMembers(imUserID, groupID string, targetUserIDs []string) (AddGroupMembersResult, error) {
	groupID = strings.TrimSpace(groupID)
	targetUserIDs = normalizeGroupUserIDs(targetUserIDs)
	if groupID == "" {
		return AddGroupMembersResult{}, errors.New("group_id不能为空")
	}
	if len(targetUserIDs) == 0 {
		return AddGroupMembersResult{}, errors.New("请选择要添加的成员")
	}
	if err := ensureGroupVisibleToUser(imUserID, groupID); err != nil {
		return AddGroupMembersResult{}, err
	}
	allowedUserIDs, rejectedUserIDs, err := filterInviteOwnerScope(imUserID, groupID, targetUserIDs)
	if err != nil {
		return AddGroupMembersResult{}, err
	}

	permission, err := canManageRegularMembersFromDB(groupID, imUserID)
	if err != nil {
		return AddGroupMembersResult{}, err
	}
	if permission == nil {
		return AddGroupMembersResult{}, errors.New("没有群管理权限")
	}
	if len(allowedUserIDs) == 0 {
		return AddGroupMembersResult{
			Failed: buildGroupAddFailedItems(rejectedUserIDs, groupAddFailedReasonOwnerMismatch),
		}, nil
	}

	client := tencentim.NewClient()
	// Silence=false：让腾讯原生下发入群系统提示，和踢人提示保持同一条链路。
	addResult, err := client.GroupAddMemberDetailed(groupID, allowedUserIDs, false)
	if err != nil {
		return AddGroupMembersResult{}, err
	}

	return AddGroupMembersResult{
		Added:          addResult.Added,
		AlreadyMembers: addResult.AlreadyMembers,
		Failed:         append(addResult.Failed, buildGroupAddFailedItems(rejectedUserIDs, groupAddFailedReasonOwnerMismatch)...),
	}, nil
}

func ensureInviteOwnerScope(imUserID, groupID string, targetUserIDs []string) error {
	allowed, rejected, err := filterInviteOwnerScope(imUserID, groupID, targetUserIDs)
	if err != nil {
		return err
	}
	if len(allowed) != len(targetUserIDs) || len(rejected) > 0 {
		return errors.New("不能邀请不同归属用户入群")
	}
	return nil
}

func filterInviteOwnerScope(imUserID, groupID string, targetUserIDs []string) ([]string, []string, error) {
	userRepo := repository.NewUserRepository()
	operator, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, nil, errors.New("操作用户不存在")
	}

	group, err := repository.NewGroupRepository().FindActiveByIMGroupID(groupID)
	if err != nil {
		return nil, nil, errors.New("群组不存在")
	}
	if operator.OwnerID != group.OwnerID {
		return nil, nil, errors.New("不能跨归属邀请成员入群")
	}

	targetUsers, err := userRepo.FindByIMUserIDs(targetUserIDs)
	if err != nil {
		return nil, nil, err
	}
	targetOwnerMap := make(map[string]uint, len(targetUsers))
	for _, user := range targetUsers {
		targetOwnerMap[user.IMUserID] = user.OwnerID
	}
	allowed := make([]string, 0, len(targetUserIDs))
	rejected := make([]string, 0)
	for _, userID := range targetUserIDs {
		if ownerID, ok := targetOwnerMap[userID]; !ok || ownerID != group.OwnerID {
			rejected = append(rejected, userID)
			continue
		}
		allowed = append(allowed, userID)
	}
	return allowed, rejected, nil
}

func buildGroupAddFailedItems(userIDs []string, reason string) []tencentim.GroupAddMemberItemResult {
	if len(userIDs) == 0 {
		return nil
	}
	failed := make([]tencentim.GroupAddMemberItemResult, 0, len(userIDs))
	for _, userID := range userIDs {
		failed = append(failed, tencentim.GroupAddMemberItemResult{
			Account: userID,
			Result:  -1,
			Reason:  reason,
		})
	}
	return failed
}

func RemoveGroupMember(imUserID, groupID, targetUserID string) error {
	groupID = strings.TrimSpace(groupID)
	targetUserID = strings.TrimSpace(targetUserID)
	if groupID == "" {
		return errors.New("group_id不能为空")
	}
	if targetUserID == "" {
		return errors.New("user_id不能为空")
	}
	if targetUserID == imUserID {
		return errors.New("不能移除自己")
	}
	if err := ensureGroupVisibleToUser(imUserID, groupID); err != nil {
		return err
	}

	if err := ensureGroupMemberManagementAllowed(groupID, imUserID, targetUserID); err != nil {
		return err
	}

	client := tencentim.NewClient()
	if err := client.GroupDeleteMember(groupID, []string{targetUserID}); err != nil {
		return err
	}
	_ = repository.NewGroupBusinessAdminRepository().MarkDeleted(groupID, targetUserID)
	grouprolecache.InvalidateGroup(context.Background(), groupID)
	return nil
}

func SetGroupMemberMute(imUserID, groupID, targetUserID string, mute bool) error {
	groupID = strings.TrimSpace(groupID)
	targetUserID = strings.TrimSpace(targetUserID)
	if groupID == "" {
		return errors.New("group_id不能为空")
	}
	if targetUserID == "" {
		return errors.New("user_id不能为空")
	}
	if targetUserID == imUserID {
		return errors.New("不能操作自己")
	}
	if err := ensureGroupVisibleToUser(imUserID, groupID); err != nil {
		return err
	}

	if err := ensureGroupMemberManagementAllowed(groupID, imUserID, targetUserID); err != nil {
		return err
	}

	muteTime := int64(0)
	if mute {
		muteTime = int64(defaultGroupMemberMuteDuration / time.Second)
	}
	return tencentim.NewClient().GroupSetMemberMuteTime(groupID, targetUserID, muteTime)
}

func normalizeGroupUserIDs(userIDs []string) []string {
	seen := make(map[string]struct{}, len(userIDs))
	result := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		userID = strings.TrimSpace(userID)
		if userID == "" {
			continue
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		result = append(result, userID)
	}
	return result
}
