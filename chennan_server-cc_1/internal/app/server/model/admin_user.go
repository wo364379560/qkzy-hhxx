package model

import "time"

// AdminUser 后台管理员/助理账号模型。
type AdminUser struct {
	ID                 uint   `gorm:"primaryKey"`
	Username           string `gorm:"uniqueIndex;size:64;not null"`
	DisplayName        string `gorm:"size:64;not null;default:''"`
	PasswordHash       string `gorm:"size:255;not null"`
	Role               string `gorm:"type:enum('super_admin','admin','assistant');default:'assistant'"`
	ParentAdminID      *uint  `gorm:"index"`
	CreatedBy          uint
	Status             string `gorm:"type:enum('active','banned');default:'active'"`
	AuthVersion        int    `gorm:"default:1"`
	IPWhitelistEnabled bool   `gorm:"default:false"`

	TOTPSecret  string `gorm:"size:255"`
	TOTPEnabled bool   `gorm:"default:false"`
	TOTPBoundAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// AdminUserIPWhitelist 是后台助理账号的 IP 白名单规则。
type AdminUserIPWhitelist struct {
	ID          uint   `gorm:"primaryKey"`
	AdminUserID uint   `gorm:"not null;uniqueIndex:uk_admin_user_cidr"`
	CIDR        string `gorm:"column:cidr;size:64;not null;uniqueIndex:uk_admin_user_cidr"`
	Note        string `gorm:"size:255;not null;default:''"`
	CreatedBy   uint

	CreatedAt time.Time
	UpdatedAt time.Time
}

// AdminAccessAudit 是后台访问审计记录。
type AdminAccessAudit struct {
	ID                uint   `gorm:"primaryKey"`
	AdminUserID       *uint  `gorm:"index"`
	AdminUsername     string `gorm:"size:64;index"`
	AdminRole         string `gorm:"size:32;index"`
	AccessType        string `gorm:"size:32;index"`
	ActionKey         string `gorm:"size:64;index"`
	ActionSummary     string `gorm:"size:128"`
	ActionDetail      string `gorm:"size:255"`
	Method            string `gorm:"size:16;index"`
	Route             string `gorm:"size:191;index"`
	TargetID          *uint  `gorm:"index"`
	TargetType        string `gorm:"size:32;index"`
	TargetUsername    string `gorm:"size:64;index"`
	TargetDisplayName string `gorm:"size:128"`
	ChangeSetJSON     string `gorm:"type:json"`
	Success           bool   `gorm:"index"`
	StatusCode        int
	IPAddress         string `gorm:"size:64"`
	UserAgent         string `gorm:"size:255"`
	FailureReason     string `gorm:"size:255"`

	CreatedAt time.Time
}

// AdminLoginAudit 是后台登录审计记录。
type AdminLoginAudit struct {
	ID            uint   `gorm:"primaryKey"`
	AdminUserID   *uint  `gorm:"index"`
	Username      string `gorm:"size:64;index"`
	AdminRole     string `gorm:"size:32;index"`
	Stage         string `gorm:"size:32;index"`
	Result        string `gorm:"size:32;index"`
	FailureReason string `gorm:"size:255"`
	IPAddress     string `gorm:"size:64"`
	UserAgent     string `gorm:"size:255"`

	CreatedAt time.Time
}
