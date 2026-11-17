package model

import "time"

// FriendOwnerOperation is a durable retry/outbox and result audit. Manual retries
// retain previous results in RetryHistory before re-evaluating current owners.
type FriendOwnerOperation struct {
	ID             uint64 `gorm:"primaryKey"`
	OperationID    string `gorm:"size:64;not null;uniqueIndex"`
	EventKey       string `gorm:"size:64;not null;uniqueIndex"`
	Source         string `gorm:"size:32;not null"`
	Caller         string `gorm:"size:64;not null"`
	Target         string `gorm:"size:64;not null"`
	State          string `gorm:"size:16;not null;index:idx_owner_work,priority:1"`
	ReasonCode     string `gorm:"size:64;not null;default:''"`
	Attempts       int    `gorm:"not null;default:0"`
	ManualRetries  int    `gorm:"not null;default:0"`
	LastRetriedBy  uint
	LastRetriedAt  *time.Time
	RetryHistory   string     `gorm:"type:json;default:null"`
	AvailableAt    time.Time  `gorm:"not null;index:idx_owner_work,priority:2"`
	LeaseUntil     *time.Time `gorm:"index"`
	LeaseToken     string     `gorm:"size:64;not null;default:''"`
	EffectsPending bool       `gorm:"not null;default:false;index"`
	NeedsReview    bool       `gorm:"not null;default:false;index"`
	BeforeJSON     string     `gorm:"type:json;default:null"`
	AfterJSON      string     `gorm:"type:json;default:null"`
	ActorID        uint
	ResultJSON     string `gorm:"type:json;default:null"`
	LastError      string `gorm:"size:128;not null;default:''"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
