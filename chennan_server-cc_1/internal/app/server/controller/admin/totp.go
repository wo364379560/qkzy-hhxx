package controller_admin

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/security"
	"github.com/ixayda/ix-Go/internal/pkg/authwork"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/internal/pkg/totp"
	pkgjwt "github.com/ixayda/ix-Go/pkg/jwt"
)

// BindTOTPResult 绑定 TOTP 响应。
type BindTOTPResult struct {
	Token       string    `json:"token"`
	User        *AuthUser `json:"user"`
	BackupCodes []string  `json:"backup_codes"`
}

// VerifyTOTPResult 验证 TOTP 响应。
type VerifyTOTPResult struct {
	Token string    `json:"token"`
	User  *AuthUser `json:"user"`
}

// BackupCodesResult 是备份码状态。
type BackupCodesResult struct {
	TOTPEnabled    bool `json:"totp_enabled"`
	RemainingCount int  `json:"remaining_count"`
}

// RegenerateBackupCodesResult 是重新生成备份码的响应。
type RegenerateBackupCodesResult struct {
	BackupCodes []string `json:"backup_codes"`
}

// BindTOTP 完成 TOTP 绑定。
func BindTOTP(tempToken, code string) (*BindTOTPResult, error) {
	return bindTOTPWithAudit(tempToken, code, "", "")
}

// BindTOTPWithAudit 完成 TOTP 绑定并记录登录审计。
func BindTOTPWithAudit(tempToken, code, clientIP, userAgent string) (*BindTOTPResult, error) {
	return bindTOTPWithAudit(tempToken, code, clientIP, userAgent)
}

func bindTOTPWithAudit(tempToken, code, clientIP, userAgent string) (*BindTOTPResult, error) {
	claims, err := pkgjwt.ParseTempToken(tempToken)
	if err != nil {
		return nil, AuthErr("临时 token 无效或已过期")
	}
	if claims.Purpose != "totp_setup" {
		return nil, AuthErr("token 用途不匹配")
	}
	if claims.TOTPSecret == "" {
		return nil, AuthErr("绑定信息已失效，请重新登录")
	}
	repo := repository.NewAdminUserRepository()
	admin, err := repo.FindByID(claims.AdminID)
	if err != nil {
		return nil, err
	}
	if !security.MatchesAuthVersion(claims.AuthVersion, admin.AuthVersion) {
		return nil, AuthErr("绑定信息已失效，请重新登录")
	}
	loginAudit := &model.AdminLoginAudit{
		AdminUserID: &admin.ID,
		Username:    admin.Username,
		AdminRole:   admin.Role,
		Stage:       "totp_setup",
		Result:      "failed",
		IPAddress:   clientIP,
		UserAgent:   userAgent,
	}
	if !totp.ValidateTOTP(claims.TOTPSecret, code) {
		loginAudit.FailureReason = "验证码错误"
		if claims.LoginAuditID > 0 {
			_ = UpdateAdminLoginAuditOutcome(claims.LoginAuditID, "totp_setup", "failed", loginAudit.FailureReason)
		} else {
			_ = RecordAdminLoginAudit(loginAudit)
		}
		return nil, AuthErr("验证码错误")
	}

	encrypted, err := totp.EncryptTOTPSecret(claims.TOTPSecret)
	if err != nil {
		loginAudit.FailureReason = "加密 TOTP 密钥失败"
		if claims.LoginAuditID > 0 {
			_ = UpdateAdminLoginAuditOutcome(claims.LoginAuditID, "totp_setup", "failed", loginAudit.FailureReason)
		} else {
			_ = RecordAdminLoginAudit(loginAudit)
		}
		return nil, err
	}
	if admin.Status != "active" {
		loginAudit.FailureReason = "账号已被禁用"
		if claims.LoginAuditID > 0 {
			_ = UpdateAdminLoginAuditOutcome(claims.LoginAuditID, "totp_setup", "failed", loginAudit.FailureReason)
		} else {
			_ = RecordAdminLoginAudit(loginAudit)
		}
		return nil, AuthErr("账号已被禁用")
	}
	if admin.TOTPEnabled {
		loginAudit.FailureReason = "当前账号已绑定双因素认证，请重新登录"
		if claims.LoginAuditID > 0 {
			_ = UpdateAdminLoginAuditOutcome(claims.LoginAuditID, "totp_setup", "failed", loginAudit.FailureReason)
		} else {
			_ = RecordAdminLoginAudit(loginAudit)
		}
		return nil, AuthErr("当前账号已绑定双因素认证，请重新登录")
	}

	enabled, err := config.Get2FAEnabled(context.Background(), datasource.Datasource.Mysql.GetClient())
	if err != nil {
		loginAudit.FailureReason = "系统未启用双因素认证"
		if claims.LoginAuditID > 0 {
			_ = UpdateAdminLoginAuditOutcome(claims.LoginAuditID, "totp_setup", "failed", loginAudit.FailureReason)
		} else {
			_ = RecordAdminLoginAudit(loginAudit)
		}
		return nil, err
	}
	if !enabled {
		loginAudit.FailureReason = "系统未启用双因素认证"
		if claims.LoginAuditID > 0 {
			_ = UpdateAdminLoginAuditOutcome(claims.LoginAuditID, "totp_setup", "failed", loginAudit.FailureReason)
		} else {
			_ = RecordAdminLoginAudit(loginAudit)
		}
		return nil, AuthErr("系统未启用双因素认证")
	}

	now := time.Now()
	admin.TOTPSecret = encrypted
	admin.TOTPEnabled = true
	admin.TOTPBoundAt = &now

	codes, err := totp.GenerateBackupCodes(8)
	if err != nil {
		return nil, err
	}
	hashes, err := hashBackupCodes(codes)
	if err != nil {
		return nil, err
	}

	db := datasource.Datasource.Mysql.GetClient()
	if err := db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.AdminUser{}).
			Where("id = ? AND status = ? AND totp_enabled = ? AND GREATEST(COALESCE(auth_version, 1), 1) = ?", admin.ID, "active", false, security.NormalizeAuthVersion(claims.AuthVersion)).
			Updates(map[string]any{
				"totp_secret":   encrypted,
				"totp_enabled":  true,
				"totp_bound_at": &now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return AuthErr("绑定信息已失效，请重新登录")
		}
		return replaceBackupCodes(tx, claims.AdminID, hashes)
	}); err != nil {
		return nil, err
	}

	token, err := pkgjwt.Generate(
		admin.ID,
		admin.Role,
		security.NormalizeAuthVersion(admin.AuthVersion),
	)
	if err != nil {
		loginAudit.FailureReason = "签发 token 失败"
		if claims.LoginAuditID > 0 {
			_ = UpdateAdminLoginAuditOutcome(claims.LoginAuditID, "totp_setup", "failed", loginAudit.FailureReason)
		} else {
			_ = RecordAdminLoginAudit(loginAudit)
		}
		return nil, err
	}
	if claims.LoginAuditID > 0 {
		_ = UpdateAdminLoginAuditOutcome(claims.LoginAuditID, "totp_setup", "success", "")
	} else {
		loginAudit.Result = "success"
		loginAudit.FailureReason = ""
		_ = RecordAdminLoginAudit(loginAudit)
	}

	return &BindTOTPResult{
		Token:       token,
		User:        buildAuthUser(admin),
		BackupCodes: codes,
	}, nil
}

