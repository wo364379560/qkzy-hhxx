package handler_admin

import (
	"encoding/json"

	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/handler/authlimit"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type setChatAttachmentUploadReq struct {
	Enabled         bool   `json:"enabled"`
	MaxSizeMB       int64  `json:"max_size_mb"`
	CurrentPassword string `json:"current_password" binding:"required"`
}

// GetChatAttachmentUploadSettings 获取聊天大文件上传配置。
func GetChatAttachmentUploadSettings(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	result, err := controller_admin.GetChatAttachmentUploadSettings()
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// SetChatAttachmentUploadSettings 设置聊天大文件上传配置。
func SetChatAttachmentUploadSettings(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	var req setChatAttachmentUploadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	before, _ := controller_admin.GetChatAttachmentUploadSettings()
	if err := controller_admin.SetChatAttachmentUploadSettings(
		c.GetUint("admin_id"),
		req.CurrentPassword,
		req.Enabled,
		req.MaxSizeMB,
	); err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	after, _ := controller_admin.GetChatAttachmentUploadSettings()
	middleware.SetAdminAccessTarget(c, "system_config", nil, "chat_attachment_upload", "聊天附件上传")
	middleware.SetAdminAccessAction(c, "config.update_chat_attachment_upload", "修改聊天附件上传")
	middleware.SetAdminAccessActionDetail(c, "修改聊天附件上传配置")
	if payload, err := json.Marshal(map[string]any{
		"enabled": map[string]any{
			"before": before != nil && before.Enabled,
			"after":  after != nil && after.Enabled,
		},
		"max_size_mb": map[string]any{
			"before": func() int64 {
				if before == nil {
					return 0
				}
				return before.MaxSizeMB
			}(),
			"after": func() int64 {
				if after == nil {
					return 0
				}
				return after.MaxSizeMB
			}(),
		},
	}); err == nil {
		middleware.SetAdminAccessChangeSet(c, string(payload))
	}
	response.OK(c, nil)
}
