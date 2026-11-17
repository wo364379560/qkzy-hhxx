package controller_chat

import (
	"context"
	"errors"
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/grouprolecache"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"gorm.io/gorm"
)

type groupManagementPermission struct {
	Role            string
	IsOwner         bool
	IsTencentAdmin  bool
	IsBusinessAdmin bool
}

func canManageRegularMembersFromDB(groupID, imUserID string) (*groupManagementPermission, error) {
	imUserID = strings.TrimSpace(imUserID)
	roles, err := refreshTencentGroupMemberRoles(groupID, []string{imUserID}, imUserID)
	if err != nil {
		return nil, err
	}
	return groupManagementPermissionForRole(groupID, imUserID, roles[imUserID])
}

// Tencent owns native roles; local business grants apply only to current members.
// A previously synced owner is not a freshness marker for the entire admin list.
func groupManagementPermissionForRole(groupID, imUserID, tencentRole string) (*groupManagementPermission, error) {
	switch tencentRole {
	case groupRoleOwner:
		return &groupManagementPermission{Role: repository.GroupRoleOwner, IsOwner: true}, nil
	case groupRoleAdmin:
		return &groupManagementPermission{Role: repository.GroupRoleTencentAdmin, IsTencentAdmin: true}, nil
	case "NotMember":
		return nil, nil
	case groupRoleMember:
	default:
		return nil, errors.New("腾讯群成员身份数据不完整")
	}
	row, err := repository.NewGroupBusinessAdminRepository().FindActive(groupID, imUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	switch strings.TrimSpace(row.Role) {
	case repository.GroupRoleBusinessAdmin, "":
		return &groupManagementPermission{Role: repository.GroupRoleBusinessAdmin, IsBusinessAdmin: true}, nil
	}
	return nil, nil
}

func ensureGroupMemberManagementAllowed(groupID, imUserID, targetUserID string) error {
	// Read operator and target together. Missing local rows must not make a
	// Tencent owner/admin appear to be a regular member to the REST admin account.
	roles, err := refreshTencentGroupMemberRoles(groupID, []string{imUserID, targetUserID}, imUserID)
	if err != nil {
		return err
	}
	caller, err := groupManagementPermissionForRole(groupID, imUserID, roles[imUserID])
	if err != nil {
		return err
	}
	if caller == nil {
		return errors.New("没有群管理权限")
	}
	if roles[targetUserID] == "NotMember" {
		return errors.New("目标用户不是群成员")
	}
	target, err := groupManagementPermissionForRole(groupID, targetUserID, roles[targetUserID])
	if err != nil {
		return err
	}
	if target == nil {
		return nil
	}
	switch target.Role {
	case repository.GroupRoleOwner:
		return errors.New("不能操作群主")
	case repository.GroupRoleTencentAdmin:
		if !caller.IsOwner {
			return errors.New("不能操作群管理员")
		}
	case repository.GroupRoleBusinessAdmin:
		if !caller.IsOwner {
			return errors.New("不能操作业务管理员")
		}
	}
	return nil
}

// SyncTencentGroupMemberRoles handles authenticated role-change notifications.
// Query current roles rather than replaying a delayed or repeated callback.
func SyncTencentGroupMemberRoles(groupID string, userIDs []string, updatedBy string) error {
	_, err := refreshTencentGroupMemberRoles(groupID, userIDs, updatedBy)
	return err
}

func refreshTencentGroupMemberRoles(groupID string, userIDs []string, updatedBy string) (map[string]string, error) {
	groupID = strings.TrimSpace(groupID)
	userIDs = normalizeGroupUserIDs(userIDs)
	if groupID == "" || len(userIDs) == 0 {
		return nil, errors.New("群组和成员不能为空")
	}
	roles, err := tencentim.NewClient().GroupGetRoles(groupID, userIDs)
	if err != nil {
		return nil, errors.New("查询腾讯群成员身份失败，请稍后重试")
	}
	snapshot := make(map[string]string, len(userIDs))
	for _, userID := range userIDs {
		role := strings.TrimSpace(roles[userID])
		switch role {
		case groupRoleOwner, groupRoleAdmin, groupRoleMember, "NotMember":
			snapshot[userID] = role
		default:
			return nil, errors.New("腾讯群成员身份数据不完整")
		}
	}
	if err := repository.NewGroupBusinessAdminRepository().SyncTencentRoles(groupID, snapshot, updatedBy, false); err != nil {
		return nil, err
	}
	grouprolecache.InvalidateGroup(context.Background(), groupID)
	return snapshot, nil
}
