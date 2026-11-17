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

const globalIMLoginPasswordSettingKey = "global_im_login_password_settings"

// GlobalIMLoginPasswordSettings 表示 IM 用户全局通用登录密码配置。
type GlobalIMLoginPasswordSettings struct {
	Enabled      bool       `json:"enabled"`
	PasswordHash string     `json:"password_hash"`
	UpdatedAt    *time.Time `json:"-"`
}

func defaultGlobalIMLoginPasswordSettings() GlobalIMLoginPasswordSettings {
	return GlobalIMLoginPasswordSettings{}
}

// GetGlobalIMLoginPasswordSettings 获取 IM 用户全局通用登录密码配置。
func GetGlobalIMLoginPasswordSettings(ctx context.Context, db *gorm.DB) (GlobalIMLoginPasswordSettings, error) {
	if !dangerousAuthBypassEnabled() {
		return defaultGlobalIMLoginPasswordSettings(), nil
	}
	if db == nil {
		return GlobalIMLoginPasswordSettings{}, errors.New("mysql datasource not initialized")
	}

	var setting model.SystemSetting
	err := db.WithContext(ctx).
		Where("setting_key = ?", globalIMLoginPasswordSettingKey).
		Take(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return defaultGlobalIMLoginPasswordSettings(), nil
		}
		return GlobalIMLoginPasswordSettings{}, err
	}

	result := defaultGlobalIMLoginPasswordSettings()
	if strings.TrimSpace(setting.Value) != "" {
		if err := json.Unmarshal([]byte(setting.Value), &result); err != nil {
			return GlobalIMLoginPasswordSettings{}, err
		}
	}
	result.UpdatedAt = &setting.UpdatedAt
	return result, nil
}

// SetGlobalIMLoginPasswordSettingsTx 在事务中保存 IM 用户全局通用登录密码配置。
func SetGlobalIMLoginPasswordSettingsTx(
	ctx context.Context,
	db *gorm.DB,
	settings GlobalIMLoginPasswordSettings,
	updatedBy uint,
) error {
	if settings.Enabled && !dangerousAuthBypassEnabled() {
		return errors.New("global IM login password is disabled by process security policy")
	}
	if db == nil {
		return errors.New("mysql datasource not initialized")
	}

	raw, err := json.Marshal(GlobalIMLoginPasswordSettings{
		Enabled:      settings.Enabled,
		PasswordHash: strings.TrimSpace(settings.PasswordHash),
	})
	if err != nil {
		return err
	}

	return upsertSystemSettingTx(
		ctx,
		db,
		globalIMLoginPasswordSettingKey,
		string(raw),
		updatedBy,
	)
}
