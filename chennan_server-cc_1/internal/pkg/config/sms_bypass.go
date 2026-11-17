package config

import (
	"context"
	"errors"
	"strconv"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	globalBypassSMSEnabledSettingKey = "global_bypass_sms_enabled"
	globalBypassSMSCodeSettingKey    = "global_bypass_sms_code"
)

// SMSBypassSettings 表示全局万能验证码配置。
type SMSBypassSettings struct {
	Enabled bool
	Code    string
}

// GetSMSBypassSettings 获取全局万能验证码配置。
func GetSMSBypassSettings(ctx context.Context, db *gorm.DB) (SMSBypassSettings, error) {
	if db == nil {
		return SMSBypassSettings{}, errors.New("mysql datasource not initialized")
	}

	settings := SMSBypassSettings{}

	var enabledSetting model.SystemSetting
	err := db.WithContext(ctx).
		Where("setting_key = ?", globalBypassSMSEnabledSettingKey).
		Take(&enabledSetting).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return SMSBypassSettings{}, err
	}
	if err == nil {
		enabled, parseErr := strconv.ParseBool(enabledSetting.Value)
		if parseErr != nil {
			return SMSBypassSettings{}, parseErr
		}
		settings.Enabled = enabled
	}

	var codeSetting model.SystemSetting
	err = db.WithContext(ctx).
		Where("setting_key = ?", globalBypassSMSCodeSettingKey).
		Take(&codeSetting).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return SMSBypassSettings{}, err
	}
	if err == nil {
		settings.Code = codeSetting.Value
	}

	return settings, nil
}

// SetSMSBypassSettingsTx 在事务中保存全局万能验证码配置。
func SetSMSBypassSettingsTx(ctx context.Context, db *gorm.DB, enabled bool, code string, updatedBy uint) error {
	if db == nil {
		return errors.New("mysql datasource not initialized")
	}

	if err := upsertSystemSettingTx(ctx, db, globalBypassSMSEnabledSettingKey, strconv.FormatBool(enabled), updatedBy); err != nil {
		return err
	}
	return upsertSystemSettingTx(ctx, db, globalBypassSMSCodeSettingKey, code, updatedBy)
}

func upsertSystemSettingTx(ctx context.Context, db *gorm.DB, key, value string, updatedBy uint) error {
	setting := &model.SystemSetting{
		SettingKey: key,
		Value:      value,
	}
	if updatedBy > 0 {
		setting.UpdatedBy = &updatedBy
	}

	return db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "setting_key"}},
			DoUpdates: clause.Assignments(map[string]any{
				"value":      setting.Value,
				"updated_by": setting.UpdatedBy,
			}),
		}).
		Create(setting).Error
}
