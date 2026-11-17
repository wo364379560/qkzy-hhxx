package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/security"
	"github.com/ixayda/ix-Go/internal/pkg/clientip"
	"github.com/ixayda/ix-Go/pkg/base/response"
	pkgjwt "github.com/ixayda/ix-Go/pkg/jwt"
)

const (
	CtxAdminID   = "admin_id"
	CtxAdminRole = "admin_role"
)

// AuthAdmin 验证 Admin JWT，将 admin_id 和 role 注入 context。
func AuthAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			SetAdminAccessFailureReason(c, "缺少认证 token")
			response.Unauthorized(c, "缺少认证 token")
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := pkgjwt.Parse(tokenStr)
		if err != nil {
			SetAdminAccessFailureReason(c, "token 无效或已过期")
			response.Unauthorized(c, "token 无效或已过期")
			c.Abort()
			return
		}

		adminRepo := repository.NewAdminUserRepository()
		admin, err := adminRepo.FindByID(claims.AdminID)
		if err != nil {
			SetAdminAccessFailureReason(c, "账号不存在或已失效")
			response.Unauthorized(c, "账号不存在或已失效")
			c.Abort()
			return
		}
		if admin.Status != "active" {
			SetAdminAccessFailureReason(c, "账号已被禁用")
			response.Unauthorized(c, "账号已被禁用")
			c.Abort()
			return
		}
		if !security.MatchesAuthVersion(claims.AuthVersion, admin.AuthVersion) {
			SetAdminAccessFailureReason(c, "登录状态已失效，请重新登录")
			response.Unauthorized(c, "登录状态已失效，请重新登录")
			c.Abort()
			return
		}
		role := admin.Role
		c.Set("admin_username", admin.Username)
		if controller_admin.IsAssistantRole(role) || role == controller_admin.RoleAdmin {
			ruleRepo := repository.NewAdminUserIPWhitelistRepository()
			rules, err := ruleRepo.ListByAdminUserID(admin.ID)
			if err != nil {
				response.InternalError(c)
				c.Abort()
				return
			}

			clientIP := clientip.FromContext(c)
			allowed, err := security.EvaluateAdminUserIPWhitelist(admin, clientIP, rules)
			if err != nil {
				response.InternalError(c)
				c.Abort()
				return
			}
			if !allowed {
				SetAdminAccessFailureReason(c, fmt.Sprintf("您的账号已启用 IP 白名单限制，当前访问 IP (%s) 不在白名单内，请联系上级管理员添加", clientIP))
				response.Unauthorized(c, fmt.Sprintf("您的账号已启用 IP 白名单限制，当前访问 IP (%s) 不在白名单内，请联系上级管理员添加", clientIP))
				c.Abort()
				return
			}
		}

		c.Set(CtxAdminID, claims.AdminID)
		c.Set(CtxAdminRole, role)
		c.Next()
	}
}
