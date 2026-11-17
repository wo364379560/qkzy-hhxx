package config

import (
	"context"
	"errors"
	"strconv"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"gorm.io/gorm"
)

const admin2FAEnabledSettingKey = "admin_2fa_enabled"

// Get2FAEnabled 获取全局 2FA 开关状态。
func Get2FAEnabled(ctx context.Context, db *gorm.DB) (bool, error) {
	if db == nil {
		return false, errors.New("mysql datasource not initialized")
	}

	var setting model.SystemSetting
	err := db.WithContext(ctx).
		Where("setting_key = ?", admin2FAEnabledSettingKey).
		Take(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	enabled, err := strconv.ParseBool(setting.Value)
	if err != nil {
		return false, err
	}
	return enabled, nil
}

// Set2FAEnabledTx 在指定数据库连接或事务中设置全局 2FA 开关。
func Set2FAEnabledTx(ctx context.Context, db *gorm.DB, enabled bool, updatedBy uint) error {
	if db == nil {
		return errors.New("mysql datasource not initialized")
	}

	return upsertSystemSettingTx(ctx, db, admin2FAEnabledSettingKey, strconv.FormatBool(enabled), updatedBy)
}
