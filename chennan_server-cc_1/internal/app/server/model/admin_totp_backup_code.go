package model

import "time"

// AdminTOTPBackupCode TOTP 备份码模型。
type AdminTOTPBackupCode struct {
	ID       uint   `gorm:"primaryKey"`
	AdminID  uint   `gorm:"index;not null"`
	CodeHash string `gorm:"size:255;not null"`
	Used     bool   `gorm:"default:false;index"`
	UsedAt   *time.Time

	CreatedAt time.Time
}
