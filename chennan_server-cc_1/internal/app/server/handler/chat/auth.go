package handler_chat

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/internal/app/server/handler/authlimit"
	"github.com/ixayda/ix-Go/internal/app/server/ratelimit"
	"github.com/ixayda/ix-Go/internal/pkg/clientip"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/turnstile"
	"github.com/ixayda/ix-Go/pkg/base/response"
	"github.com/ixayda/ix-Go/pkg/logger"
)

type sendSMSReq struct {
	Phone    string `json:"phone"     binding:"required,max=20"`
	DeviceID string `json:"device_id" binding:"omitempty,max=128"`
	Scene    string `json:"scene"     binding:"omitempty,max=32"`
}

// SendSMS 发送手机验证码。
func SendSMS(c *gin.Context) {
	var req sendSMSReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Phone == "" {
		response.InvalidParams(c, "手机号不能为空")
		return
	}
	clientIP := clientip.FromContext(c)
	if !authlimit.Enforce(c, "sms_send", []ratelimit.SubjectRules{
		{Subject: ratelimit.HashSubject("phone", req.Phone), Rules: ratelimit.PublicSMSPhoneRules},
		{Subject: ratelimit.HashSubject("device", req.DeviceID), Rules: ratelimit.PublicSMSDeviceRules},
	}) {
		return
	}
	if err := controller_chat.SendSMS(c.Request.Context(), controller_chat.SendSMSReq{
		Phone:    req.Phone,
		DeviceID: req.DeviceID,
		Scene:    req.Scene,
		IP:       clientIP,
	}); err != nil {
		var ae controller_chat.AuthErr
		if errors.As(err, &ae) {
			response.BadRequest(c, ae.Error())
		} else {
			c.Error(err)
			response.InternalError(c)
		}
		return
	}
	response.OK(c, nil)
}

type registerReq struct {
	Phone      string `json:"phone"       binding:"required,max=20"`
	Code       string `json:"code"        binding:"omitempty,max=10"`
	Nickname   string `json:"nickname"    binding:"omitempty,max=32"`
	Password   string `json:"password"    binding:"required,min=6,max=32"`
	InviteCode string `json:"invite_code" binding:"omitempty,max=32"`
	DeviceID   string `json:"device_id"   binding:"omitempty,max=128"`
	DeviceName string `json:"device_name" binding:"omitempty,max=128"`
	Platform   string `json:"platform"    binding:"omitempty,max=32"`
	AppVersion string `json:"app_version" binding:"omitempty,max=64"`
}

// Register 注册新用户。
func Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Phone == "" {
		response.InvalidParams(c, "手机号不能为空")
		return
	}
	clientIP := clientip.FromContext(c)
	if !authlimit.Enforce(c, "chat_register", []ratelimit.SubjectRules{
		{Subject: ratelimit.HashSubject("phone", req.Phone), Rules: ratelimit.PublicRegisterPhoneRules},
		{Subject: ratelimit.HashSubject("device", req.DeviceID), Rules: ratelimit.PublicRegisterDeviceRules},
	}) {
		return
	}
	result, err := controller_chat.Register(c.Request.Context(), controller_chat.RegisterReq{
		Phone:      req.Phone,
		Code:       req.Code,
		Nickname:   req.Nickname,
		Password:   req.Password,
		InviteCode: req.InviteCode,
		SessionMeta: controller_chat.SessionMeta{
			DeviceID:   req.DeviceID,
			DeviceName: req.DeviceName,
			Platform:   req.Platform,
			AppVersion: req.AppVersion,
			IP:         clientIP,
			UserAgent:  c.Request.UserAgent(),
		},
	})
	if err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		var ae controller_chat.AuthErr
		if errors.As(err, &ae) {
			response.BadRequest(c, ae.Error())
		} else {
			response.InternalError(c)
		}
		return
	}
	response.OK(c, result)
}

