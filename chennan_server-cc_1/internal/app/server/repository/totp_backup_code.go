package repository

import (
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

// TOTPBackupCodeRepository TOTP 备份码仓库。
type TOTPBackupCodeRepository struct {
	db *gorm.DB
}

// NewTOTPBackupCodeRepository 创建备份码仓库实例。
func NewTOTPBackupCodeRepository() *TOTPBackupCodeRepository {
	return &TOTPBackupCodeRepository{db: datasource.Datasource.Mysql.GetClient()}
}

// Create 创建备份码。
func (r *TOTPBackupCodeRepository) Create(code *model.AdminTOTPBackupCode) error {
	return r.db.Create(code).Error
}

// FindUnusedByAdminID 查找管理员未使用的备份码。
func (r *TOTPBackupCodeRepository) FindUnusedByAdminID(adminID uint) ([]model.AdminTOTPBackupCode, error) {
	var codes []model.AdminTOTPBackupCode
	err := r.db.Where("admin_id = ? AND used = ?", adminID, false).Find(&codes).Error
	return codes, err
}

// MarkAsUsed 标记备份码为已使用。
func (r *TOTPBackupCodeRepository) MarkAsUsed(id uint) error {
	now := time.Now()
	result := r.db.Model(&model.AdminTOTPBackupCode{}).
		Where("id = ? AND used = ?", id, false).
		Updates(map[string]interface{}{
			"used":    true,
			"used_at": &now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// DeleteByAdminID 删除管理员的所有备份码。
func (r *TOTPBackupCodeRepository) DeleteByAdminID(adminID uint) error {
	return r.db.Where("admin_id = ?", adminID).Delete(&model.AdminTOTPBackupCode{}).Error
}
