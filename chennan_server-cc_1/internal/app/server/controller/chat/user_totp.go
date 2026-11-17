package controller_chat

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/internal/pkg/totp"
	pkgjwt "github.com/ixayda/ix-Go/pkg/jwt"
)

type UserTOTPStatusResult struct {
	Enabled bool   `json:"enabled"`
	BoundAt string `json:"bound_at"`
}

type UserTOTPSetupResult struct {
	TempToken  string `json:"temp_token"`
	TOTPSecret string `json:"totp_secret"`
	TOTPURI    string `json:"totp_uri"`
}

type UserTOTPBindResult struct {
	Enabled bool `json:"enabled"`
}

type UserTOTPVerifyReq struct {
	TempToken string
	Code      string
	SessionMeta
}

type UserTOTPDisableReq struct {
	CurrentPassword string
	Code            string
}

type UserTOTPBackupCodesResult struct{}

func GetUserTOTPStatus(imUserID string) (*UserTOTPStatusResult, error) {
	user, err := repository.NewUserRepository().FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}
	setting, err := repository.NewUserTOTPRepository().FindByUserID(user.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	result := &UserTOTPStatusResult{}
	if setting == nil || !setting.Enabled {
		return result, nil
	}
	result.Enabled = true
	if setting.BoundAt != nil {
		result.BoundAt = setting.BoundAt.Format("2006-01-02 15:04:05")
	}
	return result, nil
}

func SetupUserTOTP(imUserID string) (*UserTOTPSetupResult, error) {
	user, err := repository.NewUserRepository().FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}
	if err := ensureUserTOTPManageable(user); err != nil {
		return nil, err
	}
	setting, err := repository.NewUserTOTPRepository().FindByUserID(user.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if setting != nil && setting.Enabled && strings.TrimSpace(setting.TOTPSecret) != "" {
		return nil, AuthErr("Google 验证已开启")
	}
	secret, uri, err := totp.GenerateTOTPSecret(user.Phone)
	if err != nil {
		return nil, err
	}
	tempToken, err := pkgjwt.GenerateUserTempToken(user.ID, "user_totp_setup", secret, "", "", "", "")
	if err != nil {
		return nil, err
	}
	return &UserTOTPSetupResult{
		TempToken:  tempToken,
		TOTPSecret: secret,
		TOTPURI:    uri,
	}, nil
}

func BindUserTOTP(imUserID, tempToken, code string) (*UserTOTPBindResult, error) {
	claims, err := pkgjwt.ParseTempToken(tempToken)
	if err != nil || claims.Purpose != "user_totp_setup" || claims.UserID == 0 || claims.TOTPSecret == "" {
		return nil, AuthErr("绑定信息已失效，请重新开启")
	}
	user, err := repository.NewUserRepository().FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}
	if user.ID != claims.UserID {
		return nil, AuthErr("绑定信息不匹配")
	}
	if err := ensureUserTOTPManageable(user); err != nil {
		return nil, err
	}
	if !totp.ValidateTOTP(claims.TOTPSecret, strings.TrimSpace(code)) {
		return nil, AuthErr("验证码错误")
	}
	encrypted, err := totp.EncryptTOTPSecret(claims.TOTPSecret)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	db := datasource.Datasource.Mysql.GetClient()
	if err := db.Transaction(func(tx *gorm.DB) error {
		return repository.NewUserTOTPRepositoryWithDB(tx).Enable(user.ID, encrypted, now)
	}); err != nil {
		return nil, err
	}
	return &UserTOTPBindResult{Enabled: true}, nil
}

func VerifyUserTOTP(ctx context.Context, req UserTOTPVerifyReq) (*AuthResult, error) {
	claims, err := pkgjwt.ParseTempToken(req.TempToken)
	if err != nil || claims.Purpose != "user_totp_verify" || claims.UserID == 0 {
		return nil, AuthErr("验证已失效，请重新登录")
	}
	user, err := loadUserForTOTPLogin(claims.UserID)
	if err != nil {
		return nil, err
	}
	setting, err := repository.NewUserTOTPRepository().FindByUserID(user.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, AuthErr("Google 验证未开启")
		}
		return nil, err
	}
	if !setting.Enabled || setting.TOTPSecret == "" {
		return nil, AuthErr("Google 验证未开启")
	}
	secret, err := totp.DecryptTOTPSecret(setting.TOTPSecret)
	if err != nil {
		return nil, err
	}
	verified := false
	if strings.TrimSpace(req.Code) != "" {
		verified = totp.ValidateTOTP(secret, strings.TrimSpace(req.Code))
	}
	if !verified {
		return nil, AuthErr("验证码错误")
	}
	return completeLoginAfterTOTP(ctx, user, req.SessionMeta, claims)
}

