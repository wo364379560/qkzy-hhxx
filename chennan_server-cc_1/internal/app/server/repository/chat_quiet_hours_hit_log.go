package repository

import (
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

// ChatQuietHoursHitLogRepository handles persistence of quiet-hours hit logs.
type ChatQuietHoursHitLogRepository struct {
	db *gorm.DB
}

// NewChatQuietHoursHitLogRepository creates a repository using the default DB.
func NewChatQuietHoursHitLogRepository() *ChatQuietHoursHitLogRepository {
	return &ChatQuietHoursHitLogRepository{db: datasource.Datasource.Mysql.GetClient()}
}

// Create inserts a quiet-hours hit log row.
func (r *ChatQuietHoursHitLogRepository) Create(log *model.ChatQuietHoursHitLog) error {
	return r.db.Create(log).Error
}

func applyChatQuietHoursHitLogFilters(q *gorm.DB, keyword, action, source string, scopeOwnerIDs []uint) *gorm.DB {
	if len(scopeOwnerIDs) > 0 {
		q = q.Where("owner_id IN ?", scopeOwnerIDs)
	}
	if action = strings.TrimSpace(action); action != "" {
		q = q.Where("action = ?", action)
	}
	if source = strings.TrimSpace(source); source != "" {
		q = q.Where("source = ?", source)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where(
			"im_user_id LIKE ? OR phone LIKE ? OR target_account LIKE ? OR target_group_id LIKE ? OR ip_address LIKE ? OR reason LIKE ?",
			like, like, like, like, like, like,
		)
	}
	return q
}

// ListPaged queries quiet-hours hit logs with pagination and filters.
func (r *ChatQuietHoursHitLogRepository) ListPaged(page, pageSize int, keyword, action, source string, scopeOwnerIDs []uint) ([]model.ChatQuietHoursHitLog, int64, error) {
	q := applyChatQuietHoursHitLogFilters(
		r.db.Model(&model.ChatQuietHoursHitLog{}),
		keyword,
		action,
		source,
		scopeOwnerIDs,
	)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.ChatQuietHoursHitLog
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&rows).Error
	return rows, total, err
}
