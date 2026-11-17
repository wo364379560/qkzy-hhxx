package handler_admin

import (
	"encoding/json"

	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/handler/authlimit"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type setSMSRateLimitReq struct {
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
	CurrentPassword            string `json:"current_password" binding:"required"`
}

func GetSMSRateLimitSettings(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	result, err := controller_admin.GetSMSRateLimitSettings()
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

func SetSMSRateLimitSettings(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	var req setSMSRateLimitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	before, _ := controller_admin.GetSMSRateLimitSettings()
	if err := controller_admin.SetSMSRateLimitSettings(c.GetUint("admin_id"), req.CurrentPassword, controller_admin.SMSRateLimitSettingsResult{
		Enabled:                    req.Enabled,
		StrategyMode:               req.StrategyMode,
		AnomalyWindowMinutes:       req.AnomalyWindowMinutes,
		AnomalyTotalLimit:          req.AnomalyTotalLimit,
		AnomalyLoginLimit:          req.AnomalyLoginLimit,
		AnomalyRegisterLimit:       req.AnomalyRegisterLimit,
		AnomalyResetLimit:          req.AnomalyResetLimit,
		PhoneCooldownSeconds:       req.PhoneCooldownSeconds,
		PhoneHourLimit:             req.PhoneHourLimit,
		PhoneDayLimit:              req.PhoneDayLimit,
		IPHourLimit:                req.IPHourLimit,
		IPDayLimit:                 req.IPDayLimit,
		DeviceHourLimit:            req.DeviceHourLimit,
		DeviceDayLimit:             req.DeviceDayLimit,
		StrictPhoneCooldownSeconds: req.StrictPhoneCooldownSeconds,
		StrictPhoneHourLimit:       req.StrictPhoneHourLimit,
		StrictPhoneDayLimit:        req.StrictPhoneDayLimit,
		StrictIPHourLimit:          req.StrictIPHourLimit,
		StrictIPDayLimit:           req.StrictIPDayLimit,
		StrictDeviceHourLimit:      req.StrictDeviceHourLimit,
		StrictDeviceDayLimit:       req.StrictDeviceDayLimit,
	}); err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	middleware.SetAdminAccessTarget(c, "system_config", nil, "sms_rate_limit", "短信频控")
	middleware.SetAdminAccessAction(c, "config.update_sms_rate_limit", "修改短信频控")
	middleware.SetAdminAccessActionDetail(c, "修改短信发送频控阈值")
	if payload, err := json.Marshal(map[string]any{
		"enabled":                       map[string]any{"before": before.Enabled, "after": req.Enabled},
		"strategy_mode":                 map[string]any{"before": before.StrategyMode, "after": req.StrategyMode},
		"anomaly_window_minutes":        map[string]any{"before": before.AnomalyWindowMinutes, "after": req.AnomalyWindowMinutes},
		"anomaly_total_limit":           map[string]any{"before": before.AnomalyTotalLimit, "after": req.AnomalyTotalLimit},
		"anomaly_login_limit":           map[string]any{"before": before.AnomalyLoginLimit, "after": req.AnomalyLoginLimit},
		"anomaly_register_limit":        map[string]any{"before": before.AnomalyRegisterLimit, "after": req.AnomalyRegisterLimit},
		"anomaly_reset_limit":           map[string]any{"before": before.AnomalyResetLimit, "after": req.AnomalyResetLimit},
		"phone_cooldown_seconds":        map[string]any{"before": before.PhoneCooldownSeconds, "after": req.PhoneCooldownSeconds},
		"phone_hour_limit":              map[string]any{"before": before.PhoneHourLimit, "after": req.PhoneHourLimit},
		"phone_day_limit":               map[string]any{"before": before.PhoneDayLimit, "after": req.PhoneDayLimit},
		"ip_hour_limit":                 map[string]any{"before": before.IPHourLimit, "after": req.IPHourLimit},
		"ip_day_limit":                  map[string]any{"before": before.IPDayLimit, "after": req.IPDayLimit},
		"device_hour_limit":             map[string]any{"before": before.DeviceHourLimit, "after": req.DeviceHourLimit},
		"device_day_limit":              map[string]any{"before": before.DeviceDayLimit, "after": req.DeviceDayLimit},
		"strict_phone_cooldown_seconds": map[string]any{"before": before.StrictPhoneCooldownSeconds, "after": req.StrictPhoneCooldownSeconds},
		"strict_phone_hour_limit":       map[string]any{"before": before.StrictPhoneHourLimit, "after": req.StrictPhoneHourLimit},
		"strict_phone_day_limit":        map[string]any{"before": before.StrictPhoneDayLimit, "after": req.StrictPhoneDayLimit},
		"strict_ip_hour_limit":          map[string]any{"before": before.StrictIPHourLimit, "after": req.StrictIPHourLimit},
		"strict_ip_day_limit":           map[string]any{"before": before.StrictIPDayLimit, "after": req.StrictIPDayLimit},
		"strict_device_hour_limit":      map[string]any{"before": before.StrictDeviceHourLimit, "after": req.StrictDeviceHourLimit},
		"strict_device_day_limit":       map[string]any{"before": before.StrictDeviceDayLimit, "after": req.StrictDeviceDayLimit},
	}); err == nil {
		middleware.SetAdminAccessChangeSet(c, string(payload))
	}

	response.OK(c, nil)
}
