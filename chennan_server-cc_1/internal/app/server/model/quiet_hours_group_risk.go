package model

import "time"

// QuietHoursGroupRisk records groups created during quiet-hours restricted windows.
type QuietHoursGroupRisk struct {
	ID                  uint       `gorm:"primaryKey"`
	IMGroupID           string     `gorm:"size:64;uniqueIndex;not null"`
	GroupName           string     `gorm:"size:128;not null;default:''"`
	GroupType           string     `gorm:"size:32;not null;default:''"`
	LocalGroupID        *uint      `gorm:"index"`
	OwnerIMUserID       string     `gorm:"size:64;index;not null;default:''"`
	OwnerUserID         *uint      `gorm:"index"`
	OwnerID             *uint      `gorm:"index"`
	PolicyScope         string     `gorm:"size:32;index;not null;default:''"`
	PolicyAdminUserID   *uint      `gorm:"index"`
	PolicyStart         string     `gorm:"size:5;not null;default:''"`
	PolicyEnd           string     `gorm:"size:5;not null;default:''"`
	TencentCreateTime   *time.Time `gorm:"index"`
	TriggerSource       string     `gorm:"size:64;index;not null;default:''"`
	CallbackCommand     string     `gorm:"size:64;not null;default:''"`
	Action              string     `gorm:"size:64;index;not null;default:''"`
	Status              string     `gorm:"size:32;index;not null;default:'pending'"`
	Reason              string     `gorm:"size:512;not null;default:''"`
	TencentResult       string     `gorm:"type:text"`
	DryRun              bool       `gorm:"not null;default:false"`
	LastCheckedAt       *time.Time
	DismissedAt         *time.Time
	ResolvedAt          *time.Time
	ResolvedByAdminID   *uint  `gorm:"index"`
	ResolveReason       string `gorm:"size:512;not null;default:''"`
	ManualActionAdminID *uint  `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// QuietHoursGroupProtection protects a group from automatic quiet-hours dismissal.
type QuietHoursGroupProtection struct {
	ID               uint   `gorm:"primaryKey"`
	IMGroupID        string `gorm:"size:64;uniqueIndex;not null"`
	GroupName        string `gorm:"size:128;not null;default:''"`
	Reason           string `gorm:"size:512;not null;default:''"`
	Status           string `gorm:"size:32;index;not null;default:'active'"`
	CreatedByAdminID *uint  `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
