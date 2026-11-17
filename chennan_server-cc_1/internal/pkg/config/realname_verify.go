package config

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"gorm.io/gorm"
)

const realnameVerifyEnabledSettingKey = "realname_verify_enabled"

// GetRealnameVerifyEnabled 获取全局实名认证开关。
func GetRealnameVerifyEnabled(ctx context.Context, db *gorm.DB) (bool, error) {
	if db == nil {
		return false, errors.New("mysql datasource not initialized")
	}

	var setting model.SystemSetting
	err := db.WithContext(ctx).
		Where("setting_key = ?", realnameVerifyEnabledSettingKey).
		Take(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	return parseRealnameVerifyEnabled(setting.Value), nil
}

func parseRealnameVerifyEnabled(raw string) bool {
	value := strings.TrimSpace(strings.ToLower(raw))
	switch value {
	case "1", "t", "true", "on", "yes":
		return true
	default:
		return false
	}
}

// SetRealnameVerifyEnabledTx 在指定数据库连接或事务中设置全局实名认证开关。
func SetRealnameVerifyEnabledTx(ctx context.Context, db *gorm.DB, enabled bool, updatedBy uint) error {
	if db == nil {
		return errors.New("mysql datasource not initialized")
	}

	return upsertSystemSettingTx(ctx, db, realnameVerifyEnabledSettingKey, strconv.FormatBool(enabled), updatedBy)
}
