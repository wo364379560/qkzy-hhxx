package repository

import (
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

// UserOperationLogRepository handles mobile user operation log persistence.
type UserOperationLogRepository struct {
	db *gorm.DB
}

// NewUserOperationLogRepository creates a repository using the default DB.
func NewUserOperationLogRepository() *UserOperationLogRepository {
	return &UserOperationLogRepository{db: datasource.Datasource.Mysql.GetClient()}
}

// Create writes one user operation log.
func (r *UserOperationLogRepository) Create(log *model.UserOperationLog) error {
	return r.db.Create(log).Error
}

// ListUserOperationLogFilter contains admin query filters.
type ListUserOperationLogFilter struct {
	Keyword       string
	OperationType string
	Status        string
	Phone         string
	IMUserID      string
	OwnerID       uint
	IPAddress     string
	StartTime     *time.Time
	EndTime       *time.Time
	ScopeOwnerIDs []uint
}

func applyUserOperationLogFilters(q *gorm.DB, filter ListUserOperationLogFilter) *gorm.DB {
	if len(filter.ScopeOwnerIDs) > 0 {
		q = q.Where("owner_id IN ?", filter.ScopeOwnerIDs)
	}
	if filter.OperationType != "" {
		q = q.Where("operation_type = ?", filter.OperationType)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.Phone = strings.TrimSpace(filter.Phone); filter.Phone != "" {
		q = q.Where("phone = ?", filter.Phone)
	}
	if filter.IMUserID = strings.TrimSpace(filter.IMUserID); filter.IMUserID != "" {
		q = q.Where("im_user_id = ?", filter.IMUserID)
	}
	if filter.OwnerID > 0 {
		q = q.Where("owner_id = ?", filter.OwnerID)
	}
	if filter.IPAddress = strings.TrimSpace(filter.IPAddress); filter.IPAddress != "" {
		q = q.Where("ip_address = ?", filter.IPAddress)
	}
	if filter.StartTime != nil {
		q = q.Where("created_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		q = q.Where("created_at <= ?", *filter.EndTime)
	}
	if filter.Keyword = strings.TrimSpace(filter.Keyword); filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		q = q.Where(
			"phone LIKE ? OR im_user_id LIKE ? OR nickname LIKE ? OR owner_name LIKE ?",
			like,
			like,
			like,
			like,
		)
	}
	return q
}

// ListPaged returns paginated user operation logs.
func (r *UserOperationLogRepository) ListPaged(page, pageSize int, filter ListUserOperationLogFilter) ([]model.UserOperationLog, int64, error) {
	q := applyUserOperationLogFilters(r.db.Model(&model.UserOperationLog{}), filter)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []model.UserOperationLog
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}
