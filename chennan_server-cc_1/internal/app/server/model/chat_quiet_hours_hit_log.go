package model

import "time"

// ChatQuietHoursHitLog records a user-side action blocked by quiet-hours policy.
type ChatQuietHoursHitLog struct {
	ID              uint   `gorm:"primaryKey"`
	Action          string `gorm:"size:32;index"`
	Source          string `gorm:"size:32;index"`
	CallbackCommand string `gorm:"size:64"`
	UserID          *uint  `gorm:"index"`
	OwnerID         *uint  `gorm:"index"`
	IMUserID        string `gorm:"size:64;index"`
	Phone           string `gorm:"size:32;index"`
	TargetAccount   string `gorm:"size:64"`
	TargetGroupID   string `gorm:"size:64"`
	IPAddress       string `gorm:"size:64"`
	UserAgent       string `gorm:"size:255"`
	Reason          string `gorm:"size:255"`

	CreatedAt time.Time
}
