package repository

import (
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserDeviceRepository 管理用户设备身份表。
type UserDeviceRepository struct {
	db *gorm.DB
}

func NewUserDeviceRepository() *UserDeviceRepository {
	return &UserDeviceRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func NewUserDeviceRepositoryWithDB(db *gorm.DB) *UserDeviceRepository {
	return &UserDeviceRepository{db: db}
}

// UpsertLogin 登录成功时写入/更新设备行（UNIQUE user_id+device_id）。
func (r *UserDeviceRepository) UpsertLogin(userID uint, deviceID, deviceName, platform, appVersion, ip string, at time.Time) error {
	deviceID = strings.TrimSpace(deviceID)
	if userID == 0 || deviceID == "" {
		return nil
	}
	row := model.UserDevice{
		UserID:       userID,
		DeviceID:     deviceID,
		Platform:     strings.TrimSpace(platform),
		DeviceName:   strings.TrimSpace(deviceName),
		AppVersion:   strings.TrimSpace(appVersion),
		LastIP:       strings.TrimSpace(ip),
		LastLoginAt:  at,
		LastOnlineAt: at,
		Status:       model.UserDeviceStatusOnline,
		CreatedAt:    at,
		UpdatedAt:    at,
	}
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "device_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"platform":       row.Platform,
			"device_name":    row.DeviceName,
			"app_version":    row.AppVersion,
			"last_ip":        row.LastIP,
			"last_login_at":  at,
			"last_online_at": at,
			"status":         model.UserDeviceStatusOnline,
			"updated_at":     at,
		}),
	}).Create(&row).Error
}

func (r *UserDeviceRepository) ListByUserID(userID uint) ([]model.UserDevice, error) {
	var rows []model.UserDevice
	err := r.db.Where("user_id = ?", userID).
		Order("last_login_at DESC").
		Find(&rows).Error
	return rows, err
}

func (r *UserDeviceRepository) MarkOffline(userID uint, deviceID string, at time.Time) error {
	deviceID = strings.TrimSpace(deviceID)
	if userID == 0 || deviceID == "" {
		return nil
	}
	return r.db.Model(&model.UserDevice{}).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		Updates(map[string]any{
			"status":     model.UserDeviceStatusOffline,
			"updated_at": at,
		}).Error
}

func (r *UserDeviceRepository) Delete(userID uint, deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if userID == 0 || deviceID == "" {
		return nil
	}
	return r.db.Where("user_id = ? AND device_id = ?", userID, deviceID).
		Delete(&model.UserDevice{}).Error
}

// DeleteOthersSameFingerprint 删除同平台+同 IP 的其它设备行（卸载重装后清幽灵，列表只留一台）。
func (r *UserDeviceRepository) DeleteOthersSameFingerprint(userID uint, keepDeviceID, platform, ip string) error {
	keepDeviceID = strings.TrimSpace(keepDeviceID)
	platform = strings.TrimSpace(platform)
	ip = strings.TrimSpace(ip)
	if userID == 0 || keepDeviceID == "" || platform == "" || ip == "" {
		return nil
	}
	return r.db.Where(
		"user_id = ? AND device_id <> ? AND platform = ? AND last_ip = ?",
		userID, keepDeviceID, platform, ip,
	).Delete(&model.UserDevice{}).Error
}

// Do not mark a newer login offline. The session cutoff and last-login guard
// are part of the UPDATE itself, not a racy read/count followed by an update.
func (r *UserDeviceRepository) MarkOfflineThroughSession(userID uint, deviceID string, sessionID uint, loginCutoff, at time.Time) error {
	deviceID = strings.TrimSpace(deviceID)
	if userID == 0 || deviceID == "" || loginCutoff.IsZero() {
		return nil
	}
	newerSession := r.db.Model(&model.UserRefreshSession{}).Select("1").
		Where("user_id = ? AND device_id = ? AND id > ?", userID, deviceID, sessionID)
	return r.db.Model(&model.UserDevice{}).
		Where("user_id = ? AND device_id = ? AND last_login_at <= ?", userID, deviceID, loginCutoff).
		Where("NOT EXISTS (?)", newerSession).
		Updates(map[string]any{"status": model.UserDeviceStatusOffline, "updated_at": at}).Error
}
