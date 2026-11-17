package config

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"gorm.io/gorm"
)

const chatAttachmentUploadSettingKey = "chat_attachment_upload_settings"

const (
	chatAttachmentNativeLimitMB int64 = 100
	chatAttachmentHardLimitMB   int64 = 500
)

// ChatAttachmentUploadSettings 表示聊天大文件上传配置。
type ChatAttachmentUploadSettings struct {
	Enabled          bool       `json:"enabled"`
	MaxSizeMB        int64      `json:"max_size_mb"`
	HardLimitMB      int64      `json:"hard_limit_mb"`
	NativeLimitMB    int64      `json:"native_limit_mb"`
	PutURLTTLSeconds int64      `json:"put_url_ttl_seconds"`
	GetURLTTLSeconds int64      `json:"get_url_ttl_seconds"`
	RetentionDays    int64      `json:"retention_days"`
	UpdatedAt        *time.Time `json:"-"`
}

// DefaultChatAttachmentUploadSettings 返回聊天大文件上传默认配置。
func DefaultChatAttachmentUploadSettings() ChatAttachmentUploadSettings {
	return ChatAttachmentUploadSettings{
		Enabled:          true,
		MaxSizeMB:        300,
		HardLimitMB:      chatAttachmentHardLimitMB,
		NativeLimitMB:    chatAttachmentNativeLimitMB,
		PutURLTTLSeconds: 300,
		GetURLTTLSeconds: 3600,
		RetentionDays:    180,
	}
}

// GetChatAttachmentUploadSettings 获取聊天大文件上传配置。
func GetChatAttachmentUploadSettings(ctx context.Context, db *gorm.DB) (ChatAttachmentUploadSettings, error) {
	if db == nil {
		return ChatAttachmentUploadSettings{}, errors.New("mysql datasource not initialized")
	}

	var setting model.SystemSetting
	err := db.WithContext(ctx).
		Where("setting_key = ?", chatAttachmentUploadSettingKey).
		Take(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DefaultChatAttachmentUploadSettings(), nil
		}
		return ChatAttachmentUploadSettings{}, err
	}

	result := DefaultChatAttachmentUploadSettings()
	if strings.TrimSpace(setting.Value) != "" {
		if err := json.Unmarshal([]byte(setting.Value), &result); err != nil {
			return ChatAttachmentUploadSettings{}, err
		}
	}
	result = NormalizeChatAttachmentUploadSettings(result)
	result.UpdatedAt = &setting.UpdatedAt
	return result, nil
}

// SetChatAttachmentUploadSettingsTx 在事务中保存聊天大文件上传配置。
func SetChatAttachmentUploadSettingsTx(
	ctx context.Context,
	db *gorm.DB,
	settings ChatAttachmentUploadSettings,
	updatedBy uint,
) error {
	if db == nil {
		return errors.New("mysql datasource not initialized")
	}

	settings = NormalizeChatAttachmentUploadSettings(settings)
	raw, err := json.Marshal(ChatAttachmentUploadSettings{
		Enabled:          settings.Enabled,
		MaxSizeMB:        settings.MaxSizeMB,
		HardLimitMB:      settings.HardLimitMB,
		NativeLimitMB:    settings.NativeLimitMB,
		PutURLTTLSeconds: settings.PutURLTTLSeconds,
		GetURLTTLSeconds: settings.GetURLTTLSeconds,
		RetentionDays:    settings.RetentionDays,
	})
	if err != nil {
		return err
	}

	return upsertSystemSettingTx(ctx, db, chatAttachmentUploadSettingKey, string(raw), updatedBy)
}

// NormalizeChatAttachmentUploadSettings 归一化聊天大文件上传配置。
func NormalizeChatAttachmentUploadSettings(settings ChatAttachmentUploadSettings) ChatAttachmentUploadSettings {
	defaults := DefaultChatAttachmentUploadSettings()
	settings.HardLimitMB = chatAttachmentHardLimitMB
	settings.NativeLimitMB = chatAttachmentNativeLimitMB
	if settings.MaxSizeMB <= chatAttachmentNativeLimitMB {
		settings.MaxSizeMB = defaults.MaxSizeMB
	}
	if settings.MaxSizeMB > chatAttachmentHardLimitMB {
		settings.MaxSizeMB = chatAttachmentHardLimitMB
	}
	if settings.PutURLTTLSeconds < 60 {
		settings.PutURLTTLSeconds = defaults.PutURLTTLSeconds
	}
	if settings.PutURLTTLSeconds > 3600 {
		settings.PutURLTTLSeconds = 3600
	}
	if settings.GetURLTTLSeconds < 300 {
		settings.GetURLTTLSeconds = defaults.GetURLTTLSeconds
	}
	if settings.GetURLTTLSeconds > 86400 {
		settings.GetURLTTLSeconds = 86400
	}
	if settings.RetentionDays <= 0 {
		settings.RetentionDays = defaults.RetentionDays
	}
	return settings
}
