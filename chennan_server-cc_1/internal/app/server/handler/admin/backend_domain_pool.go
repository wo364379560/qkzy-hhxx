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

type setBackendDomainPoolReq struct {
	TTLSeconds      int      `json:"ttl_seconds"`
	APIOrigins      []string `json:"api_origins"`
	CurrentPassword string   `json:"current_password" binding:"required"`
}

// GetBackendDomainPoolSettings 获取业务域名池配置。
func GetBackendDomainPoolSettings(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	settings, err := controller_admin.GetBackendDomainPoolSettings()
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, settings)
}

// SetBackendDomainPoolSettings 设置业务域名池配置。
func SetBackendDomainPoolSettings(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	var req setBackendDomainPoolReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	before, _ := controller_admin.GetBackendDomainPoolSettings()

	settings, err := controller_admin.SetBackendDomainPoolSettings(
		c.GetUint("admin_id"),
		req.CurrentPassword,
		config.BackendDomainPoolSettings{
			TTLSeconds: req.TTLSeconds,
			APIOrigins: req.APIOrigins,
		},
	)
	if err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	middleware.SetAdminAccessTarget(c, "system_config", nil, "backend_domain_pool", "后台域名池")
	middleware.SetAdminAccessAction(c, "config.update_backend_domain_pool", "修改后台域名池")
	middleware.SetAdminAccessActionDetail(c, "修改后台域名池配置")
	if payload, err := json.Marshal(map[string]any{
		"ttl_seconds": map[string]any{
			"before": before.TTLSeconds,
			"after":  settings.TTLSeconds,
		},
		"api_origins": map[string]any{
			"before": before.APIOrigins,
			"after":  settings.APIOrigins,
		},
	}); err == nil {
		middleware.SetAdminAccessChangeSet(c, string(payload))
	}
	response.OK(c, settings)
}
