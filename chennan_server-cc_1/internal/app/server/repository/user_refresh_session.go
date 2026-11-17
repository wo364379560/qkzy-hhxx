package repository

import (
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserRefreshSessionRepository 处理 user_refresh_sessions 表的数据库操作。
type UserRefreshSessionRepository struct {
	db *gorm.DB
}

// NewUserRefreshSessionRepository 创建 UserRefreshSessionRepository。
func NewUserRefreshSessionRepository() *UserRefreshSessionRepository {
	return &UserRefreshSessionRepository{db: datasource.Datasource.Mysql.GetClient()}
}

// NewUserRefreshSessionRepositoryWithDB 使用指定 DB/事务创建 UserRefreshSessionRepository。
func NewUserRefreshSessionRepositoryWithDB(db *gorm.DB) *UserRefreshSessionRepository {
	return &UserRefreshSessionRepository{db: db}
}

// Create 创建 refresh session。
func (r *UserRefreshSessionRepository) Create(session *model.UserRefreshSession) error {
	return r.db.Create(session).Error
}

// FindByTokenHash 按 refresh token hash 查询 session。
func (r *UserRefreshSessionRepository) FindByTokenHash(tokenHash string) (*model.UserRefreshSession, error) {
	var session model.UserRefreshSession
	if err := r.db.Where("refresh_token_hash = ? OR (previous_refresh_token_hash = ? AND previous_refresh_expires_at > ?)", tokenHash, tokenHash, time.Now()).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// FindByTokenHashForUpdate 按 refresh token hash 查询 session 并加行锁。
func (r *UserRefreshSessionRepository) FindByTokenHashForUpdate(tokenHash string) (*model.UserRefreshSession, error) {
	var session model.UserRefreshSession
	if err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("refresh_token_hash = ? OR (previous_refresh_token_hash = ? AND previous_refresh_expires_at > ?)", tokenHash, tokenHash, time.Now()).
		First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// Touch 更新最近使用时间，并可选滑动续期。
func (r *UserRefreshSessionRepository) Touch(id uint, lastUsedAt, expiresAt time.Time) error {
	fields := map[string]any{
		"last_used_at": lastUsedAt,
	}
	if !expiresAt.IsZero() {
		fields["expires_at"] = expiresAt
	}

	return r.db.Model(&model.UserRefreshSession{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Updates(fields).Error
}

// Rotate 轮换 refresh token hash，并更新最近使用时间、过期时间与会话元信息。
func (r *UserRefreshSessionRepository) Rotate(
	id uint,
	newTokenHash, previousTokenHash, tokenCiphertext string,
	previousExpiresAt *time.Time,
	lastUsedAt, expiresAt time.Time,
	deviceID, deviceName, platform, appVersion, ip, userAgent string,
) error {
	fields := map[string]any{
		"refresh_token_hash":          newTokenHash,
		"previous_refresh_token_hash": previousTokenHash,
		"previous_refresh_expires_at": previousExpiresAt,
		"refresh_token_ciphertext":    tokenCiphertext,
		"last_used_at":                lastUsedAt,
	}
	if !expiresAt.IsZero() {
		fields["expires_at"] = expiresAt
	}
	if deviceID != "" {
		fields["device_id"] = deviceID
	}
	if deviceName != "" {
		fields["device_name"] = deviceName
	}
	if platform != "" {
		fields["platform"] = platform
	}
	if appVersion != "" {
		fields["app_version"] = appVersion
	}
	if ip != "" {
		fields["ip"] = ip
	}
	if userAgent != "" {
		fields["user_agent"] = userAgent
	}

	return r.db.Model(&model.UserRefreshSession{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Updates(fields).Error
}

// RevokeByTokenHash 撤销指定 refresh token 对应的 session。
func (r *UserRefreshSessionRepository) RevokeByTokenHash(tokenHash string) error {
	now := time.Now()
	return r.db.Model(&model.UserRefreshSession{}).
		Where("(refresh_token_hash = ? OR (previous_refresh_token_hash = ? AND previous_refresh_expires_at > ?)) AND revoked_at IS NULL", tokenHash, tokenHash, time.Now()).
		Update("revoked_at", now).Error
}

// RevokeAllByUserID 撤销用户的全部有效 refresh sessions。
func (r *UserRefreshSessionRepository) RevokeAllByUserID(userID uint) error {
	now := time.Now()
	return r.db.Model(&model.UserRefreshSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

// ListActiveByUserID 查询用户全部有效 refresh sessions。
func (r *UserRefreshSessionRepository) ListActiveByUserID(userID uint) ([]model.UserRefreshSession, error) {
	var sessions []model.UserRefreshSession
	err := r.db.
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Order("last_used_at DESC").
		Find(&sessions).Error
	return sessions, err
}

// FindLatestActiveByUserID 查询用户最近一次有效 refresh session。
func (r *UserRefreshSessionRepository) FindLatestActiveByUserID(userID uint) (*model.UserRefreshSession, error) {
	var session model.UserRefreshSession
	if err := r.db.
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ? AND ip <> ''", userID, time.Now()).
		Order("last_used_at DESC").
		First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// FindLatestActiveByUserIDs 批量查询用户最近一次有效 refresh session。
func (r *UserRefreshSessionRepository) FindLatestActiveByUserIDs(userIDs []uint) (map[uint]model.UserRefreshSession, error) {
	if len(userIDs) == 0 {
		return map[uint]model.UserRefreshSession{}, nil
	}

	var sessions []model.UserRefreshSession
	if err := r.db.
		Where("user_id IN ? AND revoked_at IS NULL AND expires_at > ? AND ip <> ''", userIDs, time.Now()).
		Order("user_id ASC, last_used_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, err
	}

	result := make(map[uint]model.UserRefreshSession, len(userIDs))
	for _, session := range sessions {
		if _, exists := result[session.UserID]; exists {
			continue
		}
		result[session.UserID] = session
	}
	return result, nil
}

// ListForDeviceAggregation 查询用户有效会话，以及 recentDays 内撤销的会话（用于设备列表聚合）。
func (r *UserRefreshSessionRepository) ListForDeviceAggregation(userID uint, recentDays int) ([]model.UserRefreshSession, error) {
	if recentDays <= 0 {
		recentDays = 90
	}
	now := time.Now()
	revokedSince := now.AddDate(0, 0, -recentDays)

	var sessions []model.UserRefreshSession
	err := r.db.
		Where(
			"user_id = ? AND ((revoked_at IS NULL AND expires_at > ?) OR (revoked_at IS NOT NULL AND revoked_at >= ?))",
			userID, now, revokedSince,
		).
		Order("last_used_at DESC").
		Find(&sessions).Error
	return sessions, err
}

// RevokeAllByUserIDAndDeviceID 撤销指定用户+设备的全部有效 refresh sessions。
func (r *UserRefreshSessionRepository) RevokeAllByUserIDAndDeviceID(userID uint, deviceID string) (int64, error) {
	now := time.Now()
	result := r.db.Model(&model.UserRefreshSession{}).
		Where("user_id = ? AND device_id = ? AND revoked_at IS NULL", userID, deviceID).
		Update("revoked_at", now)
	return result.RowsAffected, result.Error
}

// RevokeLegacyByUserIDAndIP 撤销无 device_id 的历史会话（按 IP 聚合清理）。
func (r *UserRefreshSessionRepository) RevokeLegacyByUserIDAndIP(userID uint, ip string) (int64, error) {
	now := time.Now()
	result := r.db.Model(&model.UserRefreshSession{}).
		Where("user_id = ? AND device_id = '' AND ip = ? AND revoked_at IS NULL", userID, ip).
		Update("revoked_at", now)
	return result.RowsAffected, result.Error
}

// RevokeOthersByUserID 撤销用户除 keepDeviceID 外的全部有效会话。
func (r *UserRefreshSessionRepository) RevokeOthersByUserID(userID uint, keepDeviceID string) (int64, error) {
	now := time.Now()
	q := r.db.Model(&model.UserRefreshSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID)
	keepDeviceID = strings.TrimSpace(keepDeviceID)
	if keepDeviceID != "" {
		q = q.Where("device_id <> ?", keepDeviceID)
	}
	result := q.Update("revoked_at", now)
	return result.RowsAffected, result.Error
}

// RevokeOthersByUserIDPlatformAndIP 撤销同平台+同 IP 的其它设备会话。
// 用于 App 卸载重装后 device_id 变化时，清掉旧幽灵会话，避免列表出现两台「同 IP iPhone」。
func (r *UserRefreshSessionRepository) RevokeOthersByUserIDPlatformAndIP(userID uint, keepDeviceID, platform, ip string) (int64, error) {
	keepDeviceID = strings.TrimSpace(keepDeviceID)
	platform = strings.TrimSpace(platform)
	ip = strings.TrimSpace(ip)
	if userID == 0 || keepDeviceID == "" || platform == "" || ip == "" {
		return 0, nil
	}
	now := time.Now()
	result := r.db.Model(&model.UserRefreshSession{}).
		Where(
			"user_id = ? AND device_id <> ? AND platform = ? AND ip = ? AND revoked_at IS NULL",
			userID, keepDeviceID, platform, ip,
		).
		Update("revoked_at", now)
	return result.RowsAffected, result.Error
}

// RevokeDeviceSessionsThroughID excludes logins created after the supplied
// credential, even when their user/device identifiers are identical.
func (r *UserRefreshSessionRepository) RevokeDeviceSessionsThroughID(userID uint, deviceID string, sessionID uint, at time.Time) error {
	return r.db.Model(&model.UserRefreshSession{}).
		Where("user_id = ? AND device_id = ? AND id <= ? AND revoked_at IS NULL", userID, strings.TrimSpace(deviceID), sessionID).
		Update("revoked_at", at).Error
}
