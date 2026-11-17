package repository

import (
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

// AdminUserRepository 处理 admin_users 表的数据库操作。
type AdminUserRepository struct {
	db *gorm.DB
}

type parentAdminCount struct {
	ParentAdminID uint  `gorm:"column:parent_admin_id"`
	Total         int64 `gorm:"column:total"`
}

func NewAdminUserRepository() *AdminUserRepository {
	return &AdminUserRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func NewAdminUserRepositoryWithDB(db *gorm.DB) *AdminUserRepository {
	return &AdminUserRepository{db: db}
}

func (r *AdminUserRepository) FindByUsername(username string) (*model.AdminUser, error) {
	var admin model.AdminUser
	if err := r.db.Where("username = ?", username).First(&admin).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

func (r *AdminUserRepository) FindByID(id uint) (*model.AdminUser, error) {
	var admin model.AdminUser
	if err := r.db.First(&admin, id).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

// ListAllIDs 返回全部后台账号 ID。
func (r *AdminUserRepository) ListAllIDs() ([]uint, error) {
	var ids []uint
	err := r.db.Model(&model.AdminUser{}).Pluck("id", &ids).Error
	return ids, err
}

func (r *AdminUserRepository) Create(admin *model.AdminUser) error {
	return r.db.Create(admin).Error
}

func (r *AdminUserRepository) Delete(id uint) error {
	return r.db.Delete(&model.AdminUser{}, id).Error
}

// ListAssistants 不分页（供内部使用）。
func (r *AdminUserRepository) ListAssistants() ([]model.AdminUser, error) {
	var admins []model.AdminUser
	err := r.db.Where("role = ?", "assistant").Find(&admins).Error
	return admins, err
}

// ListByParentAdminID 返回指定上级后台账号下的直接子账号。
func (r *AdminUserRepository) ListByParentAdminID(parentAdminID uint) ([]model.AdminUser, error) {
	var admins []model.AdminUser
	err := r.db.Where("parent_admin_id = ?", parentAdminID).Find(&admins).Error
	return admins, err
}

// FindFirstByRole 返回指定角色的第一条后台账号记录。
func (r *AdminUserRepository) FindFirstByRole(role string) (*model.AdminUser, error) {
	var admin model.AdminUser
	if err := r.db.Where("role = ?", role).Order("id ASC").First(&admin).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

// ListByRole 返回指定角色的全部后台账号，按 id 升序。
func (r *AdminUserRepository) ListByRole(role string) ([]model.AdminUser, error) {
	var admins []model.AdminUser
	err := r.db.Where("role = ? AND status = ?", role, "active").Order("id ASC").Find(&admins).Error
	return admins, err
}

// ListAssistantsPaged 分页查询助理列表（仅管理员调用）。
func (r *AdminUserRepository) ListAssistantsPaged(page, pageSize int) ([]model.AdminUser, int64, error) {
	q := r.db.Model(&model.AdminUser{}).Where("role = ?", "assistant")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var admins []model.AdminUser
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&admins).Error
	return admins, total, err
}

// ListAssistantsByParentPaged 分页查询指定上级后台账号名下助理列表。
func (r *AdminUserRepository) ListAssistantsByParentPaged(parentAdminID uint, page, pageSize int) ([]model.AdminUser, int64, error) {
	q := r.db.Model(&model.AdminUser{}).
		Where("role = ? AND parent_admin_id = ?", "assistant", parentAdminID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var admins []model.AdminUser
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&admins).Error
	return admins, total, err
}

// ListAssistantsFilteredPaged 分页查询助理列表，支持用户名、状态与 IP 白名单启用状态筛选。
func (r *AdminUserRepository) ListAssistantsFilteredPaged(page, pageSize int, keyword, status string, ipWhitelistEnabled *bool) ([]model.AdminUser, int64, error) {
	q := r.db.Model(&model.AdminUser{}).Where("role = ?", "assistant")
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("username LIKE ?", like)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if ipWhitelistEnabled != nil {
		q = q.Where("ip_whitelist_enabled = ?", *ipWhitelistEnabled)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var admins []model.AdminUser
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&admins).Error
	return admins, total, err
}

// ListAssistantsByParentFilteredPaged 分页查询指定上级后台账号名下助理列表，支持用户名、状态与 IP 白名单启用状态筛选。
func (r *AdminUserRepository) ListAssistantsByParentFilteredPaged(parentAdminID uint, page, pageSize int, keyword, status string, ipWhitelistEnabled *bool) ([]model.AdminUser, int64, error) {
	q := r.db.Model(&model.AdminUser{}).
		Where("role = ? AND parent_admin_id = ?", "assistant", parentAdminID)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("username LIKE ?", like)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if ipWhitelistEnabled != nil {
		q = q.Where("ip_whitelist_enabled = ?", *ipWhitelistEnabled)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var admins []model.AdminUser
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&admins).Error
	return admins, total, err
}

// ListAdminsPaged 分页查询超级管理员/管理员列表。
func (r *AdminUserRepository) ListAdminsPaged(page, pageSize int) ([]model.AdminUser, int64, error) {
	q := r.db.Model(&model.AdminUser{}).Where("role IN ?", []string{"super_admin", "admin"})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var admins []model.AdminUser
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&admins).Error
	return admins, total, err
}

// ListManagersPaged 分页查询管理员列表。
func (r *AdminUserRepository) ListManagersPaged(page, pageSize int) ([]model.AdminUser, int64, error) {
	q := r.db.Model(&model.AdminUser{}).Where("role = ?", "admin")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var admins []model.AdminUser
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&admins).Error
	return admins, total, err
}

// ListWhitelistAccountsPaged 分页查询白名单策略概览涉及的后台账号。
func (r *AdminUserRepository) ListWhitelistAccountsPaged(page, pageSize int, roles []string, keyword, status string, ipWhitelistEnabled *bool, parentAdminID *uint) ([]model.AdminUser, int64, error) {
	q := r.db.Model(&model.AdminUser{}).Where("role IN ?", roles)
	if parentAdminID != nil {
		q = q.Where("parent_admin_id = ?", *parentAdminID)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("username LIKE ?", like)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if ipWhitelistEnabled != nil {
		q = q.Where("ip_whitelist_enabled = ?", *ipWhitelistEnabled)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var admins []model.AdminUser
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&admins).Error
	return admins, total, err
}

// CountManagers 统计管理员总数。
func (r *AdminUserRepository) CountManagers() (int64, error) {
	var count int64
	err := r.db.Model(&model.AdminUser{}).Where("role = ?", "admin").Count(&count).Error
	return count, err
}

// UpdateStatus 更新账号状态。
func (r *AdminUserRepository) UpdateStatus(id uint, status string) error {
	return r.db.Model(&model.AdminUser{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateDisplayName 更新后台账号昵称。
func (r *AdminUserRepository) UpdateDisplayName(id uint, displayName string) error {
	return r.db.Model(&model.AdminUser{}).Where("id = ?", id).Update("display_name", strings.TrimSpace(displayName)).Error
}

// UpdatePassword 更新密码哈希并提升认证版本号。
func (r *AdminUserRepository) UpdatePassword(id uint, hash string) error {
	return r.db.Model(&model.AdminUser{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"password_hash": hash,
			"auth_version":  gorm.Expr("GREATEST(auth_version, 1) + 1"),
		}).Error
}

// UpdateIPWhitelistEnabled 更新助理账号是否启用 IP 白名单。
func (r *AdminUserRepository) UpdateIPWhitelistEnabled(id uint, enabled bool) error {
	return r.db.Model(&model.AdminUser{}).Where("id = ?", id).Update("ip_whitelist_enabled", enabled).Error
}

// CountAssistants 统计助理总数。
func (r *AdminUserRepository) CountAssistants() (int64, error) {
	var count int64
	err := r.db.Model(&model.AdminUser{}).Where("role = ?", "assistant").Count(&count).Error
	return count, err
}

// CountAssistantsByParents 批量统计每个上级后台账号名下的助理数量。
func (r *AdminUserRepository) CountAssistantsByParents(parentAdminIDs []uint) (map[uint]int64, error) {
	if len(parentAdminIDs) == 0 {
		return map[uint]int64{}, nil
	}
	var rows []parentAdminCount
	err := r.db.Model(&model.AdminUser{}).
		Select("parent_admin_id, COUNT(*) AS total").
		Where("role = ? AND parent_admin_id IN ?", "assistant", parentAdminIDs).
		Group("parent_admin_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uint]int64, len(rows))
	for _, row := range rows {
		result[row.ParentAdminID] = row.Total
	}
	return result, nil
}

// FindByIDs 批量查询管理员/助理（供用户列表归属信息使用）。
func (r *AdminUserRepository) FindByIDs(ids []uint) ([]model.AdminUser, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var admins []model.AdminUser
	err := r.db.Where("id IN ?", ids).Find(&admins).Error
	return admins, err
}
