package model

import "time"

// Group 镜像腾讯云 IM 群组的本地群组模型。
type Group struct {
	ID             uint   `gorm:"primaryKey"`
	BizID          int64  `gorm:"uniqueIndex;not null"` // 业务 ID（igs.Snowflake ID）
	IMGroupID      string `gorm:"uniqueIndex;size:64"`
	Name           string `gorm:"size:128"`
	GroupType      string `gorm:"size:32;default:'Work'"`
	OwnerID        uint
	CreatedBy      uint
	OwnerBinding   string `gorm:"size:16;not null;default:legacy"`
	OwnerRevision  uint64 `gorm:"not null;default:0"`
	MirrorRevision uint64 `gorm:"not null;default:0"`
	Status         string `gorm:"type:enum('active','dismissed');default:'active'"`

	SlowModeEnabled bool  `gorm:"not null;default:false"` // 是否开启群发言频率限制
	SlowModeSeconds int   `gorm:"not null;default:0"`     // 普通成员两次发言间隔秒数
	SlowModeVersion int64 `gorm:"not null;default:0"`     // 慢速模式版本，用于 Redis key 隔离

	CreatedAt time.Time
}

// GroupNotice 是群公告扩展表。
type GroupNotice struct {
	Pinned           bool   `gorm:"not null;default:false" json:"pinned"`
	ID               uint   `gorm:"primaryKey"`
	GroupID          uint   `gorm:"uniqueIndex"`
	Content          string `gorm:"type:text;not null"`
	Summary          string `gorm:"size:255;not null;default:''"`
	UpdatedByAdminID *uint  `gorm:"index"`
	UpdatedByUserID  *uint  `gorm:"index"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// GroupMessagePin 是群内当前置顶消息扩展表。
type GroupMessagePin struct {
	ID                    uint      `gorm:"primaryKey"`
	GroupID               uint      `gorm:"index:idx_group_status,priority:1;not null"`
	IMGroupID             string    `gorm:"size:64;index"`
	MessageID             string    `gorm:"size:128;not null"`
	MessageSeq            int64     `gorm:"not null;default:0"`
	MessageSenderIMUserID string    `gorm:"size:64;not null;default:''"`
	MessageSenderName     string    `gorm:"size:128;not null;default:''"`
	MessageType           string    `gorm:"size:32;not null;default:''"`
	MessageAbstract       string    `gorm:"size:255;not null;default:''"`
	OperatorUserID        *uint     `gorm:"index"`
	Status                string    `gorm:"size:16;index:idx_group_status,priority:2;not null;default:'active'"`
	PinnedAt              time.Time `gorm:"index"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// GroupBusinessAdmin mirrors Tencent owner/admin roles and stores additional
// business administrator grants after the Tencent administrator quota is reached.
type GroupBusinessAdmin struct {
	ID        uint   `gorm:"primaryKey"`
	GroupID   string `gorm:"size:128;not null;uniqueIndex:uk_group_business_admin,priority:1;index:idx_group_business_admin_group_status,priority:1"`
	UserID    string `gorm:"size:64;not null;uniqueIndex:uk_group_business_admin,priority:2"`
	Role      string `gorm:"size:32;not null;default:'business_admin';index:idx_group_business_admin_group_status,priority:3"`
	Source    string `gorm:"size:32;not null;default:'tencent_overflow'"`
	Status    string `gorm:"size:16;not null;default:'active';index:idx_group_business_admin_group_status,priority:2"`
	CreatedBy string `gorm:"size:64;not null;default:''"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
