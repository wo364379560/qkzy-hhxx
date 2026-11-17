package handler_admin

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ixayda/ix-Go/internal/app/server/alert"
	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/handler/authlimit"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/internal/pkg/clientip"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type bindTOTPReq struct {
	Code string `json:"code" binding:"required,len=6"`
}

// BindTOTP 完成 TOTP 绑定。
func BindTOTP(c *gin.Context) {
	var req bindTOTPReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	req.Code = strings.TrimSpace(req.Code)

	token := c.GetHeader("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	result, err := controller_admin.BindTOTPWithAudit(token, req.Code, clientip.FromContext(c), c.Request.UserAgent())
	if err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		var authErr controller_admin.AuthErr
		if errors.As(err, &authErr) {
			alert.AdminLogin(c.Request.Context(), "", "totp_setup", "失败", authErr.Error(), clientip.FromContext(c), c.Request.UserAgent())
			response.BadRequest(c, authErr.Error())
			return
		}
		_ = c.Error(err)
		alert.AdminLogin(c.Request.Context(), "", "totp_setup", "失败", "系统错误", clientip.FromContext(c), c.Request.UserAgent())
		response.InternalError(c)
		return
	}
	if result.User != nil {
		alert.AdminLogin(c.Request.Context(), result.User.Username, "totp_setup", "成功", "", clientip.FromContext(c), c.Request.UserAgent())
	}

	response.OK(c, result)
}

type verifyTOTPReq struct {
	Code       string `json:"code"        binding:"omitempty,len=6"`
	BackupCode string `json:"backup_code" binding:"omitempty,max=32"`
}

// VerifyTOTP 验证 TOTP 码。
func VerifyTOTP(c *gin.Context) {
	var req verifyTOTPReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	req.BackupCode = strings.TrimSpace(req.BackupCode)
	if req.Code == "" && req.BackupCode == "" {
		response.InvalidParams(c, "验证码或备份码不能为空")
		return
	}

	token := c.GetHeader("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	result, err := controller_admin.VerifyTOTPWithAudit(token, req.Code, req.BackupCode, clientip.FromContext(c), c.Request.UserAgent())
	if err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		var authErr controller_admin.AuthErr
		if errors.As(err, &authErr) {
			alert.AdminLogin(c.Request.Context(), "", "totp_verify", "失败", authErr.Error(), clientip.FromContext(c), c.Request.UserAgent())
			response.BadRequest(c, authErr.Error())
			return
		}
		_ = c.Error(err)
		alert.AdminLogin(c.Request.Context(), "", "totp_verify", "失败", "系统错误", clientip.FromContext(c), c.Request.UserAgent())
		response.InternalError(c)
		return
	}
	if result.User != nil {
		alert.AdminLogin(c.Request.Context(), result.User.Username, "totp_verify", "成功", "", clientip.FromContext(c), c.Request.UserAgent())
	}

	response.OK(c, result)
}

// GetBackupCodes 获取备份码列表。
func GetBackupCodes(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	result, err := controller_admin.GetBackupCodes(adminID)
	if err != nil {
		_ = c.Error(err)
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

type regenerateBackupCodesReq struct {
	CurrentPassword string `json:"current_password" binding:"required,max=128"`
}

// RegenerateBackupCodes 重新生成备份码。
func RegenerateBackupCodes(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var req regenerateBackupCodesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	result, err := controller_admin.RegenerateBackupCodes(adminID, req.CurrentPassword)
	if err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		var authErr controller_admin.AuthErr
		if errors.As(err, &authErr) {
			response.BadRequest(c, authErr.Error())
			return
		}
		_ = c.Error(err)
		response.InternalError(c)
		return
	}
	middleware.SetAdminAccessTarget(c, "admin_user", &adminID, "", "TOTP备份码")
	middleware.SetAdminAccessAction(c, "admin_user.regenerate_backup_codes", "重置备份码")
	middleware.SetAdminAccessChangeSet(c, `{"backup_codes":{"before":"old","after":"new"}}`)
	response.OK(c, result)
}
