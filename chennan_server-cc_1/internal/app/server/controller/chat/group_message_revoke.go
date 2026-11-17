package controller_chat

import (
	"errors"
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
)

// RevokeGroupMessage 群主/管理员撤回群消息（服务端调用 TIM，无 2 分钟限制）。
func RevokeGroupMessage(imUserID, groupID string, msgSeq uint64) error {
	imUserID = strings.TrimSpace(imUserID)
	groupID = strings.TrimSpace(groupID)
	if imUserID == "" {
		return errors.New("未登录")
	}
	if groupID == "" {
		return errors.New("group_id不能为空")
	}
	if msgSeq == 0 {
		return errors.New("msg_seq无效")
	}
	if err := ensureGroupVisibleToUser(imUserID, groupID); err != nil {
		return err
	}

	permission, err := canManageRegularMembersFromDB(groupID, imUserID)
	if err != nil {
		return err
	}
	if permission == nil {
		return errors.New("没有群管理权限")
	}
	return tencentim.NewClient().RecallGroupMessage(groupID, msgSeq)
}
