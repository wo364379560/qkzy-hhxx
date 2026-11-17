package handler_admin

import (
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/handler/authlimit"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

// Get2FAEnabled 获取全局 2FA 开关。
func Get2FAEnabled(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	enabled, err := controller_admin.Get2FAEnabled()
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, map[string]bool{"enabled": enabled})
}

// GetChatQuietHoursPolicy 获取聊天静默时段策略。
func GetChatQuietHoursPolicy(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	policy, err := controller_admin.GetChatQuietHoursPolicy()
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, policy)
}

type set2FAReq struct {
	Enabled           bool   `json:"enabled"`
	CurrentPassword   string `json:"current_password" binding:"required"`
	ConfirmDisableAll bool   `json:"confirm_disable_all"`
}

type setChatQuietHoursReq struct {
	Enabled          bool   `json:"enabled"`
	TimeZone         string `json:"timezone"`
	Start            string `json:"start" binding:"required"`
	End              string `json:"end" binding:"required"`
	BlockRegister    bool   `json:"block_register"`
	BlockSendMessage bool   `json:"block_send_message"`
	BlockCreateGroup bool   `json:"block_create_group"`
	BlockAddFriend   bool   `json:"block_add_friend"`
	CurrentPassword  string `json:"current_password" binding:"required"`
}

// Set2FAEnabled 设置全局 2FA 开关。
func Set2FAEnabled(c *gin.Context) {
	role := c.GetString("admin_role")
	if !controller_admin.IsSuperAdminRole(role) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	var req set2FAReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	adminID := c.GetUint("admin_id")
	before, _ := controller_admin.Get2FAEnabled()
	if err := controller_admin.Set2FAEnabled(adminID, req.CurrentPassword, req.Enabled, req.ConfirmDisableAll); err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	middleware.SetAdminAccessTarget(c, "system_config", nil, "global_2fa", "全局2FA")
	middleware.SetAdminAccessAction(c, "config.update_2fa", "修改全局2FA")
	middleware.SetAdminAccessActionDetail(c, "修改全局 2FA 配置")
	if payload, err := json.Marshal(map[string]any{
		"enabled": map[string]any{
			"before": before,
			"after":  req.Enabled,
		},
	}); err == nil {
		middleware.SetAdminAccessChangeSet(c, string(payload))
	}
	response.OK(c, nil)
}

// SetChatQuietHoursPolicy 设置聊天静默时段策略。
func SetChatQuietHoursPolicy(c *gin.Context) {
	role := c.GetString("admin_role")
	if !controller_admin.IsSuperAdminRole(role) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	var req setChatQuietHoursReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	adminID := c.GetUint("admin_id")
	before, _ := controller_admin.GetChatQuietHoursPolicy()
	err := controller_admin.SetChatQuietHoursPolicy(adminID, req.CurrentPassword, config.ChatQuietHoursPolicy{
		Enabled:          req.Enabled,
		TimeZone:         req.TimeZone,
		Start:            req.Start,
		End:              req.End,
		BlockRegister:    req.BlockRegister,
		BlockSendMessage: req.BlockSendMessage,
		BlockCreateGroup: req.BlockCreateGroup,
		BlockAddFriend:   req.BlockAddFriend,
	})
	if err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	middleware.SetAdminAccessTarget(c, "system_config", nil, "chat_quiet_hours", "全局静默时段")
	middleware.SetAdminAccessAction(c, "config.update_chat_quiet_hours", "修改全局静默时段")
	middleware.SetAdminAccessActionDetail(c, "修改全局静默时段管控")
	if payload, marshalErr := json.Marshal(map[string]any{
		"enabled": map[string]any{
			"before": before.Enabled,
			"after":  req.Enabled,
		},
		"timezone": map[string]any{
			"before": before.TimeZone,
			"after":  req.TimeZone,
		},
		"start": map[string]any{
			"before": before.Start,
			"after":  req.Start,
		},
		"end": map[string]any{
			"before": before.End,
			"after":  req.End,
		},
		"block_register": map[string]any{
			"before": before.BlockRegister,
			"after":  req.BlockRegister,
		},
		"block_send_message": map[string]any{
			"before": before.BlockSendMessage,
			"after":  req.BlockSendMessage,
		},
		"block_create_group": map[string]any{
			"before": before.BlockCreateGroup,
			"after":  req.BlockCreateGroup,
		},
		"block_add_friend": map[string]any{
			"before": before.BlockAddFriend,
			"after":  req.BlockAddFriend,
		},
	}); marshalErr == nil {
		middleware.SetAdminAccessChangeSet(c, string(payload))
	}
	response.OK(c, nil)
}

// ResetAssistantTOTP 重置助理 TOTP。
func ResetAssistantTOTP(c *gin.Context) {
	role := c.GetString("admin_role")
	if !controller_admin.HasManagerPrivileges(role) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}

	if err := controller_admin.ResetAssistantTOTP(c.GetUint("admin_id"), uint(id)); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	assistantID := uint(id)
	middleware.SetAdminAccessTarget(c, "assistant", &assistantID, "", "")
	middleware.SetAdminAccessAction(c, "assistant.reset_totp", "重置助理2FA")
	middleware.SetAdminAccessActionDetail(c, "重置助理 2FA")
	response.OK(c, nil)
}
