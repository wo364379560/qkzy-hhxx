package repository

import (
	"errors"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

// UserSecuritySettingsRepository handles user_security_settings table.
type UserSecuritySettingsRepository struct {
	db *gorm.DB
}

func NewUserSecuritySettingsRepository() *UserSecuritySettingsRepository {
	return &UserSecuritySettingsRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func NewUserSecuritySettingsRepositoryWithDB(db *gorm.DB) *UserSecuritySettingsRepository {
	return &UserSecuritySettingsRepository{db: db}
}

// FindByUserID returns settings for user, or ErrRecordNotFound.
func (r *UserSecuritySettingsRepository) FindByUserID(userID uint) (*model.UserSecuritySettings, error) {
	var setting model.UserSecuritySettings
	if err := r.db.Where("user_id = ?", userID).First(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

// GetOrDefault returns stored settings, or in-memory defaults when absent.
func (r *UserSecuritySettingsRepository) GetOrDefault(userID uint) (*model.UserSecuritySettings, error) {
	setting, err := r.FindByUserID(userID)
	if err == nil {
		return setting, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.DefaultUserSecuritySettings(userID), nil
	}
	return nil, err
}

// Upsert creates or updates security settings for user.
func (r *UserSecuritySettingsRepository) Upsert(userID uint, remoteLoginAlert, loginVerification *bool) (*model.UserSecuritySettings, error) {
	setting, err := r.FindByUserID(userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		setting = model.DefaultUserSecuritySettings(userID)
		if remoteLoginAlert != nil {
			setting.RemoteLoginAlert = *remoteLoginAlert
		}
		if loginVerification != nil {
			setting.LoginVerification = *loginVerification
		}
		if err := r.db.Create(setting).Error; err != nil {
			return nil, err
		}
		return setting, nil
	}

	fields := map[string]any{}
	if remoteLoginAlert != nil {
		fields["remote_login_alert"] = *remoteLoginAlert
		setting.RemoteLoginAlert = *remoteLoginAlert
	}
	if loginVerification != nil {
		fields["login_verification"] = *loginVerification
		setting.LoginVerification = *loginVerification
	}
	if len(fields) == 0 {
		return setting, nil
	}
	if err := r.db.Model(setting).Updates(fields).Error; err != nil {
		return nil, err
	}
	return setting, nil
}
