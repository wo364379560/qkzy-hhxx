package model

import "time"

const (
	UserDeviceStatusOnline  = "online"
	UserDeviceStatusOffline = "offline"
)

// UserDevice 用户设备身份（与 refresh session 分离）。
// 同一用户 + 同一 device_id 仅一条记录；登录时 upsert，不重复 insert。
type UserDevice struct {
	ID           uint      `gorm:"primaryKey"`
	UserID       uint      `gorm:"uniqueIndex:uk_user_device;not null"`
	DeviceID     string    `gorm:"size:128;uniqueIndex:uk_user_device;not null;default:''"`
	Platform     string    `gorm:"size:32;not null;default:''"`
	DeviceName   string    `gorm:"size:128;not null;default:''"`
	AppVersion   string    `gorm:"size:64;not null;default:''"`
	LastIP       string    `gorm:"size:64;not null;default:''"`
	LastLoginAt  time.Time `gorm:"not null;index"`
	LastOnlineAt time.Time `gorm:"not null;index"`
	Status       string    `gorm:"size:16;not null;default:'offline';index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
