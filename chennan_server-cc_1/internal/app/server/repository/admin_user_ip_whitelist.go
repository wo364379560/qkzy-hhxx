package repository

import (
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

// AdminUserIPWhitelistRepository 处理 admin_user_ip_whitelists 表的数据库操作。
type AdminUserIPWhitelistRepository struct {
	db *gorm.DB
}

// NewAdminUserIPWhitelistRepository 创建助理 IP 白名单仓储。
func NewAdminUserIPWhitelistRepository() *AdminUserIPWhitelistRepository {
	return &AdminUserIPWhitelistRepository{db: datasource.Datasource.Mysql.GetClient()}
}

// ListByAdminUserID 查询某个助理账号下的所有白名单规则。
func (r *AdminUserIPWhitelistRepository) ListByAdminUserID(adminUserID uint) ([]model.AdminUserIPWhitelist, error) {
	var rules []model.AdminUserIPWhitelist
	err := r.db.Where("admin_user_id = ?", adminUserID).Order("id ASC").Find(&rules).Error
	return rules, err
}

// ListByAdminUserIDs 批量查询多个助理账号下的所有白名单规则。
func (r *AdminUserIPWhitelistRepository) ListByAdminUserIDs(adminUserIDs []uint) ([]model.AdminUserIPWhitelist, error) {
	if len(adminUserIDs) == 0 {
		return []model.AdminUserIPWhitelist{}, nil
	}

	var rules []model.AdminUserIPWhitelist
	err := r.db.Where("admin_user_id IN ?", adminUserIDs).Order("admin_user_id ASC, id ASC").Find(&rules).Error
	return rules, err
}

// ReplaceRules 全量替换某个助理账号的白名单规则。
func (r *AdminUserIPWhitelistRepository) ReplaceRules(adminUserID uint, rules []model.AdminUserIPWhitelist) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("admin_user_id = ?", adminUserID).Delete(&model.AdminUserIPWhitelist{}).Error; err != nil {
			return err
		}

		if len(rules) == 0 {
			return nil
		}

		return tx.Create(&rules).Error
	})
}
