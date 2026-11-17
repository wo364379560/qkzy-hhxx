package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/security"
	"github.com/ixayda/ix-Go/pkg/base/response"

	pkgjwt "github.com/ixayda/ix-Go/pkg/jwt"
)

// AuthChat 是 Flutter App JWT 鉴权中间件。
// 从 Authorization: Bearer <token> 头解析用户身份，
// 将 im_user_id 注入 gin.Context 供后续 handler 使用。
func AuthChat() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "未登录")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Unauthorized(c, "Authorization 格式错误")
			c.Abort()
			return
		}

		claims, err := pkgjwt.ParseUser(parts[1])
		if err != nil {
			response.Unauthorized(c, "token 无效或已过期")
			c.Abort()
			return
		}

		userRepo := repository.NewUserRepository()
		user, err := userRepo.FindByIMUserID(claims.IMUserID)
		if err != nil {
			response.Unauthorized(c, "账号不存在或已失效")
			c.Abort()
			return
		}
		if user.Status == "banned" || user.Status == "deleted" {
			response.Unauthorized(c, "账号已被禁用")
			c.Abort()
			return
		}
		if !security.MatchesAuthVersion(claims.AuthVersion, user.AuthVersion) {
			response.Unauthorized(c, "登录状态已失效，请重新登录")
			c.Abort()
			return
		}
		if user.OwnerID > 0 {
			adminRepo := repository.NewAdminUserRepository()
			owner, err := adminRepo.FindByID(user.OwnerID)
			if err != nil {
				response.Unauthorized(c, "账号归属不可用，请联系管理员")
				c.Abort()
				return
			}
			if owner.Status != "active" {
				response.Unauthorized(c, "账号归属已被禁用，请联系管理员")
				c.Abort()
				return
			}
		}

		c.Set("im_user_id", claims.IMUserID)
		c.Next()
	}
}
