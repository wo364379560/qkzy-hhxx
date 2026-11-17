package repository

import (
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserTrustedDeviceRepository 管理永久信任设备。
type UserTrustedDeviceRepository struct {
	db *gorm.DB
}

func NewUserTrustedDeviceRepository() *UserTrustedDeviceRepository {
	return &UserTrustedDeviceRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func (r *UserTrustedDeviceRepository) Exists(userID uint, deviceID string) (bool, error) {
	deviceID = strings.TrimSpace(deviceID)
	if userID == 0 || deviceID == "" {
		return false, nil
	}
	var count int64
	err := r.db.Model(&model.UserTrustedDevice{}).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		Count(&count).Error
	return count > 0, err
}

func (r *UserTrustedDeviceRepository) Upsert(userID uint, deviceID, deviceName, platform string) error {
	deviceID = strings.TrimSpace(deviceID)
	if userID == 0 || deviceID == "" {
		return nil
	}
	now := time.Now()
	row := model.UserTrustedDevice{
		UserID:     userID,
		DeviceID:   deviceID,
		DeviceName: strings.TrimSpace(deviceName),
		Platform:   strings.TrimSpace(platform),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "device_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"device_name": row.DeviceName,
			"platform":    row.Platform,
			"updated_at":  now,
		}),
	}).Create(&row).Error
}
