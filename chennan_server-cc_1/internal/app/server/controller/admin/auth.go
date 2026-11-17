package controller_admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/security"
	"github.com/ixayda/ix-Go/internal/pkg/authwork"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/internal/pkg/totp"
	"github.com/ixayda/ix-Go/internal/pkg/turnstile"
	pkgjwt "github.com/ixayda/ix-Go/pkg/jwt"
)

// AuthUser 是后台登录态中的用户信息。
type AuthUser struct {
	ID                 uint    `json:"id"`
	Username           string  `json:"username"`
	DisplayName        string  `json:"displayName"`
	Role               string  `json:"role"`
	Status             string  `json:"status"`
	CreatedAt          string  `json:"createdAt"`
	TOTPEnabled        bool    `json:"totpEnabled"`
	TOTPBoundAt        *string `json:"totpBoundAt"`
	IPWhitelistEnabled bool    `json:"ipWhitelistEnabled"`
}

// AuthErr is safe to return to an unauthenticated client. Infrastructure,
// configuration, database and signing errors remain ordinary errors.
type AuthErr string

func (e AuthErr) Error() string { return string(e) }

func buildAuthUser(admin *model.AdminUser) *AuthUser {
	var boundAt *string
	if admin.TOTPBoundAt != nil {
		formatted := admin.TOTPBoundAt.Format("2006-01-02 15:04:05")
		boundAt = &formatted
	}

	return &AuthUser{
		ID:                 admin.ID,
		Username:           admin.Username,
		DisplayName:        admin.DisplayName,
		Role:               admin.Role,
		Status:             admin.Status,
		CreatedAt:          admin.CreatedAt.Format("2006-01-02 15:04:05"),
		TOTPEnabled:        admin.TOTPEnabled,
		TOTPBoundAt:        boundAt,
		IPWhitelistEnabled: admin.IPWhitelistEnabled,
	}
}

// LoginResult 是登录成功后的响应数据。
type LoginResult struct {
	Token             string    `json:"token,omitempty"`
	User              *AuthUser `json:"user,omitempty"`
	RequireTOTPSetup  bool      `json:"require_totp_setup,omitempty"`
	RequireTOTPVerify bool      `json:"require_totp_verify,omitempty"`
	TempToken         string    `json:"temp_token,omitempty"`
	TOTPSecret        string    `json:"totp_secret,omitempty"`
	TOTPURI           string    `json:"totp_uri,omitempty"`
}

