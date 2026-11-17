package model

import "time"

const (
	UserRealNameStatusNone     = "none"
	UserRealNameStatusVerified = "verified"
	UserRealNameStatusFailed   = "failed"
)

// UserRealName 用户实名认证记录（与用户一对一）。
type UserRealName struct {
	ID              uint   `gorm:"primaryKey"`
	UserID          uint   `gorm:"uniqueIndex;not null"`
	RealName        string `gorm:"size:64;not null;default:''"`
	IDNumberHash    string `gorm:"size:64;not null;default:'';index"`
	IDNumberMasked  string `gorm:"size:32;not null;default:''"`
	Status          string `gorm:"size:16;not null;default:'none';index"`
	VerifiedAt      *time.Time
	AliyunRequestID string `gorm:"size:64;not null;default:''"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
