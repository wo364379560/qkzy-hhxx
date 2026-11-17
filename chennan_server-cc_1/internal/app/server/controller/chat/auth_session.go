package controller_chat

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/security"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	pkgjwt "github.com/ixayda/ix-Go/pkg/jwt"
	"github.com/ixayda/ix-Go/pkg/logger"
	"gorm.io/gorm"
)

const refreshSessionInvalidMessage = "登录状态已失效，请重新登录"

// SessionMeta 是客户端会话环境元信息。
type SessionMeta struct {
	DeviceID   string
	DeviceName string
	Platform   string
	AppVersion string
	IP         string
	UserAgent  string
}

// RefreshReq 是 refresh 接口请求参数。
type RefreshReq struct {
	RefreshToken       string
	NeedUserSigRefresh bool
	UserSigExpiresAt   string
	SDKAppID           int
	SessionMeta
}

func normalizeSessionMeta(meta SessionMeta) SessionMeta {
	normalizedIP := strings.TrimSpace(meta.IP)
	if normalizedIP != "" {
		if canonicalIP, err := security.NormalizeSingleIP(normalizedIP); err == nil {
			normalizedIP = canonicalIP
		}
	}

	return SessionMeta{
		DeviceID:   strings.TrimSpace(meta.DeviceID),
		DeviceName: strings.TrimSpace(meta.DeviceName),
		Platform:   strings.TrimSpace(meta.Platform),
		AppVersion: strings.TrimSpace(meta.AppVersion),
		IP:         normalizedIP,
		UserAgent:  strings.TrimSpace(meta.UserAgent),
	}
}

func effectiveRefreshExpiry() time.Duration {
	expiry := config.Cfg.Get().JWT.RefreshExpiry
	if expiry <= 0 {
		return 4320 * time.Hour
	}
	return expiry
}

func effectiveUserSigExpiry() time.Duration {
	expiry := config.Cfg.Get().TencentIM.UserSigExpiry
	if expiry <= 0 {
		return 168 * time.Hour
	}
	return expiry
}

func effectiveUserSigRefreshThreshold() time.Duration {
	threshold := config.Cfg.Get().TencentIM.UserSigRefreshThreshold
	if threshold <= 0 {
		return 24 * time.Hour
	}
	return threshold
}

func formatRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func parseRFC3339(raw string) (time.Time, bool) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func revokeRefreshSessionByID(tx *gorm.DB, sessionID uint, now time.Time) error {
	return tx.Model(&model.UserRefreshSession{}).
		Where("id = ? AND revoked_at IS NULL", sessionID).
		Update("revoked_at", now).Error
}

func validateRefreshSessionDeviceBinding(
	tx *gorm.DB,
	session *model.UserRefreshSession,
	reqDeviceID string,
	now time.Time,
) error {
	if session == nil {
		return AuthErr(refreshSessionInvalidMessage)
	}

	storedDeviceID := strings.TrimSpace(session.DeviceID)
	reqDeviceID = strings.TrimSpace(reqDeviceID)

	switch {
	case storedDeviceID == "" && reqDeviceID == "":
		// Compatibility mode for legacy sessions and legacy clients.
		return nil
	case storedDeviceID == "" && reqDeviceID != "":
		// Compatibility mode for legacy sessions: the upcoming Rotate call will
		// bind the session to this device ID.
		return nil
	case storedDeviceID != "" && reqDeviceID == "":
		if err := revokeRefreshSessionByID(tx, session.ID, now); err != nil {
			return err
		}
		return AuthErr(refreshSessionInvalidMessage)
	case storedDeviceID != reqDeviceID:
		if err := revokeRefreshSessionByID(tx, session.ID, now); err != nil {
			return err
		}
		return AuthErr(refreshSessionInvalidMessage)
	default:
		return nil
	}
}

func issueUserSig(user *model.User) (string, time.Time, error) {
	cfg := config.Cfg.Get().TencentIM
	expiry := effectiveUserSigExpiry()
	expiresAt := time.Now().Add(expiry)
	userSig, err := tencentim.GenUserSig(cfg.SDKAppID, cfg.AppKey, user.IMUserID, int64(expiry.Seconds()))
	if err != nil {
		return "", time.Time{}, err
	}
	return userSig, expiresAt, nil
}

func issueAccessToken(user *model.User) (string, time.Time, error) {
	return pkgjwt.GenerateUserWithExpiry(
		user.IMUserID,
		security.NormalizeAuthVersion(user.AuthVersion),
	)
}

