package model

import "time"

// UserTOTPSetting stores a Web/App user's Google Authenticator binding.
type UserTOTPSetting struct {
	ID         uint   `gorm:"primaryKey"`
	UserID     uint   `gorm:"uniqueIndex;not null"`
	TOTPSecret string `gorm:"size:255;not null"`
	Enabled    bool   `gorm:"default:false;index"`
	BoundAt    *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserTOTPBackupCode stores one-time backup codes for user TOTP login.
type UserTOTPBackupCode struct {
	ID       uint   `gorm:"primaryKey"`
	UserID   uint   `gorm:"index;not null"`
	CodeHash string `gorm:"size:255;not null"`
	Used     bool   `gorm:"default:false;index"`
	UsedAt   *time.Time

	CreatedAt time.Time
}