// Login 验证用户名+密码，返回 JWT token 和用户信息。
func Login(ctx context.Context, username, password, turnstileToken, clientIP, userAgent string) (*LoginResult, error) {
	username = strings.TrimSpace(username)
	repo := repository.NewAdminUserRepository()
	now := time.Now()
	riskRepo := repository.NewLoginRiskControlRepository()
	loginAudit := &model.AdminLoginAudit{
		Username:  username,
		Stage:     "turnstile",
		IPAddress: clientIP,
		UserAgent: userAgent,
		Result:    "failed",
	}

	if err := turnstile.Verify(ctx, turnstileToken, clientIP); err != nil {
		loginAudit.FailureReason = err.Error()
		_ = RecordAdminLoginAudit(loginAudit)
		return nil, AuthErr("人机验证失败，请重试")
	}
	loginAudit.Stage = "password"

	admin, err := repo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = riskRepo.RecordAdminIPLoginFailure(clientIP, username, now)
			loginAudit.FailureReason = "用户名或密码错误"
			_ = RecordAdminLoginAudit(loginAudit)
			_ = riskRepo.AutoBlockAdminIPLoginFailure(clientIP, username, 0, 0, now)
			return nil, AuthErr("用户名或密码错误")
		}
		loginAudit.FailureReason = "系统错误"
		_ = RecordAdminLoginAudit(loginAudit)
		return nil, err
	}
	loginAudit.AdminUserID = &admin.ID
	loginAudit.AdminRole = admin.Role
	adminID := admin.ID

	if admin.Status != "active" {
		loginAudit.FailureReason = "账号已被禁用"
		_ = RecordAdminLoginAudit(loginAudit)
		return nil, AuthErr("账号已被禁用")
	}

	if admin.Role != RoleSuperAdmin {
		if row, active, err := riskRepo.ActiveAdminCooldown(admin.Username, now); err != nil {
			loginAudit.FailureReason = "系统错误"
			_ = RecordAdminLoginAudit(loginAudit)
			return nil, err
		} else if active {
			message := adminLoginCooldownMessage("当前账号登录已冷却", row.CooldownUntil, now)
			loginAudit.FailureReason = message
			_ = RecordAdminLoginAudit(loginAudit)
			return nil, AuthErr(message)
		}
	}

	passwordErr := authwork.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password))
	if errors.Is(passwordErr, authwork.ErrBusy) {
		return nil, passwordErr
	}
	if passwordErr != nil {
		_ = riskRepo.RecordAdminIPLoginFailure(clientIP, username, now)
		if admin.Role != RoleSuperAdmin {
			row, triggered, recordErr := riskRepo.RecordAdminPasswordFailure(username, &adminID, admin.Role, clientIP, now)
			if recordErr != nil {
				loginAudit.FailureReason = "系统错误"
				_ = RecordAdminLoginAudit(loginAudit)
				return nil, recordErr
			}
			if triggered {
				message := adminLoginCooldownMessage("密码错误次数过多", row.CooldownUntil, now)
				loginAudit.FailureReason = message
				_ = RecordAdminLoginAudit(loginAudit)
				_ = riskRepo.AutoBlockAdminIPLoginFailure(clientIP, username, 0, 0, now)
				return nil, AuthErr(message)
			}
		}
		loginAudit.FailureReason = "用户名或密码错误"
		_ = RecordAdminLoginAudit(loginAudit)
		_ = riskRepo.AutoBlockAdminIPLoginFailure(clientIP, username, 0, 0, now)
		return nil, AuthErr("用户名或密码错误")
	}

	if err := ensureAdminUserIPWhitelistAllowed(admin, clientIP); err != nil {
		loginAudit.FailureReason = err.Error()
		_ = RecordAdminLoginAudit(loginAudit)
		return nil, err
	}

	// 检查全局 2FA 开关
	enabled, err := config.Get2FAEnabled(ctx, datasource.Datasource.Mysql.GetClient())
	if err != nil {
		loginAudit.FailureReason = "读取双因素认证配置失败"
		_ = RecordAdminLoginAudit(loginAudit)
		return nil, err
	}
	if !enabled {
		// 2FA 未启用，直接返回 token
		token, err := pkgjwt.Generate(
			admin.ID,
			admin.Role,
			security.NormalizeAuthVersion(admin.AuthVersion),
		)
		if err != nil {
			loginAudit.FailureReason = "签发 token 失败"
			_ = RecordAdminLoginAudit(loginAudit)
			return nil, err
		}
		loginAudit.Result = "success"
		loginAudit.FailureReason = ""
		_ = RecordAdminLoginAudit(loginAudit)
		_ = riskRepo.ResolveAdminPasswordRisk(admin.Username, admin.ID, admin.Role, time.Now())
		return &LoginResult{
			Token: token,
			User:  buildAuthUser(admin),
		}, nil
	}

	// 检查是否已绑定 TOTP
	if !admin.TOTPEnabled {
		// 未绑定，生成新密钥
		secret, uri, err := totp.GenerateTOTPSecret(admin.Username)
		if err != nil {
			loginAudit.FailureReason = "生成 TOTP 密钥失败"
			_ = RecordAdminLoginAudit(loginAudit)
			return nil, err
		}
		loginAudit.Result = "require_totp_setup"
		loginAudit.FailureReason = ""
		_ = RecordAdminLoginAudit(loginAudit)
		_ = riskRepo.ResolveAdminPasswordRisk(admin.Username, admin.ID, admin.Role, time.Now())
		tempToken, err := pkgjwt.GenerateTempToken(admin.ID, "totp_setup", secret, loginAudit.ID, security.NormalizeAuthVersion(admin.AuthVersion))
		if err != nil {
			if loginAudit.ID > 0 {
				_ = UpdateAdminLoginAuditOutcome(loginAudit.ID, "password", "failed", "签发临时 token 失败")
			}
			return nil, err
		}
		return &LoginResult{
			RequireTOTPSetup: true,
			TempToken:        tempToken,
			TOTPSecret:       secret,
			TOTPURI:          uri,
		}, nil
	}

	loginAudit.Result = "require_totp_verify"
	loginAudit.FailureReason = ""
	_ = RecordAdminLoginAudit(loginAudit)
	_ = riskRepo.ResolveAdminPasswordRisk(admin.Username, admin.ID, admin.Role, time.Now())
	tempToken, err := pkgjwt.GenerateTempToken(admin.ID, "totp_verify", "", loginAudit.ID, security.NormalizeAuthVersion(admin.AuthVersion))
	if err != nil {
		if loginAudit.ID > 0 {
			_ = UpdateAdminLoginAuditOutcome(loginAudit.ID, "password", "failed", "签发临时 token 失败")
		}
		return nil, err
	}

	return &LoginResult{
		RequireTOTPVerify: true,
		TempToken:         tempToken,
	}, nil
}