// issueLoginSessionWithAlert 登录成功签发会话，并异步发送安全助手提醒。
func issueLoginSessionWithAlert(ctx context.Context, user *model.User, meta SessionMeta) (*AuthResult, error) {
	result, err := issueNewUserSession(ctx, user, meta)
	if err != nil {
		return nil, err
	}
	NotifyLoginSuccessAlert(user, meta)
	go ReconcileOwnerByFriends(user.IMUserID)
	return result, nil
}

func issueNewUserSession(_ context.Context, user *model.User, meta SessionMeta) (*AuthResult, error) {
	normalizedMeta := normalizeSessionMeta(meta)

	accessToken, tokenExpiresAt, err := issueAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := security.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	refreshTokenHash := security.HashRefreshToken(refreshToken)
	now := time.Now()
	refreshExpiresAt := now.Add(effectiveRefreshExpiry())

	userSig, userSigExpiresAt, err := issueUserSig(user)
	if err != nil {
		return nil, err
	}

	sessionRepo := repository.NewUserRefreshSessionRepository()
	// 同一设备只保留一条有效会话。
	if normalizedMeta.DeviceID != "" {
		if _, err := sessionRepo.RevokeAllByUserIDAndDeviceID(user.ID, normalizedMeta.DeviceID); err != nil {
			return nil, err
		}
		// 卸载重装后 device_id 偶发变化：清掉同平台+同 IP 的幽灵会话，避免列表出现两台同机。
		platform := strings.ToLower(normalizedMeta.Platform)
		if (platform == "ios" || platform == "android") && normalizedMeta.IP != "" {
			if _, err := sessionRepo.RevokeOthersByUserIDPlatformAndIP(
				user.ID,
				normalizedMeta.DeviceID,
				normalizedMeta.Platform,
				normalizedMeta.IP,
			); err != nil {
				return nil, err
			}
			_ = repository.NewUserDeviceRepository().DeleteOthersSameFingerprint(
				user.ID,
				normalizedMeta.DeviceID,
				normalizedMeta.Platform,
				normalizedMeta.IP,
			)
		}
	}

	session := &model.UserRefreshSession{
		UserID:           user.ID,
		RefreshTokenHash: refreshTokenHash,
		DeviceID:         normalizedMeta.DeviceID,
		DeviceName:       normalizedMeta.DeviceName,
		Platform:         normalizedMeta.Platform,
		AppVersion:       normalizedMeta.AppVersion,
		IP:               normalizedMeta.IP,
		UserAgent:        normalizedMeta.UserAgent,
		LastUsedAt:       now,
		ExpiresAt:        refreshExpiresAt,
	}
	if err := sessionRepo.Create(session); err != nil {
		return nil, err
	}
	// 设备身份表：user_id + device_id 唯一，登录只更新不新增重复行。
	if err := repository.NewUserDeviceRepository().UpsertLogin(
		user.ID,
		normalizedMeta.DeviceID,
		normalizedMeta.DeviceName,
		normalizedMeta.Platform,
		normalizedMeta.AppVersion,
		normalizedMeta.IP,
		now,
	); err != nil {
		logger.Warnf("upsert user_device failed: user_id=%d err=%v", user.ID, err)
	}
	if err := repository.NewUserRepository().TouchLastOnlineAt(user.ID, now); err != nil {
		logger.Warnf("touch last_online_at after login failed: user_id=%d err=%v", user.ID, err)
	}

	return &AuthResult{
		UserID:             user.IMUserID,
		SDKAppID:           config.Cfg.Get().TencentIM.SDKAppID,
		UserSig:            userSig,
		UserSigExpiresAt:   formatRFC3339(userSigExpiresAt),
		Token:              accessToken,
		TokenExpiresAt:     formatRFC3339(tokenExpiresAt),
		RefreshToken:       refreshToken,
		MustChangePassword: user.MustChangePassword,
		IsActivated:        user.IsActivated,
	}, nil
}

func shouldIssueUserSig(req RefreshReq) bool {
	return shouldIssueUserSigFor(
		req,
		config.Cfg.Get().TencentIM.SDKAppID,
		time.Now(),
		effectiveUserSigRefreshThreshold(),
	)
}

