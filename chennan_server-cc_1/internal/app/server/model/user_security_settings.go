package model

import "time"

// UserSecuritySettings stores per-user account protection preferences.
type UserSecuritySettings struct {
	ID                uint `gorm:"primaryKey"`
	UserID            uint `gorm:"uniqueIndex;not null"`
	RemoteLoginAlert  bool `gorm:"default:true;not null"`
	LoginVerification bool `gorm:"default:false;not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// DefaultUserSecuritySettings returns in-memory defaults when no row exists.
func DefaultUserSecuritySettings(userID uint) *UserSecuritySettings {
	return &UserSecuritySettings{
		UserID:            userID,
		RemoteLoginAlert:  true,
		LoginVerification: false,
	}
}
