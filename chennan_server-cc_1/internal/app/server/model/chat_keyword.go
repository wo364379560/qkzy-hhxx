package model

import "time"

// ChatKeyword 表示聊天关键词屏蔽规则。
type ChatKeyword struct {
	ID          uint   `gorm:"primaryKey"`
	Keyword     string `gorm:"uniqueIndex;size:128;not null"`
	Replacement string `gorm:"size:128;not null;default:'**'"`
	Enabled     bool   `gorm:"not null;default:true"`
	CreatedBy   *uint
	UpdatedBy   *uint

	CreatedAt time.Time
	UpdatedAt time.Time
}
