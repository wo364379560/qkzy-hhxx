package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ixayda/ix-Go/internal/app/server/migrationfence"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

const sourceMigrationMaintenanceMessage = "系统迁移维护中，请稍后重试"

var sourceMigrationAllowedRequests = map[string]struct{}{
	http.MethodGet + " /api/v1/chat/ping":                    {},
	http.MethodGet + " /api/v1/public/bootstrap/domain-pool": {},
	http.MethodPost + " /api/v1/webhook/tencent":             {},
}

// SourceMigrationFence blocks source-side business access while preserving
// health probes, bootstrap discovery, and Tencent before-event callbacks.
func SourceMigrationFence() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !migrationfence.Active() {
			c.Next()
			return
		}
		if _, allowed := sourceMigrationAllowedRequests[c.Request.Method+" "+c.Request.URL.Path]; allowed {
			c.Next()
			return
		}
		c.Header("Retry-After", "60")
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, response.Response{
			Code:    http.StatusServiceUnavailable,
			Message: sourceMigrationMaintenanceMessage,
		})
	}
}
