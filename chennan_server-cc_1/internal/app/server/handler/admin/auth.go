package handler_admin

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ixayda/ix-Go/internal/app/server/alert"
	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/handler/authlimit"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/internal/app/server/ratelimit"
	"github.com/ixayda/ix-Go/internal/pkg/clientip"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type loginReq struct {
	Username       string `json:"username"        binding:"required,max=64"`
	Password       string `json:"password"        binding:"required,max=128"`
	TurnstileToken string `json:"turnstile_token" binding:"omitempty,max=4096"`
}

type changePasswordReq struct {
	OldPassword string `json:"oldPassword" binding:"required,max=128"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=128"`
}

// Login Admin 登录。
func Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		response.InvalidParams(c, "用户名不能为空")
		return
	}
	clientIP := clientip.FromContext(c)
	if !authlimit.Enforce(c, "admin_login", []ratelimit.SubjectRules{
		{Subject: ratelimit.HashSubject("account", strings.ToLower(req.Username)), Rules: ratelimit.PublicAdminLoginAccountRules},
	}) {
		return
	}

	result, err := controller_admin.Login(
		c.Request.Context(),
		req.Username,
		req.Password,
		req.TurnstileToken,
		clientIP,
		c.Request.UserAgent(),
	)
	if err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		var authErr controller_admin.AuthErr
		if errors.As(err, &authErr) {
			alert.AdminLogin(c.Request.Context(), req.Username, "password", "失败", authErr.Error(), clientIP, c.Request.UserAgent())
			response.Unauthorized(c, authErr.Error())
			return
		}
		_ = c.Error(err)
		alert.AdminLogin(c.Request.Context(), req.Username, "password", "失败", "系统错误", clientIP, c.Request.UserAgent())
		response.InternalError(c)
		return
	}
	switch {
	case result.RequireTOTPSetup:
		alert.AdminLogin(c.Request.Context(), req.Username, "password", "需要绑定2FA", "", clientIP, c.Request.UserAgent())
	case result.RequireTOTPVerify:
		alert.AdminLogin(c.Request.Context(), req.Username, "password", "需要2FA验证", "", clientIP, c.Request.UserAgent())
	default:
		alert.AdminLogin(c.Request.Context(), req.Username, "password", "成功", "", clientIP, c.Request.UserAgent())
	}

	response.OK(c, result)
}

// Logout Admin 登出（客户端丢弃 token 即可，服务端无状态）。
func Logout(c *gin.Context) {
	response.OK(c, nil)
}

// ChangePassword 修改当前后台账号密码。
func ChangePassword(c *gin.Context) {
	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	adminID := c.GetUint("admin_id")
	if err := controller_admin.ChangePassword(adminID, req.OldPassword, req.NewPassword); err != nil {
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

	middleware.SetAdminAccessTarget(c, "admin_user", &adminID, "", "后台账号密码")
	middleware.SetAdminAccessAction(c, "admin_user.change_password", "修改后台密码")
	middleware.SetAdminAccessChangeSet(c, `{"password":{"before":"old","after":"new"}}`)
	response.OK(c, nil)
}
