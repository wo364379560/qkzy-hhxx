package model

import "time"

// ChatAttachment 记录聊天大文件 COS 对象与会话归属。
type ChatAttachment struct {
	ID               uint   `gorm:"primaryKey"`
	FileID           string `gorm:"size:64;uniqueIndex;not null"`
	DownloadToken    string `gorm:"size:96;index;not null"`
	UploaderUserID   uint   `gorm:"index;not null"`
	UploaderIMUserID string `gorm:"size:64;index;not null"`
	ConversationID   string `gorm:"size:128;index;not null"`
	ConversationType string `gorm:"size:16;index;not null"`
	PeerUserID       string `gorm:"size:64;index"`
	GroupID          string `gorm:"size:64;index"`
	ObjectKey        string `gorm:"size:512;not null"`
	FileName         string `gorm:"size:255;not null"`
	FileSize         int64  `gorm:"not null"`
	ContentType      string `gorm:"size:128;not null;default:'application/octet-stream'"`
	IsVideo          bool   `gorm:"index;not null;default:false"`
	Status           string `gorm:"size:16;index;not null;default:'pending'"`
	ExpiresAt        *time.Time
	UploadedAt       *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}
