package model

import "time"

// UserRefreshSession 表示用户长期登录 refresh 会话。
type UserRefreshSession struct {
	ID               uint   `gorm:"primaryKey"`
	UserID           uint   `gorm:"index;not null"`
	RefreshTokenHash string `gorm:"size:64;uniqueIndex;not null"`
	// Only one prior rotation is recoverable, for a fixed short interval.
	PreviousRefreshTokenHash string     `gorm:"size:64;index" json:"-"`
	PreviousRefreshExpiresAt *time.Time `json:"-"`
	RefreshTokenCiphertext   string     `gorm:"size:512" json:"-"`
	DeviceID                 string     `gorm:"size:128"`
	DeviceName               string     `gorm:"size:128"`
	Platform                 string     `gorm:"size:32"`
	AppVersion               string     `gorm:"size:64"`
	IP                       string     `gorm:"size:64"`
	UserAgent                string     `gorm:"size:512"`
	LastUsedAt               time.Time  `gorm:"not null"`
	ExpiresAt                time.Time  `gorm:"index;not null"`
	RevokedAt                *time.Time `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
