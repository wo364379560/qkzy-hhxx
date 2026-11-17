package controller_admin

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/quietpolicycache"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
)

// Get2FAEnabled 获取全局 2FA 开关状态。
func Get2FAEnabled() (bool, error) {
	return config.Get2FAEnabled(context.Background(), datasource.Datasource.Mysql.GetClient())
}

// GetChatQuietHoursPolicy 获取聊天静默时段策略。
func GetChatQuietHoursPolicy() (config.ChatQuietHoursPolicy, error) {
	return config.GetChatQuietHoursPolicy(context.Background(), datasource.Datasource.Mysql.GetClient())
}

// Set2FAEnabled 设置全局 2FA 开关（仅 admin 角色）。
func Set2FAEnabled(adminID uint, currentPassword string, enabled, confirmDisableAll bool) error {
	if err := verifySuperAdminPassword(adminID, currentPassword); err != nil {
		return err
	}
	if !enabled && !confirmDisableAll {
		return errors.New("请确认关闭并清空所有后台账号的双因素认证")
	}

	db := datasource.Datasource.Mysql.GetClient()
	return db.Transaction(func(tx *gorm.DB) error {
		if err := config.Set2FAEnabledTx(context.Background(), tx, enabled, adminID); err != nil {
			return err
		}
		if enabled {
			return nil
		}

		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).
			Model(&model.AdminUser{}).
			Updates(map[string]any{
				"totp_secret":   "",
				"totp_enabled":  false,
				"totp_bound_at": nil,
			}).Error; err != nil {
			return err
		}

		return tx.Session(&gorm.Session{AllowGlobalUpdate: true}).
			Delete(&model.AdminTOTPBackupCode{}).Error
	})
}

// SetChatQuietHoursPolicy 设置聊天静默时段策略。
func SetChatQuietHoursPolicy(adminID uint, currentPassword string, policy config.ChatQuietHoursPolicy) error {
	if err := verifySuperAdminPassword(adminID, currentPassword); err != nil {
		return err
	}

	if err := config.SetChatQuietHoursPolicyTx(
		context.Background(),
		datasource.Datasource.Mysql.GetClient(),
		policy,
		adminID,
	); err != nil {
		return err
	}
	quietpolicycache.InvalidateAll(context.Background())
	return nil
}

// ResetAssistantTOTP 重置管理范围内助理的 TOTP 绑定。
func ResetAssistantTOTP(actorID, assistantID uint) error {
	if _, err := findManagedAssistant(actorID, assistantID); err != nil {
		return err
	}

	db := datasource.Datasource.Mysql.GetClient()
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.AdminUser{}).
			Where("id = ?", assistantID).
			Updates(map[string]any{
				"totp_secret":   "",
				"totp_enabled":  false,
				"totp_bound_at": nil,
				"auth_version":  gorm.Expr("GREATEST(COALESCE(auth_version, 1), 1) + 1"),
			}).Error; err != nil {
			return err
		}

		return tx.Where("admin_id = ?", assistantID).
			Delete(&model.AdminTOTPBackupCode{}).Error
	})
}
