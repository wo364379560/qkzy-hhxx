package handler_admin

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

// ListAdminAccessAudits 获取后台访问审计列表。
func ListAdminAccessAudits(c *gin.Context) {
	page, pageSize := parsePagination(c)
	role := c.GetString("admin_role")
	adminID := c.GetUint("admin_id")

	scopeAdminIDs := []uint(nil)
	scopeUsernames := []string(nil)
	roleFilter := c.Query("role")
	if !controller_admin.IsSuperAdminRole(role) {
		actor, err := controller_admin.BuildActorContext(adminID)
		if err != nil {
			response.InternalError(c)
			return
		}
		scopeAdminIDs = actor.ScopeAdminIDs
		scopeUsernames, err = controller_admin.ResolveScopeAdminUsernames(scopeAdminIDs)
		if err != nil {
			response.InternalError(c)
			return
		}
	}
	result, err := controller_admin.ListAdminAccessAudits(controller_admin.ListAdminAccessAuditsReq{
		Page:           page,
		PageSize:       pageSize,
		Keyword:        c.Query("keyword"),
		Role:           roleFilter,
		AccessType:     c.Query("accessType"),
		Method:         c.Query("method"),
		TargetType:     c.Query("targetType"),
		ActionKey:      c.Query("actionKey"),
		TargetKeyword:  c.Query("targetKeyword"),
		ScopeAdminIDs:  scopeAdminIDs,
		ScopeUsernames: scopeUsernames,
	})
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// ExportAdminAccessAudits 导出后台访问审计 CSV。
func ExportAdminAccessAudits(c *gin.Context) {
	pageSize := 1000
	role := c.GetString("admin_role")
	adminID := c.GetUint("admin_id")

	scopeAdminIDs := []uint(nil)
	scopeUsernames := []string(nil)
	roleFilter := c.Query("role")
	if !controller_admin.IsSuperAdminRole(role) {
		actor, err := controller_admin.BuildActorContext(adminID)
		if err != nil {
			response.InternalError(c)
			return
		}
		scopeAdminIDs = actor.ScopeAdminIDs
		scopeUsernames, err = controller_admin.ResolveScopeAdminUsernames(scopeAdminIDs)
		if err != nil {
			response.InternalError(c)
			return
		}
	}

	result, err := controller_admin.ListAdminAccessAudits(controller_admin.ListAdminAccessAuditsReq{
		Page:           1,
		PageSize:       pageSize,
		Keyword:        c.Query("keyword"),
		Role:           roleFilter,
		AccessType:     c.Query("accessType"),
		Method:         c.Query("method"),
		TargetType:     c.Query("targetType"),
		ActionKey:      c.Query("actionKey"),
		TargetKeyword:  c.Query("targetKeyword"),
		ScopeAdminIDs:  scopeAdminIDs,
		ScopeUsernames: scopeUsernames,
	})
	if err != nil {
		response.InternalError(c)
		return
	}

	var buffer bytes.Buffer
	buffer.WriteString("\uFEFF")
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{
		"时间", "操作者", "角色", "访问类型", "模块", "动作", "动作码", "对象类型", "对象账号", "对象名称", "对象ID", "变更", "方法", "状态码", "结果", "IP", "路由",
	}); err != nil {
		response.InternalError(c)
		return
	}

	for _, row := range result.List {
		if err := writer.Write([]string{
			row.CreatedAt,
			row.AdminUsername,
			row.AdminRole,
			row.AccessType,
			row.Module,
			row.ActionSummary,
			row.ActionKey,
			row.TargetType,
			row.TargetUsername,
			row.TargetDisplayName,
			fmt.Sprintf("%v", row.TargetID),
			row.ChangeSet,
			row.Method,
			fmt.Sprintf("%d", row.StatusCode),
			fmt.Sprintf("%t", row.Success),
			row.IPAddress,
			row.Route,
		}); err != nil {
			response.InternalError(c)
			return
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		response.InternalError(c)
		return
	}

	filename := fmt.Sprintf("admin_access_audits_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buffer.Bytes())
}

// ListChatQuietHoursHitLogs 获取静默时段命中日志列表。
func ListChatQuietHoursHitLogs(c *gin.Context) {
	page, pageSize := parsePagination(c)
	role := c.GetString("admin_role")
	adminID := c.GetUint("admin_id")

	scopeOwnerIDs := []uint(nil)
	if !controller_admin.IsSuperAdminRole(role) {
		actor, err := controller_admin.BuildActorContext(adminID)
		if err != nil {
			response.InternalError(c)
			return
		}
		scopeOwnerIDs = actor.ScopeAdminIDs
	}

	result, err := controller_admin.ListChatQuietHoursHitLogs(controller_admin.ListChatQuietHoursHitLogsReq{
		Page:          page,
		PageSize:      pageSize,
		Keyword:       c.Query("keyword"),
		Action:        c.Query("action"),
		Source:        c.Query("source"),
		ScopeOwnerIDs: scopeOwnerIDs,
	})
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// ListAdminLoginAudits 获取后台登录审计列表。
func ListAdminLoginAudits(c *gin.Context) {
	page, pageSize := parsePagination(c)
	role := c.GetString("admin_role")
	adminID := c.GetUint("admin_id")

	scopeAdminIDs := []uint(nil)
	scopeUsernames := []string(nil)
	roleFilter := c.Query("role")
	if !controller_admin.IsSuperAdminRole(role) {
		actor, err := controller_admin.BuildActorContext(adminID)
		if err != nil {
			response.InternalError(c)
			return
		}
		scopeAdminIDs = actor.ScopeAdminIDs
		scopeUsernames, err = controller_admin.ResolveScopeAdminUsernames(scopeAdminIDs)
		if err != nil {
			response.InternalError(c)
			return
		}
	}
	result, err := controller_admin.ListAdminLoginAudits(controller_admin.ListAdminLoginAuditsReq{
		Page:           page,
		PageSize:       pageSize,
		Keyword:        c.Query("keyword"),
		Role:           roleFilter,
		Stage:          c.Query("stage"),
		Result:         c.Query("result"),
		ScopeAdminIDs:  scopeAdminIDs,
		ScopeUsernames: scopeUsernames,
	})
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

type adminAuditCleanupReq struct {
	Scope         string `json:"scope" binding:"required"`
	BeforeTime    string `json:"before_time" binding:"required"`
	Keyword       string `json:"keyword"`
	Role          string `json:"role"`
	AccessType    string `json:"accessType"`
	Method        string `json:"method"`
	TargetType    string `json:"targetType"`
	ActionKey     string `json:"actionKey"`
	TargetKeyword string `json:"targetKeyword"`
	Stage         string `json:"stage"`
	Result        string `json:"result"`
}

// PreviewAdminAuditCleanup 预览后台审计日志清理结果。
func PreviewAdminAuditCleanup(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		middleware.SetAdminAccessFailureReason(c, "仅超级管理员可操作")
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	var req adminAuditCleanupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.SetAdminAccessFailureReason(c, err.Error())
		response.InvalidParams(c, err.Error())
		return
	}

	result, err := controller_admin.PreviewAdminAuditCleanup(controller_admin.AuditCleanupReq{
		Scope:         req.Scope,
		BeforeTime:    req.BeforeTime,
		Keyword:       req.Keyword,
		Role:          req.Role,
		AccessType:    req.AccessType,
		Method:        req.Method,
		TargetType:    req.TargetType,
		ActionKey:     req.ActionKey,
		TargetKeyword: req.TargetKeyword,
		Stage:         req.Stage,
		Result:        req.Result,
	})
	if err != nil {
		middleware.SetAdminAccessFailureReason(c, err.Error())
		response.BadRequest(c, err.Error())
		return
	}

	middleware.SetAdminAccessActionDetail(
		c,
		fmt.Sprintf(
			"预览清理范围:%s；时间阈值:%s；访问审计:%d；登录审计:%d；合计:%d",
			result.Scope,
			result.BeforeTime,
			result.AccessCount,
			result.LoginCount,
			result.TotalCount,
		),
	)
	middleware.SetAdminAccessTarget(c, "system_config", nil, "audit_cleanup_preview", "审计清理预览")
	middleware.SetAdminAccessAction(c, "audit.cleanup.preview", "预览审计日志清理")
	response.OK(c, result)
}

// CleanupAdminAudits 清理后台审计日志。
func CleanupAdminAudits(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		middleware.SetAdminAccessFailureReason(c, "仅超级管理员可操作")
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	var req adminAuditCleanupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.SetAdminAccessFailureReason(c, err.Error())
		response.InvalidParams(c, err.Error())
		return
	}

	result, err := controller_admin.CleanupAdminAudits(controller_admin.AuditCleanupReq{
		Scope:         req.Scope,
		BeforeTime:    req.BeforeTime,
		Keyword:       req.Keyword,
		Role:          req.Role,
		AccessType:    req.AccessType,
		Method:        req.Method,
		TargetType:    req.TargetType,
		ActionKey:     req.ActionKey,
		TargetKeyword: req.TargetKeyword,
		Stage:         req.Stage,
		Result:        req.Result,
	})
	if err != nil {
		middleware.SetAdminAccessFailureReason(c, err.Error())
		response.BadRequest(c, err.Error())
		return
	}

	middleware.SetAdminAccessActionDetail(
		c,
		fmt.Sprintf(
			"清理范围:%s；时间阈值:%s；删除访问审计:%d；删除登录审计:%d；合计:%d",
			result.Scope,
			result.BeforeTime,
			result.AccessDeleted,
			result.LoginDeleted,
			result.TotalDeleted,
		),
	)
	middleware.SetAdminAccessTarget(c, "system_config", nil, "audit_cleanup", "审计日志清理")
	middleware.SetAdminAccessAction(c, "audit.cleanup.execute", "清理审计日志")
	response.OK(c, result)
}
