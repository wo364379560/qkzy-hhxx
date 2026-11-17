package config

import (
	"context"
	"errors"
	"strconv"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"gorm.io/gorm"
)

const (
	antiFraudNoticeEnabledSettingKey      = "anti_fraud_notice_enabled"
	antiFraudNoticeContentSettingKey      = "anti_fraud_notice_content"
	antiFraudNoticeCountdownSecSettingKey = "anti_fraud_notice_countdown_sec"
	defaultAntiFraudNoticeCountdownSec    = 7
)

// AntiFraudNoticeSettings 防诈公告配置。
type AntiFraudNoticeSettings struct {
	Enabled       bool
	Content       string
	CountdownSec  int
}

// GetAntiFraudNoticeSettings 获取防诈公告配置。
func GetAntiFraudNoticeSettings(ctx context.Context, db *gorm.DB) (AntiFraudNoticeSettings, error) {
	if db == nil {
		return AntiFraudNoticeSettings{}, errors.New("mysql datasource not initialized")
	}

	settings := AntiFraudNoticeSettings{
		CountdownSec: defaultAntiFraudNoticeCountdownSec,
	}

	var enabledSetting model.SystemSetting
	err := db.WithContext(ctx).
		Where("setting_key = ?", antiFraudNoticeEnabledSettingKey).
		Take(&enabledSetting).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return AntiFraudNoticeSettings{}, err
	}
	if err == nil {
		enabled, parseErr := strconv.ParseBool(enabledSetting.Value)
		if parseErr != nil {
			return AntiFraudNoticeSettings{}, parseErr
		}
		settings.Enabled = enabled
	}

	var contentSetting model.SystemSetting
	err = db.WithContext(ctx).
		Where("setting_key = ?", antiFraudNoticeContentSettingKey).
		Take(&contentSetting).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return AntiFraudNoticeSettings{}, err
	}
	if err == nil {
		settings.Content = contentSetting.Value
	}

	var countdownSetting model.SystemSetting
	err = db.WithContext(ctx).
		Where("setting_key = ?", antiFraudNoticeCountdownSecSettingKey).
		Take(&countdownSetting).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return AntiFraudNoticeSettings{}, err
	}
	if err == nil {
		countdown, parseErr := strconv.Atoi(countdownSetting.Value)
		if parseErr != nil {
			return AntiFraudNoticeSettings{}, parseErr
		}
		if countdown > 0 {
			settings.CountdownSec = countdown
		}
	}

	return settings, nil
}

// SetAntiFraudNoticeSettingsTx 在事务中保存防诈公告配置。
func SetAntiFraudNoticeSettingsTx(ctx context.Context, db *gorm.DB, settings AntiFraudNoticeSettings, updatedBy uint) error {
	if db == nil {
		return errors.New("mysql datasource not initialized")
	}

	countdownSec := settings.CountdownSec
	if countdownSec <= 0 {
		countdownSec = defaultAntiFraudNoticeCountdownSec
	}

	if err := upsertSystemSettingTx(ctx, db, antiFraudNoticeEnabledSettingKey, strconv.FormatBool(settings.Enabled), updatedBy); err != nil {
		return err
	}
	if err := upsertSystemSettingTx(ctx, db, antiFraudNoticeContentSettingKey, settings.Content, updatedBy); err != nil {
		return err
	}
	return upsertSystemSettingTx(ctx, db, antiFraudNoticeCountdownSecSettingKey, strconv.Itoa(countdownSec), updatedBy)
}
