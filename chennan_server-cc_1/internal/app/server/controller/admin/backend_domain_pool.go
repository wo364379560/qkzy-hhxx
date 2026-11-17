package controller_admin

import (
	"context"

	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
)

// GetBackendDomainPoolSettings 获取业务域名池配置。
func GetBackendDomainPoolSettings() (config.BackendDomainPoolSettings, error) {
	return config.GetBackendDomainPoolSettings(
		context.Background(),
		datasource.Datasource.Mysql.GetClient(),
	)
}

// SetBackendDomainPoolSettings 保存业务域名池配置。
func SetBackendDomainPoolSettings(
	adminID uint,
	currentPassword string,
	settings config.BackendDomainPoolSettings,
) (config.BackendDomainPoolSettings, error) {
	if err := verifySuperAdminPassword(adminID, currentPassword); err != nil {
		return config.BackendDomainPoolSettings{}, err
	}

	return config.SetBackendDomainPoolSettingsTx(
		context.Background(),
		datasource.Datasource.Mysql.GetClient(),
		settings,
		adminID,
	)
}
