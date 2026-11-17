package repository

import (
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

// InviteCodeRepository 处理 invite_codes 表的数据库操作。
type InviteCodeRepository struct {
	db *gorm.DB
}

func NewInviteCodeRepository() *InviteCodeRepository {
	return &InviteCodeRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func (r *InviteCodeRepository) FindByCode(code string) (*model.InviteCode, error) {
	var ic model.InviteCode
	if err := r.db.Where("code = ?", code).First(&ic).Error; err != nil {
		return nil, err
	}
	return &ic, nil
}

func (r *InviteCodeRepository) FindByID(id uint) (*model.InviteCode, error) {
	var ic model.InviteCode
	if err := r.db.First(&ic, id).Error; err != nil {
		return nil, err
	}
	return &ic, nil
}

func (r *InviteCodeRepository) Create(ic *model.InviteCode) error {
	return r.db.Create(ic).Error
}

func (r *InviteCodeRepository) Delete(id uint) error {
	return r.db.Delete(&model.InviteCode{}, id).Error
}

// FindActiveByOwner 查询某 owner 是否存在有效（active）邀请码。
func (r *InviteCodeRepository) FindActiveByOwner(ownerID uint) (*model.InviteCode, error) {
	var ic model.InviteCode
	err := r.db.Where("owner_id = ? AND status = 'active'", ownerID).
		Order("id DESC").
		First(&ic).Error
	if err != nil {
		return nil, err
	}
	return &ic, nil
}

// ListByOwner 不分页（供内部使用）。
func (r *InviteCodeRepository) ListByOwner(ownerID uint) ([]model.InviteCode, error) {
	var codes []model.InviteCode
	err := r.db.Where("owner_id = ?", ownerID).Find(&codes).Error
	return codes, err
}

// FindLatestByOwners 批量获取每个 owner 最新的邀请码（用于助理列表展示）。
func (r *InviteCodeRepository) FindLatestByOwners(ownerIDs []uint) ([]model.InviteCode, error) {
	if len(ownerIDs) == 0 {
		return nil, nil
	}
	subQuery := r.db.Model(&model.InviteCode{}).
		Select("MAX(id)").
		Where("owner_id IN ?", ownerIDs).
		Group("owner_id")
	var codes []model.InviteCode
	err := r.db.Where("id IN (?)", subQuery).Find(&codes).Error
	return codes, err
}

// UpdateFields 更新邀请码指定字段。
func (r *InviteCodeRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.InviteCode{}).Where("id = ?", id).Updates(fields).Error
}

// UpdateCode 替换邀请码字符串。
func (r *InviteCodeRepository) UpdateCode(id uint, code string) error {
	return r.db.Model(&model.InviteCode{}).Where("id = ?", id).Update("code", code).Error
}
func (r *InviteCodeRepository) ListPaged(page, pageSize int, scopeOwnerIDs []uint, ownerID uint) ([]model.InviteCode, int64, error) {
	q := r.db.Model(&model.InviteCode{})
	if ownerID > 0 {
		q = q.Where("owner_id = ?", ownerID)
	} else if len(scopeOwnerIDs) > 0 {
		q = q.Where("owner_id IN ?", scopeOwnerIDs)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var codes []model.InviteCode
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&codes).Error
	return codes, total, err
}

// InviteCodeWithOwner 携带归属人用户名的邀请码记录。
type InviteCodeWithOwner struct {
	model.InviteCode
	OwnerName string
}

// ListPagedWithOwner 分页列出邀请码并 JOIN admin_users 获取归属人用户名。
func (r *InviteCodeRepository) ListPagedWithOwner(page, pageSize int, scopeOwnerIDs []uint, ownerID uint) ([]InviteCodeWithOwner, int64, error) {
	q := r.db.Model(&model.InviteCode{})
	if ownerID > 0 {
		q = q.Where("invite_codes.owner_id = ?", ownerID)
	} else if len(scopeOwnerIDs) > 0 {
		q = q.Where("invite_codes.owner_id IN ?", scopeOwnerIDs)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	jq := r.db.Model(&model.InviteCode{}).
		Select("invite_codes.*, COALESCE(admin_users.username, '') AS owner_name").
		Joins("LEFT JOIN admin_users ON admin_users.id = invite_codes.owner_id").
		Order("invite_codes.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize)
	if ownerID > 0 {
		jq = jq.Where("invite_codes.owner_id = ?", ownerID)
	} else if len(scopeOwnerIDs) > 0 {
		jq = jq.Where("invite_codes.owner_id IN ?", scopeOwnerIDs)
	}
	var rows []InviteCodeWithOwner
	err := jq.Scan(&rows).Error
	return rows, total, err
}

// IncrementUsedCount 邀请码使用次数 +1，同时在达到上限时自动置 used。
func (r *InviteCodeRepository) IncrementUsedCount(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.InviteCode{}).Where("id = ?", id).
			UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error; err != nil {
			return err
		}
		// 若有次数上限且已用完则标记 used
		tx.Model(&model.InviteCode{}).
			Where("id = ? AND max_uses > 0 AND used_count >= max_uses", id).
			Update("status", "used")
		return nil
	})
}

// Count 统计邀请码总数。
func (r *InviteCodeRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.InviteCode{}).Count(&count).Error
	return count, err
}

// CountByOwner 统计某 owner 的邀请码数量。
func (r *InviteCodeRepository) CountByOwner(ownerID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.InviteCode{}).Where("owner_id = ?", ownerID).Count(&count).Error
	return count, err
}

// CountByOwners 统计指定 owner 集合的邀请码数量。
func (r *InviteCodeRepository) CountByOwners(ownerIDs []uint) (int64, error) {
	if len(ownerIDs) == 0 {
		return 0, nil
	}
	var count int64
	err := r.db.Model(&model.InviteCode{}).Where("owner_id IN ?", ownerIDs).Count(&count).Error
	return count, err
}
