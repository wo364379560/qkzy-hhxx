package model

import "time"

// UserOperationLog records important user-side actions that must remain
// traceable even after the user row is deleted.
type UserOperationLog struct {
	ID              uint   `gorm:"primaryKey"`
	UserID          *uint  `gorm:"index"`
	IMUserID        string `gorm:"size:64;not null;default:'';index"`
	Phone           string `gorm:"size:32;not null;default:'';index"`
	Nickname        string `gorm:"size:255;not null;default:''"`
	OwnerID         *uint  `gorm:"index"`
	OwnerName       string `gorm:"size:64;not null;default:'';index"`
	OperationType   string `gorm:"size:64;not null;index:idx_user_operation_type_status_created,priority:1"`
	OperationName   string `gorm:"size:128;not null;default:''"`
	OperationDetail string `gorm:"size:512;not null;default:''"`
	IPAddress       string `gorm:"size:64;not null;default:'';index"`
	UserAgent       string `gorm:"size:512;not null;default:''"`
	DeviceID        string `gorm:"size:128;not null;default:'';index"`
	DeviceName      string `gorm:"size:128;not null;default:''"`
	Platform        string `gorm:"size:32;not null;default:''"`
	AppVersion      string `gorm:"size:64;not null;default:''"`
	Status          string `gorm:"size:16;not null;default:'success';index:idx_user_operation_type_status_created,priority:2"`
	FailureReason   string `gorm:"size:512;not null;default:''"`

	CreatedAt time.Time `gorm:"index:idx_user_operation_type_status_created,priority:3"`
}
