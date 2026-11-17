package model

import "time"

// Report 用户举报记录。
type Report struct {
	ID           uint   `gorm:"primaryKey"`
	ReporterID   uint   `gorm:"not null;index"`
	ReporterName string `gorm:"size:64"`
	TargetID     uint   `gorm:"index"`                  // 被举报对象数据库 ID（用户/帖子/评论）
	TargetIMID   string `gorm:"size:128;index"`         // userID、groupID 或内容侧字符串 ID
	TargetType   string `gorm:"size:32;default:'user'"` // "user" | "group" | "community_post" | "community_comment"
	TargetName   string `gorm:"size:64"`
	Category     string `gorm:"type:enum('spam','abuse','fraud','other');default:'other'"`
	Description  string `gorm:"type:text"`
	Status       string `gorm:"type:enum('pending','reviewed','dismissed');default:'pending'"`
	OwnerID      uint   `gorm:"index"` // reporter 所属的助理 ID，用于助理视角过滤

	CreatedAt time.Time
	UpdatedAt time.Time
}
