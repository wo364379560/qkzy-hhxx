package handler_chat

import (
	"errors"

	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type updateSecuritySettingsReq struct {
	RemoteLoginAlert  *bool  `json:"remote_login_alert"`
	LoginVerification *bool  `json:"login_verification"`
	DeviceID          string `json:"device_id"`
	DeviceName        string `json:"device_name"`
	Platform          string `json:"platform"`
}

// GetSecuritySettings 获取账号保护设置。
func GetSecuritySettings(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	if imUserID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	result, err := controller_chat.GetSecuritySettings(imUserID)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// UpdateSecuritySettings 更新账号保护设置。
func UpdateSecuritySettings(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	if imUserID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	var req updateSecuritySettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	result, err := controller_chat.UpdateSecuritySettings(imUserID, controller_chat.UpdateSecuritySettingsReq{
		RemoteLoginAlert:  req.RemoteLoginAlert,
		LoginVerification: req.LoginVerification,
		DeviceID:          req.DeviceID,
		DeviceName:        req.DeviceName,
		Platform:          req.Platform,
	})
	if err != nil {
		var ae controller_chat.AuthErr
		if errors.As(err, &ae) {
			response.BadRequest(c, ae.Error())
			return
		}
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}
