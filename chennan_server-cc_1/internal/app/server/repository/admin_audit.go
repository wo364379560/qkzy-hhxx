package repository

import (
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

// AdminAccessAuditRepository 处理后台访问审计表。
type AdminAccessAuditRepository struct {
	db *gorm.DB
}

// NewAdminAccessAuditRepository 创建后台访问审计仓储。
func NewAdminAccessAuditRepository() *AdminAccessAuditRepository {
	return &AdminAccessAuditRepository{db: datasource.Datasource.Mysql.GetClient()}
}

// NewAdminAccessAuditRepositoryWithDB 使用指定数据库连接创建后台访问审计仓储。
func NewAdminAccessAuditRepositoryWithDB(db *gorm.DB) *AdminAccessAuditRepository {
	return &AdminAccessAuditRepository{db: db}
}

// Create 写入一条后台访问审计。
func (r *AdminAccessAuditRepository) Create(audit *model.AdminAccessAudit) error {
	return r.db.Create(audit).Error
}

// CreateLegacy 写入旧版后台访问审计字段，兼容未迁移结构化字段的数据库。
func (r *AdminAccessAuditRepository) CreateLegacy(audit *model.AdminAccessAudit) error {
	return r.db.Exec(
		`INSERT INTO admin_access_audits
		(admin_user_id, admin_username, admin_role, access_type, method, route, target_id, success, status_code, ip_address, user_agent, failure_reason, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		audit.AdminUserID,
		audit.AdminUsername,
		audit.AdminRole,
		audit.AccessType,
		audit.Method,
		audit.Route,
		audit.TargetID,
		audit.Success,
		audit.StatusCode,
		audit.IPAddress,
		audit.UserAgent,
		audit.FailureReason,
		audit.CreatedAt,
	).Error
}

func applyAdminAccessAuditFilters(q *gorm.DB, keyword, role, accessType, method, targetType, actionKey, targetKeyword string, scopeAdminIDs []uint, scopeUsernames []string) *gorm.DB {
	switch {
	case len(scopeAdminIDs) > 0 && len(scopeUsernames) > 0:
		q = q.Where("(admin_user_id IN ? OR admin_username IN ?)", scopeAdminIDs, scopeUsernames)
	case len(scopeAdminIDs) > 0:
		q = q.Where("admin_user_id IN ?", scopeAdminIDs)
	case len(scopeUsernames) > 0:
		q = q.Where("admin_username IN ?", scopeUsernames)
	}
	if role != "" {
		q = q.Where("admin_role = ?", role)
	}
	if accessType != "" {
		q = q.Where("access_type = ?", accessType)
	}
	if method != "" {
		q = q.Where("method = ?", strings.ToUpper(method))
	}
	if targetType != "" {
		q = q.Where("target_type = ?", targetType)
	}
	if actionKey != "" {
		q = q.Where("action_key = ?", actionKey)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("admin_username LIKE ? OR route LIKE ? OR ip_address LIKE ? OR action_summary LIKE ?", like, like, like, like)
	}
	if targetKeyword = strings.TrimSpace(targetKeyword); targetKeyword != "" {
		like := "%" + targetKeyword + "%"
		q = q.Where("target_username LIKE ? OR target_display_name LIKE ?", like, like)
	}
	return q
}

// ListPaged 分页查询后台访问审计。
func (r *AdminAccessAuditRepository) ListPaged(page, pageSize int, keyword, role, accessType, method, targetType, actionKey, targetKeyword string, scopeAdminIDs []uint, scopeUsernames []string) ([]model.AdminAccessAudit, int64, error) {
	q := applyAdminAccessAuditFilters(
		r.db.Model(&model.AdminAccessAudit{}),
		keyword,
		role,
		accessType,
		method,
		targetType,
		actionKey,
		targetKeyword,
		scopeAdminIDs,
		scopeUsernames,
	)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var audits []model.AdminAccessAudit
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&audits).Error
	return audits, total, err
}

// CountBefore 统计指定时间点之前的后台访问审计数量。
func (r *AdminAccessAuditRepository) CountBefore(before time.Time, keyword, role, accessType, method, targetType, actionKey, targetKeyword string) (int64, error) {
	var total int64
	err := applyAdminAccessAuditFilters(
		r.db.Model(&model.AdminAccessAudit{}),
		keyword,
		role,
		accessType,
		method,
		targetType,
		actionKey,
		targetKeyword,
		nil,
		nil,
	).
		Where("created_at < ?", before).
		Count(&total).Error
	return total, err
}

// DeleteBefore 删除指定时间点之前的后台访问审计。
func (r *AdminAccessAuditRepository) DeleteBefore(before time.Time, keyword, role, accessType, method, targetType, actionKey, targetKeyword string) (int64, error) {
	result := applyAdminAccessAuditFilters(
		r.db.Model(&model.AdminAccessAudit{}),
		keyword,
		role,
		accessType,
		method,
		targetType,
		actionKey,
		targetKeyword,
		nil,
		nil,
	).Where("created_at < ?", before).Delete(&model.AdminAccessAudit{})
	return result.RowsAffected, result.Error
}

// AdminLoginAuditRepository 处理后台登录审计表。
type AdminLoginAuditRepository struct {
	db *gorm.DB
}

// NewAdminLoginAuditRepository 创建后台登录审计仓储。
func NewAdminLoginAuditRepository() *AdminLoginAuditRepository {
	return &AdminLoginAuditRepository{db: datasource.Datasource.Mysql.GetClient()}
}

// NewAdminLoginAuditRepositoryWithDB 使用指定数据库连接创建后台登录审计仓储。
func NewAdminLoginAuditRepositoryWithDB(db *gorm.DB) *AdminLoginAuditRepository {
	return &AdminLoginAuditRepository{db: db}
}

// Create 写入一条后台登录审计。
func (r *AdminLoginAuditRepository) Create(audit *model.AdminLoginAudit) error {
	return r.db.Create(audit).Error
}

func applyAdminLoginAuditFilters(q *gorm.DB, keyword, role, stage, result string, scopeAdminIDs []uint, scopeUsernames []string) *gorm.DB {
	switch {
	case len(scopeAdminIDs) > 0 && len(scopeUsernames) > 0:
		q = q.Where("(admin_user_id IN ? OR username IN ?)", scopeAdminIDs, scopeUsernames)
	case len(scopeAdminIDs) > 0:
		q = q.Where("admin_user_id IN ?", scopeAdminIDs)
	case len(scopeUsernames) > 0:
		q = q.Where("username IN ?", scopeUsernames)
	}
	if role != "" {
		q = q.Where("admin_role = ?", role)
	}
	if stage != "" {
		q = q.Where("stage = ?", stage)
	}
	if result != "" {
		q = q.Where("result = ?", result)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("username LIKE ? OR ip_address LIKE ?", like, like)
	}
	return q
}

// UpdateOutcome 更新后台登录审计的阶段、结果和失败原因。
func (r *AdminLoginAuditRepository) UpdateOutcome(id uint, stage, result, failureReason string) error {
	return r.db.Model(&model.AdminLoginAudit{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"stage":          stage,
			"result":         result,
			"failure_reason": failureReason,
		}).Error
}

// ListPaged 分页查询后台登录审计。
func (r *AdminLoginAuditRepository) ListPaged(page, pageSize int, keyword, role, stage, result string, scopeAdminIDs []uint, scopeUsernames []string) ([]model.AdminLoginAudit, int64, error) {
	q := applyAdminLoginAuditFilters(
		r.db.Model(&model.AdminLoginAudit{}),
		keyword,
		role,
		stage,
		result,
		scopeAdminIDs,
		scopeUsernames,
	)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var audits []model.AdminLoginAudit
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&audits).Error
	return audits, total, err
}

// CountBefore 统计指定时间点之前的后台登录审计数量。
func (r *AdminLoginAuditRepository) CountBefore(before time.Time, keyword, role, stage, result string) (int64, error) {
	var total int64
	err := applyAdminLoginAuditFilters(
		r.db.Model(&model.AdminLoginAudit{}),
		keyword,
		role,
		stage,
		result,
		nil,
		nil,
	).
		Where("created_at < ?", before).
		Count(&total).Error
	return total, err
}

// DeleteBefore 删除指定时间点之前的后台登录审计。
func (r *AdminLoginAuditRepository) DeleteBefore(before time.Time, keyword, role, stage, result string) (int64, error) {
	dbResult := applyAdminLoginAuditFilters(
		r.db.Model(&model.AdminLoginAudit{}),
		keyword,
		role,
		stage,
		result,
		nil,
		nil,
	).Where("created_at < ?", before).Delete(&model.AdminLoginAudit{})
	return dbResult.RowsAffected, dbResult.Error
}
