package controller_admin

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
)

// ChatAttachmentUploadSettingsResult 是聊天大文件上传配置响应。
type ChatAttachmentUploadSettingsResult struct {
	Enabled          bool   `json:"enabled"`
	MaxSizeMB        int64  `json:"max_size_mb"`
	HardLimitMB      int64  `json:"hard_limit_mb"`
	NativeLimitMB    int64  `json:"native_limit_mb"`
	PutURLTTLSeconds int64  `json:"put_url_ttl_seconds"`
	GetURLTTLSeconds int64  `json:"get_url_ttl_seconds"`
	RetentionDays    int64  `json:"retention_days"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}

// GetChatAttachmentUploadSettings 获取聊天大文件上传配置。
func GetChatAttachmentUploadSettings() (*ChatAttachmentUploadSettingsResult, error) {
	settings, err := config.GetChatAttachmentUploadSettings(context.Background(), datasource.Datasource.Mysql.GetClient())
	if err != nil {
		return nil, err
	}
	return chatAttachmentUploadSettingsResult(settings), nil
}

// SetChatAttachmentUploadSettings 设置聊天大文件上传配置。
func SetChatAttachmentUploadSettings(adminID uint, currentPassword string, enabled bool, maxSizeMB int64) error {
	if err := verifySuperAdminPassword(adminID, currentPassword); err != nil {
		return err
	}
	if maxSizeMB <= 100 || maxSizeMB > 500 {
		return errors.New("最大文件大小必须在 101-500MB 之间")
	}

	db := datasource.Datasource.Mysql.GetClient()
	return db.Transaction(func(tx *gorm.DB) error {
		current, err := config.GetChatAttachmentUploadSettings(context.Background(), tx)
		if err != nil {
			return err
		}
		current.Enabled = enabled
		current.MaxSizeMB = maxSizeMB
		return config.SetChatAttachmentUploadSettingsTx(context.Background(), tx, current, adminID)
	})
}

func chatAttachmentUploadSettingsResult(settings config.ChatAttachmentUploadSettings) *ChatAttachmentUploadSettingsResult {
	updatedAt := ""
	if settings.UpdatedAt != nil && !settings.UpdatedAt.IsZero() {
		updatedAt = settings.UpdatedAt.In(time.Local).Format("2006-01-02 15:04:05")
	}
	return &ChatAttachmentUploadSettingsResult{
		Enabled:          settings.Enabled,
		MaxSizeMB:        settings.MaxSizeMB,
		HardLimitMB:      settings.HardLimitMB,
		NativeLimitMB:    settings.NativeLimitMB,
		PutURLTTLSeconds: settings.PutURLTTLSeconds,
		GetURLTTLSeconds: settings.GetURLTTLSeconds,
		RetentionDays:    settings.RetentionDays,
		UpdatedAt:        updatedAt,
	}
}