func shouldIssueUserSigFor(req RefreshReq, currentSDKAppID int, now time.Time, threshold time.Duration) bool {
	if req.NeedUserSigRefresh {
		return true
	}
	if currentSDKAppID > 0 && req.SDKAppID != currentSDKAppID {
		return true
	}
	userSigExpiresAt, ok := parseRFC3339(req.UserSigExpiresAt)
	if !ok {
		return true
	}
	return !userSigExpiresAt.After(now.Add(threshold))
}

// RefreshSession 使用 refresh token 换发新的 access token，并按需补发 UserSig。
func RefreshSession(ctx context.Context, req RefreshReq) (*AuthResult, error) {
	return refreshSessionWithDB(ctx, datasource.Datasource.Mysql.GetClient(), req)
}

func refreshSessionWithDB(ctx context.Context, db *gorm.DB, req RefreshReq) (*AuthResult, error) {
	refreshToken := security.NormalizeRefreshToken(req.RefreshToken)
	if refreshToken == "" {
		return nil, AuthErr("refresh_token 不能为空")
	}
	normalizedMeta := normalizeSessionMeta(req.SessionMeta)
	refreshTokenHash := security.HashRefreshToken(refreshToken)
	shouldRefreshUserSig := shouldIssueUserSig(req)

	var result *AuthResult
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		sessionRepo := repository.NewUserRefreshSessionRepositoryWithDB(tx)
		session, err := sessionRepo.FindByTokenHashForUpdate(refreshTokenHash)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return AuthErr(refreshSessionInvalidMessage)
			}
			return err
		}

		now := time.Now()
		if session.RevokedAt != nil || !session.ExpiresAt.After(now) {
			if session.RevokedAt == nil && !session.ExpiresAt.After(now) {
				if revokeErr := revokeRefreshSessionByID(tx, session.ID, now); revokeErr != nil {
					return revokeErr
				}
			}
			return AuthErr(refreshSessionInvalidMessage)
		}
		recoverToken, err := recoverRotatedRefreshToken(session, refreshTokenHash, normalizedMeta.DeviceID, now)
		if err != nil {
			return err
		}
		if err := validateRefreshSessionDeviceBinding(tx, session, normalizedMeta.DeviceID, now); err != nil {
			return err
		}

		user, err := repository.NewUserRepositoryWithDB(tx).FindByID(session.UserID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return AuthErr(refreshSessionInvalidMessage)
			}
			return err
		}
		if user.Status == "banned" || user.Status == "deleted" {
			return AuthErr("账号已失效，请重新登录")
		}
		if err := ensureOwnerActiveWithDB(tx, user.OwnerID); err != nil {
			return err
		}
		if err := ensureUserIPWhitelistAllowed(user, normalizedMeta.IP); err != nil {
			return err
		}

		accessToken, tokenExpiresAt, err := issueAccessToken(user)
		if err != nil {
			return err
		}

		result = &AuthResult{
			UserID:             user.IMUserID,
			SDKAppID:           config.Cfg.Get().TencentIM.SDKAppID,
			Token:              accessToken,
			TokenExpiresAt:     formatRFC3339(tokenExpiresAt),
			MustChangePassword: user.MustChangePassword,
			IsActivated:        user.IsActivated,
		}

		if shouldRefreshUserSig {
			userSig, userSigExpiresAt, err := issueUserSig(user)
			if err != nil {
				return err
			}
			result.UserSig = userSig
			result.UserSigExpiresAt = formatRFC3339(userSigExpiresAt)
		}

		// A retry receives the same rotated token. Do not move the grace deadline
		// or sliding expiry: repeated requests cannot keep a prior token alive.
		if recoverToken != "" {
			result.RefreshToken = recoverToken
			return nil
		}

		newRefreshToken, err := security.GenerateRefreshToken()
		if err != nil {
			return err
		}
		newRefreshTokenHash := security.HashRefreshToken(newRefreshToken)
		var previousHash, tokenCiphertext string
		var previousExpiresAt *time.Time
		if normalizedMeta.DeviceID != "" {
			deadline := now.Add(refreshRecoveryGrace)
			previousExpiresAt = &deadline
			previousHash = session.RefreshTokenHash
			tokenCiphertext, err = security.SealRefreshToken(newRefreshToken, config.Cfg.Get().JWT.Secret, refreshRecoveryBinding(session.ID, normalizedMeta.DeviceID))
			if err != nil {
				return err
			}
		}
		newRefreshExpiresAt := now.Add(effectiveRefreshExpiry())
		if err := sessionRepo.Rotate(
			session.ID,
			newRefreshTokenHash, previousHash, tokenCiphertext, previousExpiresAt,
			now,
			newRefreshExpiresAt,
			normalizedMeta.DeviceID,
			normalizedMeta.DeviceName,
			normalizedMeta.Platform,
			normalizedMeta.AppVersion,
			normalizedMeta.IP,
			normalizedMeta.UserAgent,
		); err != nil {
			return err
		}
		if err := repository.NewUserRepositoryWithDB(tx).TouchLastOnlineAt(user.ID, now); err != nil {
			logger.Warnf("touch last_online_at after refresh failed: user_id=%d err=%v", user.ID, err)
		}
		result.RefreshToken = newRefreshToken

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