// SetupVerifyUserTOTP 登录时首次绑定 Google 验证器并签发会话。
func SetupVerifyUserTOTP(ctx context.Context, req UserTOTPVerifyReq) (*AuthResult, error) {
	claims, err := pkgjwt.ParseTempToken(req.TempToken)
	if err != nil || claims.Purpose != "user_totp_setup" || claims.UserID == 0 || claims.TOTPSecret == "" {
		return nil, AuthErr("绑定信息已失效，请重新登录")
	}
	user, err := loadUserForTOTPLogin(claims.UserID)
	if err != nil {
		return nil, err
	}
	if !totp.ValidateTOTP(claims.TOTPSecret, strings.TrimSpace(req.Code)) {
		return nil, AuthErr("验证码错误")
	}
	encrypted, err := totp.EncryptTOTPSecret(claims.TOTPSecret)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	db := datasource.Datasource.Mysql.GetClient()
	if err := db.Transaction(func(tx *gorm.DB) error {
		return repository.NewUserTOTPRepositoryWithDB(tx).Enable(user.ID, encrypted, now)
	}); err != nil {
		return nil, err
	}
	return completeLoginAfterTOTP(ctx, user, req.SessionMeta, claims)
}

func loadUserForTOTPLogin(userID uint) (*model.User, error) {
	user, err := repository.NewUserRepository().FindByID(userID)
	if err != nil {
		return nil, err
	}
	if user.Status == "banned" {
		return nil, AuthErr("账号已被封禁")
	}
	if user.Status == "deleted" {
		return nil, AuthErr("账号不存在")
	}
	return user, nil
}

func completeLoginAfterTOTP(ctx context.Context, user *model.User, reqMeta SessionMeta, claims *pkgjwt.TempClaims) (*AuthResult, error) {
	meta := normalizeSessionMeta(reqMeta)
	if claims != nil {
		if meta.DeviceID == "" {
			meta.DeviceID = claims.DeviceID
		}
		if meta.DeviceName == "" {
			meta.DeviceName = claims.DeviceName
		}
		if meta.Platform == "" {
			meta.Platform = claims.Platform
		}
		if meta.AppVersion == "" {
			meta.AppVersion = claims.AppVersion
		}
	}
	requireApproval, err := shouldRequireDeviceApproval(user.ID, meta)
	if err != nil {
		return nil, err
	}
	if requireApproval {
		return createLoginChallengeResult(user, meta)
	}
	trustCurrentDeviceIfProtected(user.ID, meta)
	return issueLoginSessionWithAlert(ctx, user, meta)
}

func DisableUserTOTP(imUserID string, req UserTOTPDisableReq) error {
	return AuthErr("Google 验证开启后不能自行关闭，请联系管理员处理")
}

func RegenerateUserTOTPBackupCodes(imUserID, currentPassword string) (*UserTOTPBackupCodesResult, error) {
	return nil, AuthErr("备用码功能已关闭，请使用 Google 验证码登录")
}

func disableUserTOTPByUserID(userID uint, revokeSessions bool) error {
	db := datasource.Datasource.Mysql.GetClient()
	return db.Transaction(func(tx *gorm.DB) error {
		if err := repository.NewUserTOTPRepositoryWithDB(tx).Disable(userID); err != nil {
			return err
		}
		if err := repository.NewUserTOTPBackupCodeRepositoryWithDB(tx).DeleteByUserID(userID); err != nil {
			return err
		}
		if revokeSessions {
			now := time.Now()
			if err := tx.Model(&model.UserRefreshSession{}).
				Where("user_id = ? AND revoked_at IS NULL", userID).
				Update("revoked_at", now).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func ensureUserTOTPManageable(user *model.User) error {
	if user == nil {
		return AuthErr("用户不存在")
	}
	if user.Status == "deleted" {
		return AuthErr("账号不存在")
	}
	if user.Status == "banned" {
		return AuthErr("账号已被封禁")
	}
	return nil
}
