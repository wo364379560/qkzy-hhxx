package handler_admin

import (
	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

// GetStats 仪表盘统计数据。
func GetStats(c *gin.Context) {
	role := c.GetString("admin_role")
	adminID := c.GetUint("admin_id")

	if controller_admin.IsAssistantRole(role) {
		result, err := controller_admin.GetAssistantStats(adminID)
		if err != nil {
			response.InternalError(c)
			return
		}
		response.OK(c, result)
		return
	}

	result, err := controller_admin.GetManagerStats(adminID)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}