// VerifyTOTP 验证 TOTP 码或备份码。
func VerifyTOTP(tempToken, code, backupCode string) (*VerifyTOTPResult, error) {
	return verifyTOTPWithAudit(tempToken, code, backupCode, "", "")
}

// VerifyTOTPWithAudit 验证 TOTP 码或备份码并记录登录审计。
func VerifyTOTPWithAudit(tempToken, code, backupCode, clientIP, userAgent string) (*VerifyTOTPResult, error) {
	return verifyTOTPWithAudit(tempToken, code, backupCode, clientIP, userAgent)
}

func verifyTOTPWithAudit(tempToken, code, backupCode, clientIP, userAgent string) (*VerifyTOTPResult, error) {
	claims, err := pkgjwt.ParseTempToken(tempToken)
	if err != nil {
		return nil, AuthErr("临时 token 无效或已过期")
	}
	if claims.Purpose != "totp_verify" {
		return nil, AuthErr("token 用途不匹配")
	}

	repo := repository.NewAdminUserRepository()
	admin, err := repo.FindByID(claims.AdminID)
	if err != nil {
		return nil, err
	}
	if !security.MatchesAuthVersion(claims.AuthVersion, admin.AuthVersion) {
		return nil, AuthErr("验证信息已失效，请重新登录")
	}
	loginAudit := &model.AdminLoginAudit{
		AdminUserID: &admin.ID,
		Username:    admin.Username,
		AdminRole:   admin.Role,
		Stage:       "totp_verify",
		Result:      "failed",
		IPAddress:   clientIP,
		UserAgent:   userAgent,
	}
	if admin.Status != "active" {
		loginAudit.FailureReason = "账号已被禁用"
		if claims.LoginAuditID > 0 {
			_ = UpdateAdminLoginAuditOutcome(claims.LoginAuditID, "totp_verify", "failed", loginAudit.FailureReason)
		} else {
			_ = RecordAdminLoginAudit(loginAudit)
		}
		return nil, AuthErr("账号已被禁用")
	}
	if !admin.TOTPEnabled || admin.TOTPSecret == "" {
		loginAudit.FailureReason = "当前账号尚未绑定双因素认证"
		if claims.LoginAuditID > 0 {
			_ = UpdateAdminLoginAuditOutcome(claims.LoginAuditID, "totp_verify", "failed", loginAudit.FailureReason)
		} else {
			_ = RecordAdminLoginAudit(loginAudit)
		}
		return nil, AuthErr("当前账号尚未绑定双因素认证")
	}

	secret, err := totp.DecryptTOTPSecret(admin.TOTPSecret)
	if err != nil {
		return nil, err
	}

	verified := false
	if code != "" {
		verified = totp.ValidateTOTP(secret, code)
	}

	if !verified && backupCode != "" {
		backupRepo := repository.NewTOTPBackupCodeRepository()
		codes, err := backupRepo.FindUnusedByAdminID(claims.AdminID)
		if err != nil {
			return nil, err
		}
		for _, c := range codes {
			matched, err := totp.ValidateBackupCode(backupCode, c.CodeHash)
			if err != nil {
				return nil, err
			}
			if matched {
				if err := backupRepo.MarkAsUsed(c.ID); err != nil {
					return nil, err
				}
				verified = true
				break
			}
		}
	}

	if !verified {
		loginAudit.FailureReason = "验证码错误"
		if claims.LoginAuditID > 0 {
			_ = UpdateAdminLoginAuditOutcome(claims.LoginAuditID, "totp_verify", "failed", loginAudit.FailureReason)
		} else {
			_ = RecordAdminLoginAudit(loginAudit)
		}
		return nil, AuthErr("验证码错误")
	}

	token, err := pkgjwt.Generate(
		admin.ID,
		admin.Role,
		security.NormalizeAuthVersion(admin.AuthVersion),
	)
	if err != nil {
		loginAudit.FailureReason = "签发 token 失败"
		if claims.LoginAuditID > 0 {
			_ = UpdateAdminLoginAuditOutcome(claims.LoginAuditID, "totp_verify", "failed", loginAudit.FailureReason)
		} else {
			_ = RecordAdminLoginAudit(loginAudit)
		}
		return nil, err
	}
	if claims.LoginAuditID > 0 {
		_ = UpdateAdminLoginAuditOutcome(claims.LoginAuditID, "totp_verify", "success", "")
	} else {
		loginAudit.Result = "success"
		loginAudit.FailureReason = ""
		_ = RecordAdminLoginAudit(loginAudit)
	}

	return &VerifyTOTPResult{
		Token: token,
		User:  buildAuthUser(admin),
	}, nil
}

