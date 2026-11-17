package model

import "time"

// OfflineActivity 表示线下活动。
type OfflineActivity struct {
	ID                uint      `gorm:"primaryKey"`
	Title             string    `gorm:"size:128;not null"`
	Subtitle          string    `gorm:"size:255"`
	Description       string    `gorm:"size:2000"`
	City              string    `gorm:"size:64;index"`
	Venue             string    `gorm:"size:255"`
	StartAt           time.Time `gorm:"index;not null"`
	EndAt             time.Time `gorm:"not null"`
	Quota             int       `gorm:"default:0"`               // 0 表示不限人数
	Status            string    `gorm:"size:16;default:'draft'"` // draft / published / closed / cancelled
	RegistrationCount int       `gorm:"default:0"`
	CreatedBy         uint      `gorm:"index;not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// OfflineActivityRegistration 表示线下活动报名记录。
type OfflineActivityRegistration struct {
	ID          uint      `gorm:"primaryKey"`
	ActivityID  uint      `gorm:"uniqueIndex:idx_activity_user_registration;not null"`
	UserID      uint      `gorm:"uniqueIndex:idx_activity_user_registration;not null"`
	Status      string    `gorm:"size:16;default:'registered'"` // registered / cancelled
	JoinedAt    time.Time `gorm:"not null"`
	CancelledAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}
