package model

import "time"

// LoginRiskControl records account/IP login risk states for admin handling.
type LoginRiskControl struct {
	ID                uint   `gorm:"primaryKey"`
	Scope             string `gorm:"size:16;not null;index:idx_login_risk_scope_status,priority:1"`
	RiskKey           string `gorm:"size:128;not null;uniqueIndex:uk_login_risk_key"`
	Username          string `gorm:"size:64;not null;default:'';index"`
	IMUserID          string `gorm:"size:64;not null;default:'';index"`
	UserID            *uint  `gorm:"index"`
	IPAddress         string `gorm:"size:64;not null;default:'';index"`
	Status            string `gorm:"size:32;not null;default:'watching';index:idx_login_risk_scope_status,priority:2"`
	Level             string `gorm:"size:16;not null;default:'low'"`
	Reason            string `gorm:"size:512;not null;default:''"`
	FailureCount      int    `gorm:"not null;default:0"`
	SuccessCount      int    `gorm:"not null;default:0"`
	WindowStartedAt   *time.Time
	CooldownUntil     *time.Time
	ResolvedByAdminID *uint
	ResolvedAt        *time.Time
	ResolveReason     string `gorm:"size:512;not null;default:''"`
	CreatedAt         time.Time
	UpdatedAt         time.Time `gorm:"index"`
}

// IPAccessBlock stores application-layer IP blocks. It intentionally does not
// change firewall rules.
type IPAccessBlock struct {
	ID               uint   `gorm:"primaryKey"`
	IPAddress        string `gorm:"size:64;not null;uniqueIndex:uk_ip_access_blocks_ip_address"`
	Status           string `gorm:"size:16;not null;default:'active';index"`
	Reason           string `gorm:"size:512;not null;default:''"`
	TriggerUsername  string `gorm:"size:64;not null;default:'';index"`
	TriggerFailCount int    `gorm:"not null;default:0"`
	TriggerWindow    string `gorm:"size:32;not null;default:''"`
	CreatedByAdminID uint   `gorm:"not null;default:0"`
	RemovedByAdminID *uint
	RemovedAt        *time.Time
	RemoveReason     string `gorm:"size:512;not null;default:''"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CleanupUserWhitelist protects users from automatic cleanup.
type CleanupUserWhitelist struct {
	ID               uint   `gorm:"primaryKey"`
	UserID           *uint  `gorm:"index"`
	Phone            string `gorm:"size:64;not null;default:'';index"`
	IMUserID         string `gorm:"size:64;not null;default:'';index"`
	Nickname         string `gorm:"size:255;not null;default:''"`
	OwnerID          *uint  `gorm:"index"`
	Reason           string `gorm:"size:512;not null;default:''"`
	Status           string `gorm:"size:32;not null;default:'active';index"`
	CreatedByAdminID *uint
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CleanupUserDisposition records cleanup/manual ban/restore disposition.
type CleanupUserDisposition struct {
	ID               uint   `gorm:"primaryKey"`
	UserID           uint   `gorm:"not null;uniqueIndex:uk_cleanup_dispositions_user_id"`
	IMUserID         string `gorm:"size:64;not null;default:'';index"`
	Phone            string `gorm:"size:64;not null;default:''"`
	Nickname         string `gorm:"size:255;not null;default:''"`
	OwnerID          *uint  `gorm:"index"`
	CleanupStatus    string `gorm:"size:32;not null;index"`
	Reason           string `gorm:"size:512;not null;default:''"`
	CreatedByAdminID *uint
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
