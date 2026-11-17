package model

import "time"

// UserUnverifiedMessageLimit stores the cumulative send quota used before real-name verification.
type UserUnverifiedMessageLimit struct {
	ID          uint   `gorm:"primaryKey"`
	UserID      uint   `gorm:"uniqueIndex;not null"`
	IMUserID    string `gorm:"size:64;not null;index"`
	SentCount   int64  `gorm:"not null;default:0"`
	FirstSentAt *time.Time
	LastSentAt  *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}
