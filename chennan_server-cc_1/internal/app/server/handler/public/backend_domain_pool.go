package handler_public

import (
	"github.com/gin-gonic/gin"

	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

const backendDomainPoolCacheControl = "no-store"

// GetBackendDomainPoolBootstrap 公开下发 App 业务域名池配置。
// GET /api/v1/public/bootstrap/domain-pool
func GetBackendDomainPoolBootstrap(c *gin.Context) {
	c.Header("Cache-Control", backendDomainPoolCacheControl)
	settings, err := config.GetBackendDomainPoolSettings(
		c.Request.Context(),
		datasource.Datasource.Mysql.GetClient(),
	)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, settings)
}