// GetBackupCodes 获取未使用备份码的剩余状态。
func GetBackupCodes(adminID uint) (*BackupCodesResult, error) {
	repo := repository.NewAdminUserRepository()
	admin, err := repo.FindByID(adminID)
	if err != nil {
		return nil, err
	}

	result := &BackupCodesResult{TOTPEnabled: admin.TOTPEnabled}
	if !admin.TOTPEnabled {
		return result, nil
	}

	backupRepo := repository.NewTOTPBackupCodeRepository()
	codes, err := backupRepo.FindUnusedByAdminID(adminID)
	if err != nil {
		return nil, err
	}

	result.RemainingCount = len(codes)
	return result, nil
}

// RegenerateBackupCodes 重新生成备份码。
func RegenerateBackupCodes(adminID uint, password string) (*RegenerateBackupCodesResult, error) {
	repo := repository.NewAdminUserRepository()
	admin, err := repo.FindByID(adminID)
	if err != nil {
		return nil, err
	}
	if !admin.TOTPEnabled {
		return nil, AuthErr("当前账号尚未绑定双因素认证")
	}
	if err := authwork.ComparePrivateHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		if errors.Is(err, authwork.ErrBusy) {
			return nil, err
		}
		return nil, AuthErr("密码错误")
	}

	codes, err := totp.GenerateBackupCodes(8)
	if err != nil {
		return nil, err
	}
	hashes, err := hashPrivateBackupCodes(codes)
	if err != nil {
		return nil, err
	}

	db := datasource.Datasource.Mysql.GetClient()
	if err := db.Transaction(func(tx *gorm.DB) error {
		return replaceBackupCodes(tx, adminID, hashes)
	}); err != nil {
		return nil, err
	}

	return &RegenerateBackupCodesResult{BackupCodes: codes}, nil
}

func hashBackupCodes(codes []string) ([]string, error) {
	hashes := make([]string, len(codes))
	for i, code := range codes {
		hash, err := totp.HashBackupCode(code)
		if err != nil {
			return nil, err
		}
		hashes[i] = hash
	}
	return hashes, nil
}

func hashPrivateBackupCodes(codes []string) ([]string, error) {
	hashes := make([]string, len(codes))
	for i, code := range codes {
		hash, err := totp.HashBackupCodePrivate(code)
		if err != nil {
			return nil, err
		}
		hashes[i] = hash
	}
	return hashes, nil
}

func replaceBackupCodes(tx *gorm.DB, adminID uint, hashes []string) error {
	if err := tx.Where("admin_id = ?", adminID).Delete(&model.AdminTOTPBackupCode{}).Error; err != nil {
		return err
	}

	for _, hash := range hashes {
		if err := tx.Create(&model.AdminTOTPBackupCode{
			AdminID:  adminID,
			CodeHash: hash,
		}).Error; err != nil {
			return err
		}
	}

	return nil
}
