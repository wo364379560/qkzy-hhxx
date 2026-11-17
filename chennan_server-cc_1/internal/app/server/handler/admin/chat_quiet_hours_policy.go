package handler_admin

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/handler/authlimit"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type saveAccountChatQuietHoursPolicyReq struct {
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

type clearAccountChatQuietHoursPolicyReq struct {
	CurrentPassword string `json:"current_password" binding:"required"`
}

// GetManagerChatQuietHoursPolicy 获取管理员账号级静默时段策略。
func GetManagerChatQuietHoursPolicy(c *gin.Context) {
	managerID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	result, err := controller_admin.GetManagerChatQuietHoursPolicy(c.GetUint("admin_id"), managerID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// SaveManagerChatQuietHoursPolicy 保存管理员账号级静默时段策略。
func SaveManagerChatQuietHoursPolicy(c *gin.Context) {
	managerID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	var req saveAccountChatQuietHoursPolicyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	before, _ := controller_admin.GetManagerChatQuietHoursPolicy(c.GetUint("admin_id"), managerID)
	if err := controller_admin.SaveManagerChatQuietHoursPolicy(controller_admin.SaveAccountChatQuietHoursPolicyReq{
		ActorID:           c.GetUint("admin_id"),
		TargetAdminUserID: managerID,
		CurrentPassword:   req.CurrentPassword,
		Policy: config.ChatQuietHoursPolicy{
			Enabled:          req.Enabled,
			TimeZone:         req.TimeZone,
			Start:            req.Start,
			End:              req.End,
			BlockRegister:    req.BlockRegister,
			BlockSendMessage: req.BlockSendMessage,
			BlockCreateGroup: req.BlockCreateGroup,
			BlockAddFriend:   req.BlockAddFriend,
		},
	}); err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "admin_user", &managerID, "", "管理员静默时段策略")
	middleware.SetAdminAccessAction(c, "config.save_manager_quiet_hours", "保存管理员静默时段策略")
	middleware.SetAdminAccessActionDetail(c, "保存管理员账号级静默时段策略")
	if before != nil {
		if payload, err := json.Marshal(map[string]any{
			"enabled":            map[string]any{"before": before.Policy.Enabled, "after": req.Enabled},
			"timezone":           map[string]any{"before": before.Policy.TimeZone, "after": req.TimeZone},
			"start":              map[string]any{"before": before.Policy.Start, "after": req.Start},
			"end":                map[string]any{"before": before.Policy.End, "after": req.End},
			"block_register":     map[string]any{"before": before.Policy.BlockRegister, "after": req.BlockRegister},
			"block_send_message": map[string]any{"before": before.Policy.BlockSendMessage, "after": req.BlockSendMessage},
			"block_create_group": map[string]any{"before": before.Policy.BlockCreateGroup, "after": req.BlockCreateGroup},
			"block_add_friend":   map[string]any{"before": before.Policy.BlockAddFriend, "after": req.BlockAddFriend},
		}); err == nil {
			middleware.SetAdminAccessChangeSet(c, string(payload))
		}
	}
	response.OK(c, nil)
}

// ClearManagerChatQuietHoursPolicy 清除管理员账号级静默时段策略。
func ClearManagerChatQuietHoursPolicy(c *gin.Context) {
	managerID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	var req clearAccountChatQuietHoursPolicyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	if err := controller_admin.ClearManagerChatQuietHoursPolicy(c.GetUint("admin_id"), managerID, req.CurrentPassword); err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "admin_user", &managerID, "", "管理员静默时段策略")
	middleware.SetAdminAccessAction(c, "config.clear_manager_quiet_hours", "清除管理员静默时段策略")
	middleware.SetAdminAccessActionDetail(c, "清除管理员账号级静默时段策略")
	response.OK(c, nil)
}

// GetAssistantChatQuietHoursPolicy 获取助理账号级静默时段策略。
func GetAssistantChatQuietHoursPolicy(c *gin.Context) {
	assistantID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	result, err := controller_admin.GetAssistantChatQuietHoursPolicy(c.GetUint("admin_id"), assistantID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// SaveAssistantChatQuietHoursPolicy 保存助理账号级静默时段策略。
func SaveAssistantChatQuietHoursPolicy(c *gin.Context) {
	assistantID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	var req saveAccountChatQuietHoursPolicyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	before, _ := controller_admin.GetAssistantChatQuietHoursPolicy(c.GetUint("admin_id"), assistantID)
	if err := controller_admin.SaveAssistantChatQuietHoursPolicy(controller_admin.SaveAccountChatQuietHoursPolicyReq{
		ActorID:           c.GetUint("admin_id"),
		TargetAdminUserID: assistantID,
		CurrentPassword:   req.CurrentPassword,
		Policy: config.ChatQuietHoursPolicy{
			Enabled:          req.Enabled,
			TimeZone:         req.TimeZone,
			Start:            req.Start,
			End:              req.End,
			BlockRegister:    req.BlockRegister,
			BlockSendMessage: req.BlockSendMessage,
			BlockCreateGroup: req.BlockCreateGroup,
			BlockAddFriend:   req.BlockAddFriend,
		},
	}); err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "assistant", &assistantID, "", "助理静默时段策略")
	middleware.SetAdminAccessAction(c, "config.save_assistant_quiet_hours", "保存助理静默时段策略")
	middleware.SetAdminAccessActionDetail(c, "保存助理账号级静默时段策略")
	if before != nil {
		if payload, err := json.Marshal(map[string]any{
			"enabled":            map[string]any{"before": before.Policy.Enabled, "after": req.Enabled},
			"timezone":           map[string]any{"before": before.Policy.TimeZone, "after": req.TimeZone},
			"start":              map[string]any{"before": before.Policy.Start, "after": req.Start},
			"end":                map[string]any{"before": before.Policy.End, "after": req.End},
			"block_register":     map[string]any{"before": before.Policy.BlockRegister, "after": req.BlockRegister},
			"block_send_message": map[string]any{"before": before.Policy.BlockSendMessage, "after": req.BlockSendMessage},
			"block_create_group": map[string]any{"before": before.Policy.BlockCreateGroup, "after": req.BlockCreateGroup},
			"block_add_friend":   map[string]any{"before": before.Policy.BlockAddFriend, "after": req.BlockAddFriend},
		}); err == nil {
			middleware.SetAdminAccessChangeSet(c, string(payload))
		}
	}
	response.OK(c, nil)
}

// ClearAssistantChatQuietHoursPolicy 清除助理账号级静默时段策略。
func ClearAssistantChatQuietHoursPolicy(c *gin.Context) {
	assistantID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	var req clearAccountChatQuietHoursPolicyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	if err := controller_admin.ClearAssistantChatQuietHoursPolicy(c.GetUint("admin_id"), assistantID, req.CurrentPassword); err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "assistant", &assistantID, "", "助理静默时段策略")
	middleware.SetAdminAccessAction(c, "config.clear_assistant_quiet_hours", "清除助理静默时段策略")
	middleware.SetAdminAccessActionDetail(c, "清除助理账号级静默时段策略")
	response.OK(c, nil)
}