func ensureAdminUserIPWhitelistAllowed(admin *model.AdminUser, clientIP string) error {
	if (admin.Role != "assistant" && admin.Role != "admin") || !admin.IPWhitelistEnabled {
		return nil
	}

	ruleRepo := repository.NewAdminUserIPWhitelistRepository()
	rules, err := ruleRepo.ListByAdminUserID(admin.ID)
	if err != nil {
		return err
	}

	allowed, err := security.EvaluateAdminUserIPWhitelist(admin, clientIP, rules)
	if err != nil {
		return err
	}
	if !allowed {
		return AuthErr(fmt.Sprintf("您的账号已启用 IP 白名单限制，当前登录 IP (%s) 不在白名单内，请联系上级管理员添加", clientIP))
	}
	return nil
}

func adminLoginCooldownMessage(prefix string, cooldownUntil *time.Time, now time.Time) string {
	return fmt.Sprintf("%s，请于%s后重试", prefix, formatAdminLoginCooldownRemaining(cooldownUntil, now))
}

func formatAdminLoginCooldownRemaining(cooldownUntil *time.Time, now time.Time) string {
	if cooldownUntil == nil {
		return "稍后"
	}
	remaining := cooldownUntil.Sub(now)
	if remaining <= 0 {
		return "稍后"
	}
	minutes := int((remaining + time.Minute - time.Nanosecond) / time.Minute)
	if minutes < 60 {
		return fmt.Sprintf("%d分钟", minutes)
	}
	hours := minutes / 60
	leftMinutes := minutes % 60
	if leftMinutes == 0 {
		return fmt.Sprintf("%d小时", hours)
	}
	return fmt.Sprintf("%d小时%d分钟", hours, leftMinutes)
}

// ChangePassword 验证旧密码并更新当前后台账号密码。
func ChangePassword(adminID uint, oldPassword, newPassword string) error {
	if oldPassword == newPassword {
		return AuthErr("新密码不能与当前密码相同")
	}

	repo := repository.NewAdminUserRepository()
	admin, err := repo.FindByID(adminID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AuthErr("账号不存在")
		}
		return err
	}
	if admin.Status != "active" {
		return AuthErr("账号已被禁用")
	}
	if err := authwork.ComparePrivateHashAndPassword([]byte(admin.PasswordHash), []byte(oldPassword)); err != nil {
		if errors.Is(err, authwork.ErrBusy) {
			return err
		}
		return AuthErr("当前密码错误")
	}

	hash, err := authwork.GeneratePrivateFromPassword([]byte(newPassword), authwork.DefaultCost)
	if err != nil {
		return err
	}
	return repo.UpdatePassword(adminID, string(hash))
}