const refreshRecoveryGrace = 60 * time.Second

func refreshRecoveryBinding(sessionID uint, deviceID string) string {
	return strconv.FormatUint(uint64(sessionID), 10) + ":" + strings.TrimSpace(deviceID)
}

func recoverRotatedRefreshToken(session *model.UserRefreshSession, requestedHash, deviceID string, now time.Time) (string, error) {
	if session == nil || session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return "", AuthErr(refreshSessionInvalidMessage)
	}
	current := requestedHash == session.RefreshTokenHash
	withinGrace := session.PreviousRefreshExpiresAt != nil && session.PreviousRefreshExpiresAt.After(now)
	boundDevice := strings.TrimSpace(deviceID) != "" && strings.TrimSpace(deviceID) == strings.TrimSpace(session.DeviceID)
	if !current && (requestedHash != session.PreviousRefreshTokenHash || !withinGrace || !boundDevice) {
		return "", AuthErr(refreshSessionInvalidMessage)
	}
	if !withinGrace || !boundDevice || session.RefreshTokenCiphertext == "" {
		if !current {
			return "", AuthErr(refreshSessionInvalidMessage)
		}
		return "", nil
	}
	token, err := security.OpenRefreshToken(session.RefreshTokenCiphertext, config.Cfg.Get().JWT.Secret, refreshRecoveryBinding(session.ID, deviceID))
	if err != nil {
		return "", err
	}
	if security.HashRefreshToken(token) != session.RefreshTokenHash {
		return "", errors.New("refresh recovery token mismatch")
	}
	return token, nil
}

// Logout revokes this session and older sessions on the same device. A delayed
// logout from an old page must not revoke a newer login on that device.
func Logout(ctx context.Context, refreshToken, clientDeviceID string) error {
	return logoutWithDB(ctx, datasource.Datasource.Mysql.GetClient(), refreshToken, clientDeviceID)
}

func logoutWithDB(ctx context.Context, db *gorm.DB, refreshToken, clientDeviceID string) error {
	normalizedToken := security.NormalizeRefreshToken(refreshToken)
	if normalizedToken == "" {
		return nil
	}
	tokenHash := security.HashRefreshToken(normalizedToken)
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		sessionRepo := repository.NewUserRefreshSessionRepositoryWithDB(tx)
		// Login revokes the old device session before creating the new one. Locking
		// this row serializes logout with that step; an already revoked token is a
		// no-op instead of a capability to kick the next login offline.
		session, err := sessionRepo.FindByTokenHashForUpdate(tokenHash)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if session.RevokedAt != nil {
			return nil
		}
		now := time.Now()
		if err := revokeRefreshSessionByID(tx, session.ID, now); err != nil {
			return err
		}
		deviceIDs := uniqueNonEmptyStrings(session.DeviceID, clientDeviceID)
		for _, deviceID := range deviceIDs {
			if err := sessionRepo.RevokeDeviceSessionsThroughID(session.UserID, deviceID, session.ID, now); err != nil {
				return err
			}
			if err := repository.NewUserDeviceRepositoryWithDB(tx).MarkOfflineThroughSession(session.UserID, deviceID, session.ID, session.CreatedAt, now); err != nil {
				return err
			}
		}
		// Record an authenticated last-session logout at its actual time. A live
		// session on another device (or a newer login) must keep the user online.
		return tx.Model(&model.User{}).
			Where("id = ? AND status = ?", session.UserID, "active").
			Where(`NOT EXISTS (SELECT 1 FROM user_refresh_sessions
				WHERE user_id = ? AND revoked_at IS NULL AND expires_at > ?)`, session.UserID, now).
			Updates(map[string]any{
				"last_online_at":           now,
				"last_offline_at":          now,
				"last_presence_status":     "offline",
				"last_presence_checked_at": now,
			}).Error
	})
}

func uniqueNonEmptyStrings(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, raw := range values {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
