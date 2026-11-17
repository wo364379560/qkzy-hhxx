package repository

import (
	"errors"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

// UserTOTPRepository handles user Google Authenticator bindings.
type UserTOTPRepository struct {
	db *gorm.DB
}

func NewUserTOTPRepository() *UserTOTPRepository {
	return &UserTOTPRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func NewUserTOTPRepositoryWithDB(db *gorm.DB) *UserTOTPRepository {
	return &UserTOTPRepository{db: db}
}

func (r *UserTOTPRepository) FindByUserID(userID uint) (*model.UserTOTPSetting, error) {
	var setting model.UserTOTPSetting
	if err := r.db.Where("user_id = ?", userID).First(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *UserTOTPRepository) IsEnabled(userID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.UserTOTPSetting{}).
		Where("user_id = ? AND enabled = ?", userID, true).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *UserTOTPRepository) EnabledMap(userIDs []uint) (map[uint]bool, error) {
	result := make(map[uint]bool, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	var rows []model.UserTOTPSetting
	if err := r.db.Where("user_id IN ? AND enabled = ?", userIDs, true).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.UserID] = true
	}
	return result, nil
}

func (r *UserTOTPRepository) Enable(userID uint, encryptedSecret string, boundAt time.Time) error {
	var setting model.UserTOTPSetting
	err := r.db.Where("user_id = ?", userID).First(&setting).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	fields := map[string]any{
		"totp_secret": encryptedSecret,
		"enabled":     true,
		"bound_at":    &boundAt,
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.Create(&model.UserTOTPSetting{
			UserID:     userID,
			TOTPSecret: encryptedSecret,
			Enabled:    true,
			BoundAt:    &boundAt,
		}).Error
	}
	return r.db.Model(&setting).Updates(fields).Error
}

func (r *UserTOTPRepository) Disable(userID uint) error {
	return r.db.Model(&model.UserTOTPSetting{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"totp_secret": "",
			"enabled":     false,
			"bound_at":    nil,
		}).Error
}

// Require 打开用户 Google 验证开关。若尚未绑定密钥，下次 Web 登录会强制扫码绑定。
func (r *UserTOTPRepository) Require(userID uint) error {
	var setting model.UserTOTPSetting
	err := r.db.Where("user_id = ?", userID).First(&setting).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.Create(&model.UserTOTPSetting{
			UserID:     userID,
			TOTPSecret: "",
			Enabled:    true,
		}).Error
	}
	if setting.Enabled {
		return nil
	}
	return r.db.Model(&setting).Update("enabled", true).Error
}

// UserTOTPBackupCodeRepository handles user TOTP backup codes.
type UserTOTPBackupCodeRepository struct {
	db *gorm.DB
}

func NewUserTOTPBackupCodeRepository() *UserTOTPBackupCodeRepository {
	return &UserTOTPBackupCodeRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func NewUserTOTPBackupCodeRepositoryWithDB(db *gorm.DB) *UserTOTPBackupCodeRepository {
	return &UserTOTPBackupCodeRepository{db: db}
}

func (r *UserTOTPBackupCodeRepository) FindUnusedByUserID(userID uint) ([]model.UserTOTPBackupCode, error) {
	var codes []model.UserTOTPBackupCode
	err := r.db.Where("user_id = ? AND used = ?", userID, false).Find(&codes).Error
	return codes, err
}

func (r *UserTOTPBackupCodeRepository) MarkAsUsed(id uint) error {
	now := time.Now()
	return r.db.Model(&model.UserTOTPBackupCode{}).Where("id = ?", id).Updates(map[string]any{
		"used":    true,
		"used_at": &now,
	}).Error
}

func (r *UserTOTPBackupCodeRepository) DeleteByUserID(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&model.UserTOTPBackupCode{}).Error
}
