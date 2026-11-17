package model

import "time"

// User 镜像腾讯云 IM 账号的本地用户模型。
type User struct {
	ID            uint   `gorm:"primaryKey"`
	BizID         int64  `gorm:"uniqueIndex;not null"` // 业务 ID（igs.Snowflake ID）
	Phone         string `gorm:"uniqueIndex;size:32;not null"`
	PasswordHash  string `gorm:"size:255;not null"`
	OwnerID       uint
	OwnerRevision uint64 `gorm:"not null;default:0"`
	CreatedBy     uint
	Status        string `gorm:"type:enum('active','banned','deleted');default:'active'"`

	// 腾讯云 IM 字段
	IMUserID      string  `gorm:"uniqueIndex;size:64"`
	WananID       *string `gorm:"uniqueIndex;size:32"`
	Nickname      string  `gorm:"size:64;not null"`
	AvatarURL     string  `gorm:"size:512"`
	Gender        int8
	Birthday      *time.Time
	Location      string `gorm:"size:128"`
	SelfSignature string `gorm:"size:255"`
	AllowType     int8

	CanCreateGroup           bool       `gorm:"default:false"`
	GloballySearchable       bool       `gorm:"default:false"` // 后台开关：可被其他服务空间精确搜索
	IsActivated              bool       `gorm:"default:false"` // 首登完成后置 true
	MustChangePassword       bool       `gorm:"default:false"` // 建号时置 true，改密后置 false
	AuthVersion              int        `gorm:"default:1"`     // 认证版本号，递增后可让旧 JWT 立即失效
	InviteCode               string     `gorm:"size:32"`       // 注册或后台建号时绑定的邀请码；助理建号会写入该助理当前有效邀请码
	UserTag                  string     `gorm:"size:32"`
	CommunityPostDisabled    bool       `gorm:"default:false"`
	CommunityCommentDisabled bool       `gorm:"default:false"`
	IPWhitelistEnabled       bool       `gorm:"default:false"`
	IPWhitelistIP            string     `gorm:"type:text"`
	LastOnlineAt             *time.Time `gorm:"index"`
	LastOfflineAt            *time.Time
	LastPresenceStatus       string `gorm:"size:16;default:unknown"`
	LastPresenceCheckedAt    *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserConversationCustomPin 表示用户自定义会话置顶。
type UserConversationCustomPin struct {
	ID             uint      `gorm:"primaryKey"`
	OwnerUserID    uint      `gorm:"uniqueIndex:uk_owner_conversation;index:idx_owner_pin_order,priority:1;index:idx_owner_updated_at,priority:1"`
	ConversationID string    `gorm:"size:128;not null;uniqueIndex:uk_owner_conversation"`
	PinOrder       int       `gorm:"not null;default:0;index:idx_owner_pin_order,priority:2"`
	PinnedAt       time.Time `gorm:"index:idx_owner_pin_order,priority:3"`
	CreatedAt      time.Time
	UpdatedAt      time.Time `gorm:"index:idx_owner_updated_at,priority:2"`
}
