package handler_admin

import (
	"fmt"
	"github.com/gin-gonic/gin"
	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

func GetIMApplication(c *gin.Context) {
	response.OK(c, gin.H{"sdk_app_id": config.Cfg.Get().TencentIM.SDKAppID})
}
func RepairUserIMAccount(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "用户编号无效")
		return
	}
	var req struct {
		SDKAppID int `json:"expected_sdk_app_id" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		response.BadRequest(c, "请先核对聊天应用SDKAppID")
		return
	}
	result, err := controller_admin.RepairUserIMAccount(c.GetUint("admin_id"), id, req.SDKAppID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("同步IM账号；user_id=%d；sdk_app_id=%d", id, req.SDKAppID))
	response.OK(c, result)
}
