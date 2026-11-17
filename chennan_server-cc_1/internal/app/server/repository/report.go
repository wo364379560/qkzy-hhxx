package repository

import (
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

// ReportRepository 处理 reports 表的数据库操作。
type ReportRepository struct {
	db *gorm.DB
}

func NewReportRepository() *ReportRepository {
	return &ReportRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func (r *ReportRepository) Create(report *model.Report) error {
	return r.db.Create(report).Error
}

func (r *ReportRepository) FindByID(id uint) (*model.Report, error) {
	var report model.Report
	if err := r.db.First(&report, id).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *ReportRepository) Update(report *model.Report) error {
	return r.db.Save(report).Error
}

func (r *ReportRepository) UpdateStatus(id uint, status string) error {
	return r.db.Model(&model.Report{}).Where("id = ?", id).Update("status", status).Error
}

// CountByStatus 统计指定状态的举报数量，status="" 表示全部。
func (r *ReportRepository) CountByStatus(status string) (int64, error) {
	var count int64
	q := r.db.Model(&model.Report{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Count(&count).Error
	return count, err
}

// CountByOwnerAndStatus 统计某 owner 指定状态的举报数量，status="" 表示全部。
func (r *ReportRepository) CountByOwnerAndStatus(ownerID uint, status string) (int64, error) {
	var count int64
	q := r.db.Model(&model.Report{}).Where("owner_id = ?", ownerID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Count(&count).Error
	return count, err
}

// CountByOwnersAndStatus 统计指定 owner 集合下的举报数量，status="" 表示全部。
func (r *ReportRepository) CountByOwnersAndStatus(ownerIDs []uint, status string) (int64, error) {
	if len(ownerIDs) == 0 {
		return 0, nil
	}
	var count int64
	q := r.db.Model(&model.Report{}).Where("owner_id IN ?", ownerIDs)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Count(&count).Error
	return count, err
}

// ListPaged 分页查询举报列表。ownerID=0 时使用 scopeOwnerIDs 过滤；scopeOwnerIDs 为空表示不限制归属。
func (r *ReportRepository) ListPaged(page, pageSize int, scopeOwnerIDs []uint, ownerID uint, status string) ([]model.Report, int64, error) {
	q := r.db.Model(&model.Report{})
	if ownerID > 0 {
		q = q.Where("owner_id = ?", ownerID)
	} else if len(scopeOwnerIDs) > 0 {
		q = q.Where("owner_id IN ?", scopeOwnerIDs)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var reports []model.Report
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&reports).Error
	return reports, total, err
}
