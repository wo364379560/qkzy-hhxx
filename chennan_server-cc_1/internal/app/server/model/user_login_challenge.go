package model

import "time"

const (
	UserLoginChallengeStatusPending  = "pending"
	UserLoginChallengeStatusApproved = "approved"
	UserLoginChallengeStatusDenied   = "denied"
	UserLoginChallengeStatusExpired  = "expired"
	UserLoginChallengeStatusConsumed = "consumed"
)

// UserLoginChallenge 表示新设备登录待授权挑战。
type UserLoginChallenge struct {
	ID               uint       `gorm:"primaryKey"`
	UserID           uint       `gorm:"index:idx_user_login_challenges_user_status;not null"`
	DeviceID         string     `gorm:"size:128;not null;default:''"`
	DeviceName       string     `gorm:"size:128;not null;default:''"`
	Platform         string     `gorm:"size:32;not null;default:''"`
	IP               string     `gorm:"size:64;not null;default:''"`
	UserAgent        string     `gorm:"size:512;not null;default:''"`
	Status           string     `gorm:"size:16;not null;default:'pending';index:idx_user_login_challenges_user_status"`
	PreAuthTokenHash string     `gorm:"size:128;not null;default:'';index"`
	ShortCode        string     `gorm:"size:16;not null;default:''"`
	ApproverDeviceID string     `gorm:"size:128;not null;default:''"`
	ExpiresAt        time.Time  `gorm:"not null;index"`
	ConsumedAt       *time.Time `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
