package middleware

import "github.com/gin-gonic/gin"

// APINoStore applies to business APIs and bootstrap responses, including errors.
// The domain-pool TTL is an application refresh interval, not a CDN cache TTL.
func APINoStore() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Header("Pragma", "no-cache")
		c.Next()
	}
}
