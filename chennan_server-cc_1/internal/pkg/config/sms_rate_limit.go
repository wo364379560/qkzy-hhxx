package config

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/gorm"
)

const smsRateLimitSettingKey = "sms_rate_limit_policy"

const (
	SMSRateLimitStrategyOff       = "off"
	SMSRateLimitStrategyRiskBased = "risk_based"
	SMSRateLimitStrategyAlwaysOn  = "always_on"
)

type SMSRateLimitSettings struct {
	Enabled                    bool   `json:"enabled"`
	StrategyMode               string `json:"strategy_mode"`
	AnomalyWindowMinutes       int    `json:"anomaly_window_minutes"`
	AnomalyTotalLimit          int    `json:"anomaly_total_limit"`
	AnomalyLoginLimit          int    `json:"anomaly_login_limit"`
	AnomalyRegisterLimit       int    `json:"anomaly_register_limit"`
	AnomalyResetLimit          int    `json:"anomaly_reset_limit"`
	PhoneCooldownSeconds       int    `json:"phone_cooldown_seconds"`
	PhoneHourLimit             int    `json:"phone_hour_limit"`
	PhoneDayLimit              int    `json:"phone_day_limit"`
	IPHourLimit                int    `json:"ip_hour_limit"`
	IPDayLimit                 int    `json:"ip_day_limit"`
	DeviceHourLimit            int    `json:"device_hour_limit"`
	DeviceDayLimit             int    `json:"device_day_limit"`
	StrictPhoneCooldownSeconds int    `json:"strict_phone_cooldown_seconds"`
	StrictPhoneHourLimit       int    `json:"strict_phone_hour_limit"`
	StrictPhoneDayLimit        int    `json:"strict_phone_day_limit"`
	StrictIPHourLimit          int    `json:"strict_ip_hour_limit"`
	StrictIPDayLimit           int    `json:"strict_ip_day_limit"`
	StrictDeviceHourLimit      int    `json:"strict_device_hour_limit"`
	StrictDeviceDayLimit       int    `json:"strict_device_day_limit"`
}

func DefaultSMSRateLimitSettings() SMSRateLimitSettings {
	return SMSRateLimitSettings{
		Enabled:                    true,
		StrategyMode:               SMSRateLimitStrategyOff,
		AnomalyWindowMinutes:       10,
		AnomalyTotalLimit:          100,
		AnomalyLoginLimit:          60,
		AnomalyRegisterLimit:       40,
		AnomalyResetLimit:          30,
		PhoneCooldownSeconds:       60,
		PhoneHourLimit:             5,
		PhoneDayLimit:              10,
		IPHourLimit:                20,
		IPDayLimit:                 50,
		DeviceHourLimit:            10,
		DeviceDayLimit:             30,
		StrictPhoneCooldownSeconds: 180,
		StrictPhoneHourLimit:       2,
		StrictPhoneDayLimit:        5,
		StrictIPHourLimit:          5,
		StrictIPDayLimit:           15,
		StrictDeviceHourLimit:      3,
		StrictDeviceDayLimit:       10,
	}
}

func GetSMSRateLimitSettings(ctx context.Context, db *gorm.DB) (SMSRateLimitSettings, error) {
	if db == nil {
		return SMSRateLimitSettings{}, errors.New("mysql datasource not initialized")
	}

	settings := DefaultSMSRateLimitSettings()

	var record struct {
		Value string `gorm:"column:value"`
	}
	err := db.WithContext(ctx).
		Table("system_settings").
		Select("value").
		Where("setting_key = ?", smsRateLimitSettingKey).
		Take(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return settings, nil
		}
		return SMSRateLimitSettings{}, err
	}

	if strings.TrimSpace(record.Value) == "" {
		return settings, nil
	}
	if err := json.Unmarshal([]byte(record.Value), &settings); err != nil {
		return SMSRateLimitSettings{}, err
	}
	return normalizeSMSRateLimitSettings(settings), nil
}

func SetSMSRateLimitSettingsTx(ctx context.Context, db *gorm.DB, settings SMSRateLimitSettings, updatedBy uint) error {
	if db == nil {
		return errors.New("mysql datasource not initialized")
	}
	settings = normalizeSMSRateLimitSettings(settings)
	payload, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return upsertSystemSettingTx(ctx, db, smsRateLimitSettingKey, string(payload), updatedBy)
}

func normalizeSMSRateLimitSettings(settings SMSRateLimitSettings) SMSRateLimitSettings {
	defaults := DefaultSMSRateLimitSettings()
	switch settings.StrategyMode {
	case SMSRateLimitStrategyOff, SMSRateLimitStrategyRiskBased, SMSRateLimitStrategyAlwaysOn:
	default:
		settings.StrategyMode = defaults.StrategyMode
	}
	if settings.AnomalyWindowMinutes <= 0 {
		settings.AnomalyWindowMinutes = defaults.AnomalyWindowMinutes
	}
	if settings.AnomalyTotalLimit <= 0 {
		settings.AnomalyTotalLimit = defaults.AnomalyTotalLimit
	}
	if settings.AnomalyLoginLimit <= 0 {
		settings.AnomalyLoginLimit = defaults.AnomalyLoginLimit
	}
	if settings.AnomalyRegisterLimit <= 0 {
		settings.AnomalyRegisterLimit = defaults.AnomalyRegisterLimit
	}
	if settings.AnomalyResetLimit <= 0 {
		settings.AnomalyResetLimit = defaults.AnomalyResetLimit
	}
	if settings.PhoneCooldownSeconds <= 0 {
		settings.PhoneCooldownSeconds = defaults.PhoneCooldownSeconds
	}
	if settings.PhoneHourLimit <= 0 {
		settings.PhoneHourLimit = defaults.PhoneHourLimit
	}
	if settings.PhoneDayLimit <= 0 {
		settings.PhoneDayLimit = defaults.PhoneDayLimit
	}
	if settings.IPHourLimit <= 0 {
		settings.IPHourLimit = defaults.IPHourLimit
	}
	if settings.IPDayLimit <= 0 {
		settings.IPDayLimit = defaults.IPDayLimit
	}
	if settings.DeviceHourLimit <= 0 {
		settings.DeviceHourLimit = defaults.DeviceHourLimit
	}
	if settings.DeviceDayLimit <= 0 {
		settings.DeviceDayLimit = defaults.DeviceDayLimit
	}
	if settings.StrictPhoneCooldownSeconds <= 0 {
		settings.StrictPhoneCooldownSeconds = defaults.StrictPhoneCooldownSeconds
	}
	if settings.StrictPhoneHourLimit <= 0 {
		settings.StrictPhoneHourLimit = defaults.StrictPhoneHourLimit
	}
	if settings.StrictPhoneDayLimit <= 0 {
		settings.StrictPhoneDayLimit = defaults.StrictPhoneDayLimit
	}
	if settings.StrictIPHourLimit <= 0 {
		settings.StrictIPHourLimit = defaults.StrictIPHourLimit
	}
	if settings.StrictIPDayLimit <= 0 {
		settings.StrictIPDayLimit = defaults.StrictIPDayLimit
	}
	if settings.StrictDeviceHourLimit <= 0 {
		settings.StrictDeviceHourLimit = defaults.StrictDeviceHourLimit
	}
	if settings.StrictDeviceDayLimit <= 0 {
		settings.StrictDeviceDayLimit = defaults.StrictDeviceDayLimit
	}
	return settings
}
