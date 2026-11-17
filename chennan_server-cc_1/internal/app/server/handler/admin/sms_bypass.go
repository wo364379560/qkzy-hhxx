package handler_admin

import (
	"encoding/json"
	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/handler/authlimit"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type revealSMSBypassReq struct {
	CurrentPassword string `json:"current_password" binding:"required"`
}

type setSMSBypassReq struct {
	Enabled         bool    `json:"enabled"`
	Code            *string `json:"code"`
	CurrentPassword string  `json:"current_password" binding:"required"`
}

// GetSMSBypassSettings 获取万能验证码配置。
func GetSMSBypassSettings(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	result, err := controller_admin.GetSMSBypassSettings()
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// RevealSMSBypassCode 查看万能验证码明文。
func RevealSMSBypassCode(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	var req revealSMSBypassReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	code, err := controller_admin.RevealSMSBypassCode(c.GetUint("admin_id"), req.CurrentPassword)
	if err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "system_config", nil, "sms_bypass", "万能验证码")
	middleware.SetAdminAccessAction(c, "config.reveal_sms_bypass", "查看万能验证码")
	middleware.SetAdminAccessActionDetail(c, "查看万能验证码明文")
	response.OK(c, map[string]string{"code": code})
}

// SetSMSBypassSettings 设置万能验证码配置。
func SetSMSBypassSettings(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	var req setSMSBypassReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	before, _ := controller_admin.GetSMSBypassSettings()

	if err := controller_admin.SetSMSBypassSettings(c.GetUint("admin_id"), req.CurrentPassword, req.Enabled, req.Code); err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "system_config", nil, "sms_bypass", "万能验证码")
	middleware.SetAdminAccessAction(c, "config.update_sms_bypass", "修改万能验证码")
	middleware.SetAdminAccessActionDetail(c, "修改万能验证码配置")
	if payload, err := json.Marshal(map[string]any{
		"enabled": map[string]any{
			"before": before.Enabled,
			"after":  req.Enabled,
		},
		"has_code": map[string]any{
			"before": before.HasCode,
			"after":  req.Code != nil && *req.Code != "",
		},
	}); err == nil {
		middleware.SetAdminAccessChangeSet(c, string(payload))
	}
	response.OK(c, nil)
}
