package model

import "time"

// InviteCode 邀请码模型。
type InviteCode struct {
	ID        uint   `gorm:"primaryKey"`
	Code      string `gorm:"uniqueIndex;size:32;not null"`
	OwnerID   uint   `gorm:"not null"`
	MaxUses   int    `gorm:"default:0"`
	UsedCount int    `gorm:"default:0"`
	ExpiredAt *time.Time
	Status    string `gorm:"type:enum('active','used','expired');default:'active'"`

	CreatedAt time.Time
}
