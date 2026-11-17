package controller_admin

import (
	"context"
	"errors"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
)

type SMSRateLimitSettingsResult struct {
	Enabled                    bool   `json:"enabled"`
	StrategyMode               string `json:"strategy_mode"`
	AnomalyWindowMinutes       int    `json:"anomaly_window_minutes"`
	AnomalyTotalLimit          int    `json:"anomaly_total_limit"`
	AnomalyLoginLimit          int    `json:"anomaly_login_limit"`
	AnomalyRegisterLimit       int    `json:"anomaly_register_limit"`
	AnomalyResetLimit          int    `json:"anomaly_reset_limit"`
	AnomalyTriggered           bool   `json:"anomaly_triggered"`
	AnomalyReason              string `json:"anomaly_reason"`
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

func GetSMSRateLimitSettings() (*SMSRateLimitSettingsResult, error) {
	settings, err := config.GetSMSRateLimitSettings(context.Background(), datasource.Datasource.Mysql.GetClient())
	if err != nil {
		return nil, err
	}
	return &SMSRateLimitSettingsResult{
		Enabled:                    settings.Enabled,
		StrategyMode:               settings.StrategyMode,
		AnomalyWindowMinutes:       settings.AnomalyWindowMinutes,
		AnomalyTotalLimit:          settings.AnomalyTotalLimit,
		AnomalyLoginLimit:          settings.AnomalyLoginLimit,
		AnomalyRegisterLimit:       settings.AnomalyRegisterLimit,
		AnomalyResetLimit:          settings.AnomalyResetLimit,
		AnomalyTriggered:           IsSMSRateLimitAnomalyTriggered(settings),
		AnomalyReason:              GetSMSRateLimitAnomalyReason(settings),
		PhoneCooldownSeconds:       settings.PhoneCooldownSeconds,
		PhoneHourLimit:             settings.PhoneHourLimit,
		PhoneDayLimit:              settings.PhoneDayLimit,
		IPHourLimit:                settings.IPHourLimit,
		IPDayLimit:                 settings.IPDayLimit,
		DeviceHourLimit:            settings.DeviceHourLimit,
		DeviceDayLimit:             settings.DeviceDayLimit,
		StrictPhoneCooldownSeconds: settings.StrictPhoneCooldownSeconds,
		StrictPhoneHourLimit:       settings.StrictPhoneHourLimit,
		StrictPhoneDayLimit:        settings.StrictPhoneDayLimit,
		StrictIPHourLimit:          settings.StrictIPHourLimit,
		StrictIPDayLimit:           settings.StrictIPDayLimit,
		StrictDeviceHourLimit:      settings.StrictDeviceHourLimit,
		StrictDeviceDayLimit:       settings.StrictDeviceDayLimit,
	}, nil
}

func SetSMSRateLimitSettings(adminID uint, currentPassword string, settings SMSRateLimitSettingsResult) error {
	if err := verifySuperAdminPassword(adminID, currentPassword); err != nil {
		return err
	}

	if settings.PhoneCooldownSeconds < 1 {
		return errors.New("手机号冷却时间必须大于 0")
	}
	if settings.AnomalyWindowMinutes < 1 || settings.AnomalyTotalLimit < 1 ||
		settings.AnomalyLoginLimit < 1 || settings.AnomalyRegisterLimit < 1 || settings.AnomalyResetLimit < 1 ||
		settings.PhoneHourLimit < 1 || settings.PhoneDayLimit < 1 ||
		settings.IPHourLimit < 1 || settings.IPDayLimit < 1 ||
		settings.DeviceHourLimit < 1 || settings.DeviceDayLimit < 1 ||
		settings.StrictPhoneCooldownSeconds < 1 ||
		settings.StrictPhoneHourLimit < 1 || settings.StrictPhoneDayLimit < 1 ||
		settings.StrictIPHourLimit < 1 || settings.StrictIPDayLimit < 1 ||
		settings.StrictDeviceHourLimit < 1 || settings.StrictDeviceDayLimit < 1 {
		return errors.New("短信频控阈值必须大于 0")
	}

	return config.SetSMSRateLimitSettingsTx(
		context.Background(),
		datasource.Datasource.Mysql.GetClient(),
		config.SMSRateLimitSettings{
			Enabled:                    settings.Enabled,
			StrategyMode:               settings.StrategyMode,
			AnomalyWindowMinutes:       settings.AnomalyWindowMinutes,
			AnomalyTotalLimit:          settings.AnomalyTotalLimit,
			AnomalyLoginLimit:          settings.AnomalyLoginLimit,
			AnomalyRegisterLimit:       settings.AnomalyRegisterLimit,
			AnomalyResetLimit:          settings.AnomalyResetLimit,
			PhoneCooldownSeconds:       settings.PhoneCooldownSeconds,
			PhoneHourLimit:             settings.PhoneHourLimit,
			PhoneDayLimit:              settings.PhoneDayLimit,
			IPHourLimit:                settings.IPHourLimit,
			IPDayLimit:                 settings.IPDayLimit,
			DeviceHourLimit:            settings.DeviceHourLimit,
			DeviceDayLimit:             settings.DeviceDayLimit,
			StrictPhoneCooldownSeconds: settings.StrictPhoneCooldownSeconds,
			StrictPhoneHourLimit:       settings.StrictPhoneHourLimit,
			StrictPhoneDayLimit:        settings.StrictPhoneDayLimit,
			StrictIPHourLimit:          settings.StrictIPHourLimit,
			StrictIPDayLimit:           settings.StrictIPDayLimit,
			StrictDeviceHourLimit:      settings.StrictDeviceHourLimit,
			StrictDeviceDayLimit:       settings.StrictDeviceDayLimit,
		},
		adminID,
	)
}

func IsSMSRateLimitAnomalyTriggered(settings config.SMSRateLimitSettings) bool {
	return GetSMSRateLimitAnomalyReason(settings) != ""
}

func GetSMSRateLimitAnomalyReason(settings config.SMSRateLimitSettings) string {
	repo := repository.NewSMSSendLogRepository()
	now := time.Now()
	windowStart := now.Add(-time.Duration(settings.AnomalyWindowMinutes) * time.Minute)

	total, err := repo.CountAllSince(windowStart)
	if err == nil && total >= int64(settings.AnomalyTotalLimit) {
		return "总请求量超阈值"
	}

	loginCount, err := repo.CountBySceneSince("login", windowStart)
	if err == nil && loginCount >= int64(settings.AnomalyLoginLimit) {
		return "登录场景超阈值"
	}

	registerCount, err := repo.CountBySceneSince("register", windowStart)
	if err == nil && registerCount >= int64(settings.AnomalyRegisterLimit) {
		return "注册场景超阈值"
	}

	resetCount, err := repo.CountBySceneSince("reset_password", windowStart)
	if err == nil && resetCount >= int64(settings.AnomalyResetLimit) {
		return "找回密码场景超阈值"
	}

	return ""
}