type loginReq struct {
	Phone      string `json:"phone"       binding:"omitempty,max=20"`
	Code       string `json:"code"        binding:"omitempty,max=10"` // 兼容旧请求，仅用于后端返回验证码登录已关闭
	Password   string `json:"password"    binding:"omitempty,max=128"`
	DeviceID   string `json:"device_id"   binding:"omitempty,max=128"`
	DeviceName string `json:"device_name" binding:"omitempty,max=128"`
	Platform   string `json:"platform"    binding:"omitempty,max=32"`
	AppVersion string `json:"app_version" binding:"omitempty,max=64"`

	TurnstileToken string `json:"turnstile_token" binding:"omitempty,max=4096"`
}

// Login 用户登录。
func Login(c *gin.Context) {
	if isWebLoginOrigin(c) {
		c.JSON(http.StatusUpgradeRequired, response.Response{
			Code:    http.StatusUpgradeRequired,
			Message: "Web 端登录已升级，请刷新页面后重试",
		})
		return
	}
	loginWithRequest(c, false)
}

// WebLogin Web 端用户登录，需通过 Turnstile 人机验证。
func WebLogin(c *gin.Context) {
	loginWithRequest(c, true)
}

func loginWithRequest(c *gin.Context, requireTurnstile bool) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Phone == "" {
		response.InvalidParams(c, "手机号不能为空")
		return
	}
	clientIP := clientip.FromContext(c)
	if !authlimit.Enforce(c, "chat_login", []ratelimit.SubjectRules{
		{Subject: ratelimit.HashSubject("account", req.Phone), Rules: ratelimit.PublicLoginAccountRules},
		{Subject: ratelimit.HashSubject("device", req.DeviceID), Rules: ratelimit.PublicLoginDeviceRules},
	}) {
		return
	}
	if requireTurnstile {
		req.Platform = "web"
		if err := turnstile.VerifyWithConfig(
			c.Request.Context(),
			config.Cfg.Get().WebTurnstile,
			req.TurnstileToken,
			clientIP,
		); err != nil {
			response.Unauthorized(c, err.Error())
			return
		}
	}
	result, err := controller_chat.Login(c.Request.Context(), controller_chat.LoginReq{
		Phone:    req.Phone,
		Code:     req.Code,
		Password: req.Password,
		SessionMeta: controller_chat.SessionMeta{
			DeviceID:   req.DeviceID,
			DeviceName: req.DeviceName,
			Platform:   req.Platform,
			AppVersion: req.AppVersion,
			IP:         clientIP,
			UserAgent:  c.Request.UserAgent(),
		},
	})
	if err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		var ae controller_chat.AuthErr
		if errors.As(err, &ae) {
			response.Unauthorized(c, ae.Error())
		} else {
			c.Error(err)
			response.InternalError(c)
		}
		return
	}
	response.OK(c, result)
}

func isWebLoginOrigin(c *gin.Context) bool {
	for _, expected := range webTurnstileExpectedHostnames() {
		if requestHeaderHostMatches(c.GetHeader("Origin"), expected) ||
			requestHeaderHostMatches(c.GetHeader("Referer"), expected) {
			return true
		}
	}
	return false
}

func webTurnstileExpectedHostnames() []string {
	cfg := config.Cfg.Get().WebTurnstile
	if len(cfg.ExpectedHostnames) > 0 {
		return cfg.ExpectedHostnames
	}
	if expected := strings.TrimSpace(cfg.ExpectedHostname); expected != "" {
		return []string{expected}
	}
	return []string{"web.wuyoubei.com"}
}

func requestHeaderHostMatches(rawValue, expectedHost string) bool {
	rawValue = strings.TrimSpace(rawValue)
	if rawValue == "" || expectedHost == "" {
		return false
	}
	u, err := url.Parse(rawValue)
	if err != nil || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Hostname(), expectedHost)
}

type refreshReq struct {
	RefreshToken       string `json:"refresh_token" binding:"required,max=256"`
	NeedUserSigRefresh bool   `json:"need_user_sig_refresh"`
	UserSigExpiresAt   string `json:"user_sig_expires_at" binding:"omitempty,max=64"`
	SDKAppID           int    `json:"sdk_app_id"`
	DeviceID           string `json:"device_id"   binding:"omitempty,max=128"`
	DeviceName         string `json:"device_name" binding:"omitempty,max=128"`
	Platform           string `json:"platform"    binding:"omitempty,max=32"`
	AppVersion         string `json:"app_version" binding:"omitempty,max=64"`
}

