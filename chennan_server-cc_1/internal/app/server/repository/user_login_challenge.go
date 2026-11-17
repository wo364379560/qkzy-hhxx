package repository

import (
	"errors"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const loginChallengeTTL = 5 * time.Minute

// UserLoginChallengeRepository handles user_login_challenges table.
type UserLoginChallengeRepository struct {
	db *gorm.DB
}

func NewUserLoginChallengeRepository() *UserLoginChallengeRepository {
	return &UserLoginChallengeRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func NewUserLoginChallengeRepositoryWithDB(db *gorm.DB) *UserLoginChallengeRepository {
	return &UserLoginChallengeRepository{db: db}
}

func (r *UserLoginChallengeRepository) Create(challenge *model.UserLoginChallenge) error {
	return r.db.Create(challenge).Error
}

func (r *UserLoginChallengeRepository) FindByID(id uint) (*model.UserLoginChallenge, error) {
	var challenge model.UserLoginChallenge
	if err := r.db.Where("id = ?", id).First(&challenge).Error; err != nil {
		return nil, err
	}
	return &challenge, nil
}

func (r *UserLoginChallengeRepository) FindByIDForUpdate(id uint) (*model.UserLoginChallenge, error) {
	var challenge model.UserLoginChallenge
	if err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&challenge).Error; err != nil {
		return nil, err
	}
	return &challenge, nil
}

func (r *UserLoginChallengeRepository) FindByIDAndTokenHash(id uint, tokenHash string) (*model.UserLoginChallenge, error) {
	var challenge model.UserLoginChallenge
	if err := r.db.Where("id = ? AND pre_auth_token_hash = ?", id, tokenHash).First(&challenge).Error; err != nil {
		return nil, err
	}
	return &challenge, nil
}

func (r *UserLoginChallengeRepository) ListPendingByUserID(userID uint) ([]model.UserLoginChallenge, error) {
	now := time.Now()
	var items []model.UserLoginChallenge
	err := r.db.
		Where("user_id = ? AND status = ? AND expires_at > ?", userID, model.UserLoginChallengeStatusPending, now).
		Order("created_at DESC").
		Find(&items).Error
	return items, err
}

func (r *UserLoginChallengeRepository) MarkExpired(id uint, now time.Time) error {
	return r.db.Model(&model.UserLoginChallenge{}).
		Where("id = ? AND status = ?", id, model.UserLoginChallengeStatusPending).
		Update("status", model.UserLoginChallengeStatusExpired).Error
}

func (r *UserLoginChallengeRepository) Approve(id uint, approverDeviceID string, now time.Time) (bool, error) {
	result := r.db.Model(&model.UserLoginChallenge{}).
		Where("id = ? AND status = ? AND expires_at > ?", id, model.UserLoginChallengeStatusPending, now).
		Updates(map[string]any{
			"status":             model.UserLoginChallengeStatusApproved,
			"approver_device_id": approverDeviceID,
		})
	return result.RowsAffected > 0, result.Error
}

func (r *UserLoginChallengeRepository) Deny(id uint, approverDeviceID string, now time.Time) (bool, error) {
	result := r.db.Model(&model.UserLoginChallenge{}).
		Where("id = ? AND status = ? AND expires_at > ?", id, model.UserLoginChallengeStatusPending, now).
		Updates(map[string]any{
			"status":             model.UserLoginChallengeStatusDenied,
			"approver_device_id": approverDeviceID,
		})
	return result.RowsAffected > 0, result.Error
}

func (r *UserLoginChallengeRepository) Consume(id uint, now time.Time) (bool, error) {
	result := r.db.Model(&model.UserLoginChallenge{}).
		Where("id = ? AND status = ? AND expires_at > ?", id, model.UserLoginChallengeStatusApproved, now).
		Updates(map[string]any{
			"status":      model.UserLoginChallengeStatusConsumed,
			"consumed_at": now,
		})
	return result.RowsAffected > 0, result.Error
}

func (r *UserLoginChallengeRepository) ExpireStalePending(now time.Time) error {
	return r.db.Model(&model.UserLoginChallenge{}).
		Where("status = ? AND expires_at <= ?", model.UserLoginChallengeStatusPending, now).
		Update("status", model.UserLoginChallengeStatusExpired).Error
}

func LoginChallengeTTL() time.Duration {
	return loginChallengeTTL
}

// HasActiveSessionForDevice returns true when the user has a valid session on deviceID.
func (r *UserRefreshSessionRepository) HasActiveSessionForDevice(userID uint, deviceID string) (bool, error) {
	if deviceID == "" {
		return false, nil
	}
	var count int64
	err := r.db.Model(&model.UserRefreshSession{}).
		Where("user_id = ? AND device_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, deviceID, time.Now()).
		Count(&count).Error
	return count > 0, err
}

// deviceApprovalSessionFreshness：仅把近期活跃的其它设备视为「可批准方」。
// 过期很久但仍未 revoke 的 refresh 会话不再触发新设备验证，避免幽灵会话坑本机。
const deviceApprovalSessionFreshness = 7 * 24 * time.Hour

// HasActiveSessionOnOtherDevice returns true when another non-empty device_id has a recent valid session.
func (r *UserRefreshSessionRepository) HasActiveSessionOnOtherDevice(userID uint, deviceID string) (bool, error) {
	now := time.Now()
	freshAfter := now.Add(-deviceApprovalSessionFreshness)
	var count int64
	q := r.db.Model(&model.UserRefreshSession{}).
		Where(
			"user_id = ? AND device_id <> '' AND revoked_at IS NULL AND expires_at > ? AND last_used_at > ?",
			userID, now, freshAfter,
		)
	if deviceID != "" {
		q = q.Where("device_id <> ?", deviceID)
	}
	err := q.Count(&count).Error
	return count > 0, err
}

// FindActiveSessionDeviceID resolves the current device's device_id from refresh token hash.
func (r *UserRefreshSessionRepository) FindActiveSessionDeviceID(userID uint, refreshTokenHash string) (string, error) {
	if refreshTokenHash == "" {
		return "", nil
	}
	var session model.UserRefreshSession
	err := r.db.
		Where("user_id = ? AND refresh_token_hash = ? AND revoked_at IS NULL AND expires_at > ?", userID, refreshTokenHash, time.Now()).
		First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return session.DeviceID, nil
}
