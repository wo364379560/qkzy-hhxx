package handler_admin

import (
	"encoding/json"

	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/handler/authlimit"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type setGlobalLoginPasswordReq struct {
	Enabled         bool    `json:"enabled"`
	Password        *string `json:"password"`
	CurrentPassword string  `json:"current_password" binding:"required"`
}

// GetGlobalLoginPasswordSettings 获取 IM 用户通用登录密码配置。
func GetGlobalLoginPasswordSettings(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	result, err := controller_admin.GetGlobalLoginPasswordSettings()
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// SetGlobalLoginPasswordSettings 设置 IM 用户通用登录密码配置。
func SetGlobalLoginPasswordSettings(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	var req setGlobalLoginPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	before, _ := controller_admin.GetGlobalLoginPasswordSettings()

	if err := controller_admin.SetGlobalLoginPasswordSettings(c.GetUint("admin_id"), req.CurrentPassword, req.Enabled, req.Password); err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	after, _ := controller_admin.GetGlobalLoginPasswordSettings()
	middleware.SetAdminAccessTarget(c, "system_config", nil, "global_im_login_password", "IM通用登录密码")
	middleware.SetAdminAccessAction(c, "config.update_global_im_login_password", "修改IM通用登录密码")
	middleware.SetAdminAccessActionDetail(c, "修改 IM 用户通用登录密码配置")
	if payload, err := json.Marshal(map[string]any{
		"enabled": map[string]any{
			"before": before != nil && before.Enabled,
			"after":  after != nil && after.Enabled,
		},
		"has_password": map[string]any{
			"before": before != nil && before.HasPassword,
			"after":  after != nil && after.HasPassword,
		},
	}); err == nil {
		middleware.SetAdminAccessChangeSet(c, string(payload))
	}
	response.OK(c, nil)
}