// Refresh 使用 refresh token 换发新的 access token。
func Refresh(c *gin.Context) {
	started := time.Now()
	var trace [16]byte
	_, _ = rand.Read(trace[:])
	requestID := hex.EncodeToString(trace[:])
	c.Header("X-Request-ID", requestID)
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "续签请求格式错误")
		return
	}
	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" || len(refreshToken) > 256 {
		response.InvalidParams(c, "refresh_token 格式错误")
		return
	}
	req.RefreshToken = refreshToken
	clientIP := clientip.FromContext(c)
	if !authlimit.Enforce(c, "refresh_token", []ratelimit.SubjectRules{
		{Subject: ratelimit.HashSubject("refresh", refreshToken), Rules: ratelimit.PublicRefreshTokenRules},
		{Subject: ratelimit.HashSubject("device", req.DeviceID), Rules: ratelimit.PublicRefreshDeviceRules},
	}) {
		return
	}

	result, err := controller_chat.RefreshSession(c.Request.Context(), controller_chat.RefreshReq{
		RefreshToken:       req.RefreshToken,
		NeedUserSigRefresh: req.NeedUserSigRefresh,
		UserSigExpiresAt:   req.UserSigExpiresAt,
		SDKAppID:           req.SDKAppID,
		SessionMeta: controller_chat.SessionMeta{
			DeviceID:   req.DeviceID,
			DeviceName: req.DeviceName,
			Platform:   req.Platform,
			AppVersion: req.AppVersion,
			IP:         clientIP,
			UserAgent:  c.Request.UserAgent(),
		},
	})
	if err != nil {
		writeRefreshFailure(c, err, requestID, started)
		return
	}
	response.OK(c, result)
}

func writeRefreshFailure(c *gin.Context, err error, requestID string, started time.Time) {
	details := controller_chat.DescribeRefreshFailure(err)
	logger.Warnf("auth_refresh request_id=%s stage=%s reason=%s elapsed_ms=%d", requestID, details.Stage, details.ErrorCode, time.Since(started).Milliseconds())
	status, message := http.StatusUnauthorized, "登录状态已失效，请重新登录"
	if details.Retryable {
		status, message = http.StatusServiceUnavailable, "续签服务暂时不可用，请稍后重试"
		c.Header("Retry-After", "1")
	}
	c.JSON(status, response.Response{Code: status, Message: message, Data: details})
}

type logoutReq struct {
	RefreshToken string `json:"refresh_token" binding:"omitempty,max=256"`
	DeviceID     string `json:"device_id"     binding:"omitempty,max=128"`
}

// Logout 用户登出；新客户端传 refresh_token 撤销会话，旧客户端保持兼容返回成功。
func Logout(c *gin.Context) {
	var req logoutReq
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.InvalidParams(c, err.Error())
		return
	}
	if err := controller_chat.Logout(c.Request.Context(), req.RefreshToken, req.DeviceID); err != nil {
		var ae controller_chat.AuthErr
		if errors.As(err, &ae) {
			response.Unauthorized(c, ae.Error())
			return
		}
		response.InternalError(c)
		return
	}
	response.OK(c, nil)
}

type deleteAccountReq struct {
	DeviceID   string `json:"device_id"   binding:"omitempty,max=128"`
	DeviceName string `json:"device_name" binding:"omitempty,max=128"`
	Platform   string `json:"platform"    binding:"omitempty,max=32"`
	AppVersion string `json:"app_version" binding:"omitempty,max=64"`
}

// DeleteAccount 注销当前用户账号。
func DeleteAccount(c *gin.Context) {
	var req deleteAccountReq
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.InvalidParams(c, err.Error())
		return
	}

	imUserID := c.GetString("im_user_id")
	if err := controller_chat.DeleteAccount(c.Request.Context(), imUserID, controller_chat.SessionMeta{
		DeviceID:   req.DeviceID,
		DeviceName: req.DeviceName,
		Platform:   req.Platform,
		AppVersion: req.AppVersion,
		IP:         clientip.FromContext(c),
		UserAgent:  c.Request.UserAgent(),
	}); err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, nil)
}

