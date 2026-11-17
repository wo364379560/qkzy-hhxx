package handler_chat

import (
	"errors"

	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type revokeDeviceReq struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// ListDevices 获取当前用户设备列表。
func ListDevices(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	if imUserID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	result, err := controller_chat.ListDevices(imUserID, c.Query("current_device_id"))
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// RevokeDeviceByPath 按路径参数撤销设备。
func RevokeDeviceByPath(c *gin.Context) {
	revokeDevice(c, c.Param("device_id"))
}

// RevokeDeviceByBody 按请求体撤销设备。
func RevokeDeviceByBody(c *gin.Context) {
	var req revokeDeviceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	revokeDevice(c, req.DeviceID)
}

func revokeDevice(c *gin.Context, deviceID string) {
	imUserID := c.GetString("im_user_id")
	if imUserID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	if err := controller_chat.RevokeDevice(imUserID, deviceID); err != nil {
		var ae controller_chat.AuthErr
		if errors.As(err, &ae) {
			response.BadRequest(c, ae.Error())
			return
		}
		response.InternalError(c)
		return
	}
	response.OK(c, nil)
}
