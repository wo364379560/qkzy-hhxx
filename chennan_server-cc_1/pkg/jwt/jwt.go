package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ixayda/ix-Go/internal/pkg/config"
)

// ──────────────────────────────────────────────
// Admin Claims（管理系统）
// ──────────────────────────────────────────────

// Claims 是管理端 JWT 载荷。
type Claims struct {
	AdminID     uint   `json:"admin_id"`
	Role        string `json:"role"`
	AuthVersion int    `json:"auth_version"`
	jwt.RegisteredClaims
}

// Generate 生成管理端 JWT token。
func Generate(adminID uint, role string, authVersion int) (string, error) {
	cfg := config.Cfg.Get()
	secret := cfg.JWT.Secret
	expiry := adminTokenExpiry(role, cfg.JWT.Expiry, cfg.JWT.SuperAdminExpiry)
	if expiry == 0 {
		expiry = 24 * time.Hour
	}
	issuedAt := time.Now().UTC()

	claims := Claims{
		AdminID:     adminID,
		Role:        role,
		AuthVersion: authVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(issuedAt.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func adminTokenExpiry(role string, defaultExpiry, superAdminExpiry time.Duration) time.Duration {
	if role == "super_admin" && superAdminExpiry > 0 {
		return superAdminExpiry
	}
	return defaultExpiry
}

// Parse 解析并验证管理端 JWT token，返回 Claims。
func Parse(tokenStr string) (*Claims, error) {
	secret := config.Cfg.Get().JWT.Secret
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// ──────────────────────────────────────────────
// UserClaims（App 端）
// ──────────────────────────────────────────────

// UserClaims 是 App 端用户 JWT 载荷。
type UserClaims struct {
	IMUserID    string `json:"im_user_id"` // w_xxxxx 格式
	AuthVersion int    `json:"auth_version"`
	jwt.RegisteredClaims
}

// GenerateUser 生成 App 端用户 JWT token。
func GenerateUser(imUserID string, authVersion int) (string, error) {
	token, _, err := GenerateUserWithExpiry(imUserID, authVersion)
	return token, err
}

// GenerateUserWithExpiry 生成 App 端用户 JWT token，并返回准确过期时间。
func GenerateUserWithExpiry(imUserID string, authVersion int) (string, time.Time, error) {
	cfg := config.Cfg.Get()
	secret := cfg.JWT.Secret
	expiry := cfg.JWT.Expiry
	if expiry == 0 {
		expiry = 24 * time.Hour
	}
	issuedAt := time.Now().UTC()
	expiresAt := issuedAt.Add(expiry)

	claims := UserClaims{
		IMUserID:    imUserID,
		AuthVersion: authVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

// ParseUser 解析并验证 App 端用户 JWT token，返回 UserClaims。
func ParseUser(tokenStr string) (*UserClaims, error) {
	secret := config.Cfg.Get().JWT.Secret
	token, err := jwt.ParseWithClaims(tokenStr, &UserClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// ──────────────────────────────────────────────
// TempClaims（临时 Token，用于 2FA 流程）
// ──────────────────────────────────────────────

// TempClaims 是临时 JWT 载荷。
type TempClaims struct {
	AdminID      uint   `json:"admin_id"`
	AuthVersion  int    `json:"auth_version,omitempty"`
	UserID       uint   `json:"user_id,omitempty"`
	Purpose      string `json:"purpose"`               // "totp_setup" | "totp_verify"
	TOTPSecret   string `json:"totp_secret,omitempty"` // 仅绑定流程使用
	LoginAuditID uint   `json:"login_audit_id,omitempty"`
	DeviceID     string `json:"device_id,omitempty"`
	DeviceName   string `json:"device_name,omitempty"`
	Platform     string `json:"platform,omitempty"`
	AppVersion   string `json:"app_version,omitempty"`
	jwt.RegisteredClaims
}

// GenerateTempToken 生成临时 token（5 分钟有效）。
func GenerateTempToken(adminID uint, purpose, totpSecret string, loginAuditID uint, authVersions ...int) (string, error) {
	cfg := config.Cfg.Get()
	secret := cfg.JWT.Secret
	authVersion := 1
	if len(authVersions) > 0 && authVersions[0] > 0 {
		authVersion = authVersions[0]
	}

	claims := TempClaims{
		AdminID:      adminID,
		AuthVersion:  authVersion,
		Purpose:      purpose,
		TOTPSecret:   totpSecret,
		LoginAuditID: loginAuditID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateUserTempToken 生成用户侧临时 token（5 分钟有效）。
func GenerateUserTempToken(userID uint, purpose, totpSecret, deviceID, deviceName, platform, appVersion string) (string, error) {
	cfg := config.Cfg.Get()
	secret := cfg.JWT.Secret

	claims := TempClaims{
		UserID:     userID,
		Purpose:    purpose,
		TOTPSecret: totpSecret,
		DeviceID:   deviceID,
		DeviceName: deviceName,
		Platform:   platform,
		AppVersion: appVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseTempToken 解析临时 token。
func ParseTempToken(tokenStr string) (*TempClaims, error) {
	secret := config.Cfg.Get().JWT.Secret
	token, err := jwt.ParseWithClaims(tokenStr, &TempClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*TempClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