type changePasswordReq struct {
	NewPassword string `json:"new_password" binding:"required,min=8,max=32"`
	DeviceID    string `json:"device_id"    binding:"omitempty,max=128"`
	DeviceName  string `json:"device_name"  binding:"omitempty,max=128"`
	Platform    string `json:"platform"     binding:"omitempty,max=32"`
	AppVersion  string `json:"app_version"  binding:"omitempty,max=64"`
}

// ChangePassword 修改密码（未激活账号首登 或 管理员重置密码后使用）。
func ChangePassword(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	result, err := controller_chat.ChangePassword(c.Request.Context(), imUserID, req.NewPassword, controller_chat.SessionMeta{
		DeviceID:   req.DeviceID,
		DeviceName: req.DeviceName,
		Platform:   req.Platform,
		AppVersion: req.AppVersion,
		IP:         clientip.FromContext(c),
		UserAgent:  c.Request.UserAgent(),
	})
	if err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		var ae controller_chat.AuthErr
		if errors.As(err, &ae) {
			response.Forbidden(c, ae.Error())
			return
		}
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

type resetPasswordReq struct {
	Phone       string `json:"phone"        binding:"required,max=20"`
	Code        string `json:"code"         binding:"required,max=10"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=32"`
	DeviceID    string `json:"device_id"    binding:"omitempty,max=128"`
	DeviceName  string `json:"device_name"  binding:"omitempty,max=128"`
	Platform    string `json:"platform"     binding:"omitempty,max=32"`
	AppVersion  string `json:"app_version"  binding:"omitempty,max=64"`
}

// ResetPassword 忘记密码时通过验证码重置（未登录）。
func ResetPassword(c *gin.Context) {
	var req resetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Phone == "" {
		response.InvalidParams(c, "手机号不能为空")
		return
	}
	clientIP := clientip.FromContext(c)
	if !authlimit.Enforce(c, "password_reset", []ratelimit.SubjectRules{
		{Subject: ratelimit.HashSubject("phone", req.Phone), Rules: ratelimit.PublicPasswordResetPhoneRules},
		{Subject: ratelimit.HashSubject("device", req.DeviceID), Rules: ratelimit.PublicPasswordResetDeviceRules},
	}) {
		return
	}
	if err := controller_chat.ResetPassword(c.Request.Context(), req.Phone, req.Code, req.NewPassword, controller_chat.SessionMeta{
		DeviceID:   req.DeviceID,
		DeviceName: req.DeviceName,
		Platform:   req.Platform,
		AppVersion: req.AppVersion,
		IP:         clientIP,
		UserAgent:  c.Request.UserAgent(),
	}); err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
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

type updatePasswordReq struct {
	CurrentPassword string `json:"current_password" binding:"required,min=6,max=32"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=32"`
	DeviceID        string `json:"device_id"       binding:"omitempty,max=128"`
	DeviceName      string `json:"device_name"     binding:"omitempty,max=128"`
	Platform        string `json:"platform"        binding:"omitempty,max=32"`
	AppVersion      string `json:"app_version"     binding:"omitempty,max=64"`
}

// UpdatePassword 登录后主动修改密码。
func UpdatePassword(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	var req updatePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	if err := controller_chat.UpdatePassword(
		c.Request.Context(),
		imUserID,
		req.CurrentPassword,
		req.NewPassword,
		controller_chat.SessionMeta{
			DeviceID:   req.DeviceID,
			DeviceName: req.DeviceName,
			Platform:   req.Platform,
			AppVersion: req.AppVersion,
			IP:         clientip.FromContext(c),
			UserAgent:  c.Request.UserAgent(),
		},
	); err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
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

type bindInviteCodeReq struct {
	InviteCode string `json:"invite_code" binding:"required,max=32"`
}

// BindInviteCode 为当前登录用户绑定邀请码并切换归属。
func BindInviteCode(c *gin.Context) {
	imUserID := c.GetString("im_user_id")
	var req bindInviteCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	result, err := controller_chat.BindInviteCode(
		c.Request.Context(),
		imUserID,
		req.InviteCode,
	)
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
