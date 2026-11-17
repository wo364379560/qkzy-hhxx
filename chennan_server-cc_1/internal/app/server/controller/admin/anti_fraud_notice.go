package controller_admin

import (
	"context"

	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
)

// AntiFraudNoticeSettingsResult 防诈公告配置响应。
type AntiFraudNoticeSettingsResult struct {
	Enabled      bool   `json:"enabled"`
	Content      string `json:"content"`
	CountdownSec int    `json:"countdown_sec"`
}

// GetAntiFraudNoticeSettings 获取防诈公告配置。
func GetAntiFraudNoticeSettings() (*AntiFraudNoticeSettingsResult, error) {
	settings, err := config.GetAntiFraudNoticeSettings(context.Background(), datasource.Datasource.Mysql.GetClient())
	if err != nil {
		return nil, err
	}
	return &AntiFraudNoticeSettingsResult{
		Enabled:      settings.Enabled,
		Content:      settings.Content,
		CountdownSec: settings.CountdownSec,
	}, nil
}

// SetAntiFraudNoticeSettings 设置防诈公告配置。
func SetAntiFraudNoticeSettings(adminID uint, currentPassword string, settings AntiFraudNoticeSettingsResult) error {
	if err := verifySuperAdminPassword(adminID, currentPassword); err != nil {
		return err
	}
	if settings.CountdownSec <= 0 {
		settings.CountdownSec = 7
	}

	return config.SetAntiFraudNoticeSettingsTx(
		context.Background(),
		datasource.Datasource.Mysql.GetClient(),
		config.AntiFraudNoticeSettings{
			Enabled:      settings.Enabled,
			Content:      settings.Content,
			CountdownSec: settings.CountdownSec,
		},
		adminID,
	)
}
