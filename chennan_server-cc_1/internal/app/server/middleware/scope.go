package middleware

import "github.com/gin-gonic/gin"

// Scope 是助理数据 scope 注入中间件（stub）。
// 为助理角色的请求注入 owner_id 过滤条件。
func Scope() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 从 JWT/session 中提取 admin_id，注入到 context
		c.Next()
	}
}
