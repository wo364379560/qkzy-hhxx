package model

import "time"

// UserTrustedDevice 永久信任设备：开启「新设备登录验证」后，信任设备可跳过挑战。
type UserTrustedDevice struct {
	ID         uint   `gorm:"primaryKey"`
	UserID     uint   `gorm:"uniqueIndex:uk_user_trusted_device;not null"`
	DeviceID   string `gorm:"size:128;uniqueIndex:uk_user_trusted_device;not null;default:''"`
	DeviceName string `gorm:"size:128;not null;default:''"`
	Platform   string `gorm:"size:32;not null;default:''"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
