package controller_chat

import (
	"errors"
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
)

// An upload authorization must be tied to the authenticated user, not form IDs.
func AuthorizeAvatarUpload(account, kind, userID, groupID string) error {
	if account == "" {
		return errors.New("请先登录")
	}
	if kind == "user" {
		if strings.TrimSpace(userID) != account {
			return errors.New("不能修改其他用户头像")
		}
		return nil
	}
	if err := ensureGroupVisibleToUser(account, groupID); err != nil {
		return err
	}
	client := tencentim.NewClient()
	role, err := client.GroupGetRole(groupID, account)
	if err != nil {
		return errors.New("群成员权限查询失败，请稍后重试")
	}
	if role == "Owner" || role == "Admin" {
		return nil
	}
	if role != "Member" {
		return errors.New("尚未加入此群，不能修改群头像")
	}
	groups, err := client.GroupGetInfo([]string{groupID})
	if err != nil {
		return errors.New("群类型查询失败，请稍后重试")
	}
	if group, ok := groups[groupID]; ok && group.Type == "Work" {
		return nil
	}
	return errors.New("只有群主或管理员可以修改群头像")
}
