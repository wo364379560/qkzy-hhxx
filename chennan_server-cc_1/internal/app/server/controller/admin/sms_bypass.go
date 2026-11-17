package controller_admin

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/authwork"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
)

var smsBypassCodePattern = regexp.MustCompile(`^\d{6}$`)

// SMSBypassSettingsResult 是万能验证码配置响应。
type SMSBypassSettingsResult struct {
	Enabled    bool   `json:"enabled"`
	HasCode    bool   `json:"has_code"`
	MaskedCode string `json:"masked_code"`
}

// GetSMSBypassSettings 获取万能验证码配置摘要。
func GetSMSBypassSettings() (*SMSBypassSettingsResult, error) {
	settings, err := config.GetSMSBypassSettings(context.Background(), datasource.Datasource.Mysql.GetClient())
	if err != nil {
		return nil, err
	}

	hasCode := strings.TrimSpace(settings.Code) != ""
	maskedCode := ""
	if hasCode {
		maskedCode = "******"
	}

	return &SMSBypassSettingsResult{
		Enabled:    settings.Enabled,
		HasCode:    hasCode,
		MaskedCode: maskedCode,
	}, nil
}

// RevealSMSBypassCode 校验密码并返回万能验证码明文。
func RevealSMSBypassCode(adminID uint, currentPassword string) (string, error) {
	if err := verifySuperAdminPassword(adminID, currentPassword); err != nil {
		return "", err
	}

	settings, err := config.GetSMSBypassSettings(context.Background(), datasource.Datasource.Mysql.GetClient())
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(settings.Code) == "" {
		return "", errors.New("当前未配置万能验证码")
	}
	return settings.Code, nil
}

// SetSMSBypassSettings 设置万能验证码配置。
func SetSMSBypassSettings(adminID uint, currentPassword string, enabled bool, code *string) error {
	if err := verifySuperAdminPassword(adminID, currentPassword); err != nil {
		return err
	}

	db := datasource.Datasource.Mysql.GetClient()
	return db.Transaction(func(tx *gorm.DB) error {
		settings, err := config.GetSMSBypassSettings(context.Background(), tx)
		if err != nil {
			return err
		}

		nextCode := settings.Code
		if code != nil {
			normalized, err := normalizeSMSBypassCode(*code)
			if err != nil {
				return err
			}
			nextCode = normalized
		}

		if enabled && strings.TrimSpace(nextCode) == "" {
			return errors.New("启用前请先设置 6 位万能验证码")
		}

		return config.SetSMSBypassSettingsTx(context.Background(), tx, enabled, nextCode, adminID)
	})
}

func verifySuperAdminPassword(adminID uint, currentPassword string) error {
	repo := repository.NewAdminUserRepository()
	admin, err := repo.FindByID(adminID)
	if err != nil {
		return err
	}
	if !IsSuperAdminRole(admin.Role) {
		return errors.New("仅超级管理员可操作")
	}
	if err := authwork.ComparePrivateHashAndPassword([]byte(admin.PasswordHash), []byte(currentPassword)); err != nil {
		if errors.Is(err, authwork.ErrBusy) {
			return err
		}
		return errors.New("当前密码错误")
	}
	return nil
}

func normalizeSMSBypassCode(code string) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", nil
	}
	if !smsBypassCodePattern.MatchString(code) {
		return "", errors.New("万能验证码必须为 6 位数字")
	}
	return code, nil
}
