package controller_chat

import (
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/repository"
)

// SecuritySettingsResult 账号保护设置。
type SecuritySettingsResult struct {
	RemoteLoginAlert  bool `json:"remote_login_alert"`
	LoginVerification bool `json:"login_verification"`
}

// UpdateSecuritySettingsReq 增量更新账号保护设置。
type UpdateSecuritySettingsReq struct {
	RemoteLoginAlert  *bool
	LoginVerification *bool
	// 开启「新设备登录验证」时传入，用于把当前手机标为信任设备，避免本机退出再登仍被挑战。
	DeviceID   string
	DeviceName string
	Platform   string
}

// GetSecuritySettings 获取当前用户账号保护设置（无记录时返回默认值）。
func GetSecuritySettings(imUserID string) (*SecuritySettingsResult, error) {
	user, err := repository.NewUserRepository().FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}
	setting, err := repository.NewUserSecuritySettingsRepository().GetOrDefault(user.ID)
	if err != nil {
		return nil, err
	}
	return &SecuritySettingsResult{
		RemoteLoginAlert:  setting.RemoteLoginAlert,
		LoginVerification: setting.LoginVerification,
	}, nil
}

// UpdateSecuritySettings 更新当前用户账号保护设置。
func UpdateSecuritySettings(imUserID string, req UpdateSecuritySettingsReq) (*SecuritySettingsResult, error) {
	if req.RemoteLoginAlert == nil && req.LoginVerification == nil {
		return nil, AuthErr("请至少提供一个可更新字段")
	}
	user, err := repository.NewUserRepository().FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}
	setting, err := repository.NewUserSecuritySettingsRepository().Upsert(user.ID, req.RemoteLoginAlert, req.LoginVerification)
	if err != nil {
		return nil, err
	}
	// 开启登录验证时：信任本机，并清掉其它设备上的有效会话。
	// 否则同机残留的「幽灵 device_id」会在退出再登时被当成另一台设备，反复要求确认。
	if req.LoginVerification != nil && *req.LoginVerification {
		deviceID := normalizeDeviceID(req.DeviceID)
		if deviceID != "" {
			_ = repository.NewUserTrustedDeviceRepository().Upsert(
				user.ID,
				deviceID,
				strings.TrimSpace(req.DeviceName),
				strings.TrimSpace(req.Platform),
			)
			_, _ = repository.NewUserRefreshSessionRepository().RevokeOthersByUserID(user.ID, deviceID)
		}
	}
	return &SecuritySettingsResult{
		RemoteLoginAlert:  setting.RemoteLoginAlert,
		LoginVerification: setting.LoginVerification,
	}, nil
}

// normalizeDeviceID trims device id for revoke/list matching.
func normalizeDeviceID(deviceID string) string {
	return strings.TrimSpace(deviceID)
}

// NormalizeDeviceID exports device id normalization for handlers.
func NormalizeDeviceID(deviceID string) string {
	return normalizeDeviceID(deviceID)
}
