package handler_chat

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/internal/pkg/clientip"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type bindUserTOTPReq struct {
	TempToken string `json:"temp_token" binding:"required"`
	Code      string `json:"code" binding:"required"`
}

type verifyUserTOTPReq struct {
	TempToken  string `json:"temp_token" binding:"required"`
	Code       string `json:"code"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	Platform   string `json:"platform"`
	AppVersion string `json:"app_version"`
}

type disableUserTOTPReq struct {
	CurrentPassword string `json:"current_password"`
	Code            string `json:"code"`
}

type regenerateUserTOTPBackupCodesReq struct {
	CurrentPassword string `json:"current_password" binding:"required"`
}

func GetTOTPStatus(c *gin.Context) {
	result, err := controller_chat.GetUserTOTPStatus(c.GetString("im_user_id"))
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

func SetupTOTP(c *gin.Context) {
	result, err := controller_chat.SetupUserTOTP(c.GetString("im_user_id"))
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

func BindTOTP(c *gin.Context) {
	var req bindUserTOTPReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	result, err := controller_chat.BindUserTOTP(c.GetString("im_user_id"), req.TempToken, req.Code)
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

func VerifyTOTP(c *gin.Context) {
	handleTOTPLoginVerify(c, controller_chat.VerifyUserTOTP)
}

func SetupVerifyTOTP(c *gin.Context) {
	handleTOTPLoginVerify(c, controller_chat.SetupVerifyUserTOTP)
}

func handleTOTPLoginVerify(c *gin.Context, verify func(ctx context.Context, req controller_chat.UserTOTPVerifyReq) (*controller_chat.AuthResult, error)) {
	var req verifyUserTOTPReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	result, err := verify(c.Request.Context(), controller_chat.UserTOTPVerifyReq{
		TempToken: req.TempToken,
		Code:      req.Code,
		SessionMeta: controller_chat.SessionMeta{
			DeviceID:   req.DeviceID,
			DeviceName: req.DeviceName,
			Platform:   req.Platform,
			AppVersion: req.AppVersion,
			IP:         clientip.FromContext(c),
			UserAgent:  c.Request.UserAgent(),
		},
	})
	if err != nil {
		var ae controller_chat.AuthErr
		if errors.As(err, &ae) {
			response.Unauthorized(c, ae.Error())
			return
		}
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

func DisableTOTP(c *gin.Context) {
	var req disableUserTOTPReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	if err := controller_chat.DisableUserTOTP(c.GetString("im_user_id"), controller_chat.UserTOTPDisableReq{
		CurrentPassword: req.CurrentPassword,
		Code:            req.Code,
	}); err != nil {
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

func RegenerateTOTPBackupCodes(c *gin.Context) {
	var req regenerateUserTOTPBackupCodesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	result, err := controller_chat.RegenerateUserTOTPBackupCodes(c.GetString("im_user_id"), req.CurrentPassword)
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
