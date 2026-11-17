package controller_admin

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/pkg/authwork"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
)

// GlobalLoginPasswordSettingsResult 是 IM 用户通用登录密码配置响应。
type GlobalLoginPasswordSettingsResult struct {
	Enabled     bool   `json:"enabled"`
	HasPassword bool   `json:"has_password"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// GetGlobalLoginPasswordSettings 获取 IM 用户通用登录密码配置摘要。
func GetGlobalLoginPasswordSettings() (*GlobalLoginPasswordSettingsResult, error) {
	settings, err := config.GetGlobalIMLoginPasswordSettings(context.Background(), datasource.Datasource.Mysql.GetClient())
	if err != nil {
		return nil, err
	}
	return globalLoginPasswordSettingsResult(settings), nil
}

// SetGlobalLoginPasswordSettings 设置 IM 用户通用登录密码配置。
func SetGlobalLoginPasswordSettings(adminID uint, currentPassword string, enabled bool, password *string) error {
	if err := verifySuperAdminPassword(adminID, currentPassword); err != nil {
		return err
	}

	var requestedHash string
	if password != nil {
		normalized, err := normalizeGlobalLoginPassword(*password)
		if err != nil {
			return err
		}
		if normalized != "" {
			hash, err := authwork.GeneratePrivateFromPassword([]byte(normalized), authwork.DefaultCost)
			if err != nil {
				return err
			}
			requestedHash = string(hash)
		}
	}

	db := datasource.Datasource.Mysql.GetClient()
	return db.Transaction(func(tx *gorm.DB) error {
		settings, err := config.GetGlobalIMLoginPasswordSettings(context.Background(), tx)
		if err != nil {
			return err
		}

		nextHash := strings.TrimSpace(settings.PasswordHash)
		if requestedHash != "" {
			nextHash = requestedHash
		}

		if enabled && strings.TrimSpace(nextHash) == "" {
			return errors.New("启用前请先设置通用登录密码")
		}

		return config.SetGlobalIMLoginPasswordSettingsTx(
			context.Background(),
			tx,
			config.GlobalIMLoginPasswordSettings{
				Enabled:      enabled,
				PasswordHash: nextHash,
			},
			adminID,
		)
	})
}

func globalLoginPasswordSettingsResult(settings config.GlobalIMLoginPasswordSettings) *GlobalLoginPasswordSettingsResult {
	updatedAt := ""
	if settings.UpdatedAt != nil && !settings.UpdatedAt.IsZero() {
		updatedAt = settings.UpdatedAt.Format("2006-01-02 15:04:05")
	}
	return &GlobalLoginPasswordSettingsResult{
		Enabled:     settings.Enabled,
		HasPassword: strings.TrimSpace(settings.PasswordHash) != "",
		UpdatedAt:   updatedAt,
	}
}

func normalizeGlobalLoginPassword(password string) (string, error) {
	password = strings.TrimSpace(password)
	if password == "" {
		return "", nil
	}
	if len(password) < 8 || len(password) > 32 {
		return "", errors.New("通用登录密码长度必须为 8-32 位")
	}
	return password, nil
}
