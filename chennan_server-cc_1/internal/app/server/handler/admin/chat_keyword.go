package handler_admin

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

func setChatKeywordAuditChangeSet(c *gin.Context, changeSet map[string]any) {
	if len(changeSet) == 0 {
		return
	}
	if payload, err := json.Marshal(changeSet); err == nil {
		middleware.SetAdminAccessChangeSet(c, string(payload))
	}
}

type saveChatKeywordReq struct {
	Keyword     string `json:"keyword" binding:"required"`
	Replacement string `json:"replacement"`
	Enabled     *bool  `json:"enabled"`
}

type importChatKeywordsReq struct {
	Content string `json:"content" binding:"required"`
	Enabled *bool  `json:"enabled"`
}

// ListChatKeywords 获取聊天关键词规则列表。
func ListChatKeywords(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	page, pageSize := parsePagination(c)
	result, err := controller_admin.ListChatKeywords(c.GetUint("admin_id"), controller_admin.ListChatKeywordsReq{
		Page:     page,
		PageSize: pageSize,
		Keyword:  strings.TrimSpace(c.Query("keyword")),
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.OK(c, result)
}

// CreateChatKeyword 创建聊天关键词规则。
func CreateChatKeyword(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	var req saveChatKeywordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	result, err := controller_admin.CreateChatKeyword(c.GetUint("admin_id"), controller_admin.SaveChatKeywordReq{
		Keyword:     req.Keyword,
		Replacement: req.Replacement,
		Enabled:     enabled,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	middleware.SetAdminAccessTarget(c, "system_config", &result.ID, result.Keyword, result.Keyword)
	middleware.SetAdminAccessAction(c, "chat_keyword.create", "创建聊天关键词")
	setChatKeywordAuditChangeSet(c, map[string]any{
		"keyword":     map[string]any{"before": "", "after": result.Keyword},
		"replacement": map[string]any{"before": "", "after": result.Replacement},
		"enabled":     map[string]any{"before": false, "after": result.Enabled},
	})
	response.OK(c, result)
}

// UpdateChatKeyword 更新聊天关键词规则。
func UpdateChatKeyword(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	id, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	var req saveChatKeywordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	result, err := controller_admin.UpdateChatKeyword(c.GetUint("admin_id"), id, controller_admin.SaveChatKeywordReq{
		Keyword:     req.Keyword,
		Replacement: req.Replacement,
		Enabled:     enabled,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	middleware.SetAdminAccessTarget(c, "system_config", &id, req.Keyword, req.Keyword)
	middleware.SetAdminAccessAction(c, "chat_keyword.update", "更新聊天关键词")
	setChatKeywordAuditChangeSet(c, map[string]any{
		"keyword":     map[string]any{"before": "已更新前值", "after": result.Keyword},
		"replacement": map[string]any{"before": "已更新前值", "after": result.Replacement},
		"enabled":     map[string]any{"before": "已更新前值", "after": result.Enabled},
	})
	response.OK(c, result)
}

// DeleteChatKeyword 删除聊天关键词规则。
func DeleteChatKeyword(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	id, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	if err := controller_admin.DeleteChatKeyword(c.GetUint("admin_id"), id); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	middleware.SetAdminAccessTarget(c, "system_config", &id, "", "聊天关键词")
	middleware.SetAdminAccessAction(c, "chat_keyword.delete", "删除聊天关键词")
	setChatKeywordAuditChangeSet(c, map[string]any{
		"status": map[string]any{"before": "active", "after": "deleted"},
	})
	response.OK(c, nil)
}

// BatchImportChatKeywords 批量导入聊天关键词规则。
func BatchImportChatKeywords(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	var req importChatKeywordsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	result, err := controller_admin.BatchImportChatKeywords(c.GetUint("admin_id"), controller_admin.BatchImportChatKeywordsReq{
		Content: req.Content,
		Enabled: enabled,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	middleware.SetAdminAccessTarget(c, "system_config", nil, "chat_keywords", "聊天关键词库")
	middleware.SetAdminAccessAction(c, "chat_keyword.batch_import", "批量导入聊天关键词")
	setChatKeywordAuditChangeSet(c, map[string]any{
		"content_lines": map[string]any{"before": 0, "after": strings.Count(req.Content, "\n") + 1},
		"enabled":       map[string]any{"before": false, "after": enabled},
	})
	response.OK(c, result)
}
