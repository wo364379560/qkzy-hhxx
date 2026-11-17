package handler_admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

func ListSMSSendLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	result, err := controller_admin.ListSMSSendLogs(
		page,
		pageSize,
		c.Query("keyword"),
		c.Query("scene"),
		c.Query("status"),
	)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}
