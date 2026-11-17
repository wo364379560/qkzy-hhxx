package middleware

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/clientip"
)

const ctxAdminAccessFailureReason = "admin_access_failure_reason"
const ctxAdminAccessActionDetail = "admin_access_action_detail"
const ctxAdminAccessTargetType = "admin_access_target_type"
const ctxAdminAccessTargetID = "admin_access_target_id"
const ctxAdminAccessTargetUsername = "admin_access_target_username"
const ctxAdminAccessTargetDisplayName = "admin_access_target_display_name"
const ctxAdminAccessActionKey = "admin_access_action_key"
const ctxAdminAccessActionSummary = "admin_access_action_summary"
const ctxAdminAccessChangeSet = "admin_access_change_set"

// SetAdminAccessFailureReason 记录后台访问失败原因，供审计中间件写入。
func SetAdminAccessFailureReason(c *gin.Context, reason string) {
	c.Set(ctxAdminAccessFailureReason, reason)
}

// SetAdminAccessActionDetail 记录后台动作详情，供审计中间件写入。
func SetAdminAccessActionDetail(c *gin.Context, detail string) {
	c.Set(ctxAdminAccessActionDetail, detail)
}

// SetAdminAccessTarget 记录后台操作目标信息，供审计中间件写入。
func SetAdminAccessTarget(c *gin.Context, targetType string, targetID *uint, targetUsername, targetDisplayName string) {
	c.Set(ctxAdminAccessTargetType, targetType)
	c.Set(ctxAdminAccessTargetID, targetID)
	c.Set(ctxAdminAccessTargetUsername, targetUsername)
	c.Set(ctxAdminAccessTargetDisplayName, targetDisplayName)
}

// SetAdminAccessAction 记录后台动作信息，供审计中间件写入。
func SetAdminAccessAction(c *gin.Context, actionKey, actionSummary string) {
	c.Set(ctxAdminAccessActionKey, actionKey)
	c.Set(ctxAdminAccessActionSummary, actionSummary)
}

// SetAdminAccessChangeSet 记录后台字段变更信息，供审计中间件写入。
func SetAdminAccessChangeSet(c *gin.Context, changeSet string) {
	c.Set(ctxAdminAccessChangeSet, changeSet)
}

// AdminAccessAudit 记录后台系统用户访问与非法访问。
func AdminAccessAudit() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		fullPath := c.FullPath()
		if shouldSkipAdminAccessAudit(fullPath) {
			return
		}

		statusCode := c.Writer.Status()
		adminID, hasAdminID := c.Get(CtxAdminID)
		adminUsername, _ := c.Get("admin_username")
		adminRole, _ := c.Get(CtxAdminRole)

		accessType := ""
		switch {
		case statusCode == 401 || statusCode == 403:
			accessType = "illegal_access"
		case hasAdminID:
			if !shouldAuditSystemUserAction(c.Request.Method, fullPath) {
				return
			}
			accessType = "system_user"
		default:
			return
		}

		var actorID *uint
		if hasAdminID {
			if id, ok := adminID.(uint); ok {
				actorID = &id
			}
		}

		var routeTargetID *uint
		if rawID := strings.TrimSpace(c.Param("id")); rawID != "" {
			if parsed, err := strconv.ParseUint(rawID, 10, 64); err == nil {
				id := uint(parsed)
				routeTargetID = &id
			}
		}

		failureReason, _ := c.Get(ctxAdminAccessFailureReason)
		actionDetail, _ := c.Get(ctxAdminAccessActionDetail)
		targetType, _ := c.Get(ctxAdminAccessTargetType)
		targetIDValue, _ := c.Get(ctxAdminAccessTargetID)
		targetUsername, _ := c.Get(ctxAdminAccessTargetUsername)
		targetDisplayName, _ := c.Get(ctxAdminAccessTargetDisplayName)
		actionKey, _ := c.Get(ctxAdminAccessActionKey)
		actionSummary, _ := c.Get(ctxAdminAccessActionSummary)
		changeSet, _ := c.Get(ctxAdminAccessChangeSet)

		targetID := routeTargetID
		if id, ok := targetIDValue.(*uint); ok && id != nil {
			targetID = id
		}

		_ = controller_admin.RecordAdminAccessAudit(&model.AdminAccessAudit{
			AdminUserID:       actorID,
			AdminUsername:     stringifyAuditValue(adminUsername),
			AdminRole:         stringifyAuditValue(adminRole),
			AccessType:        accessType,
			ActionKey:         stringifyAuditValue(actionKey),
			ActionSummary:     stringifyAuditValue(actionSummary),
			ActionDetail:      stringifyAuditValue(actionDetail),
			Method:            c.Request.Method,
			Route:             fullPath,
			TargetID:          targetID,
			TargetType:        stringifyAuditValue(targetType),
			TargetUsername:    stringifyAuditValue(targetUsername),
			TargetDisplayName: stringifyAuditValue(targetDisplayName),
			ChangeSetJSON:     stringifyAuditValue(changeSet),
			Success:           statusCode >= 200 && statusCode < 400,
			StatusCode:        statusCode,
			IPAddress:         clientip.FromContext(c),
			UserAgent:         c.Request.UserAgent(),
			FailureReason:     stringifyAuditValue(failureReason),
		})
	}
}

func shouldSkipAdminAccessAudit(fullPath string) bool {
	switch fullPath {
	case "/api/v1/admin/login", "/api/v1/admin/totp/bind", "/api/v1/admin/totp/verify":
		return true
	default:
		return fullPath == ""
	}
}

func shouldAuditSystemUserAction(method, fullPath string) bool {
	if method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" {
		return false
	}

	switch fullPath {
	case "/api/v1/admin/logout":
		return false
	case "/api/v1/admin/users/sync/preview":
		return false
	case "/api/v1/admin/groups/sync/preview":
		return false
	default:
		return true
	}
}

func stringifyAuditValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
