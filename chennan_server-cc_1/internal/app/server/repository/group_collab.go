package repository

import (
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

type GroupCollabRepository struct {
	db *gorm.DB
}

func NewGroupCollabRepository() *GroupCollabRepository {
	return &GroupCollabRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func (r *GroupCollabRepository) ListTodos(imGroupID string) ([]model.GroupTodo, error) {
	var rows []model.GroupTodo
	err := r.db.
		Where("im_group_id = ? AND status IN ?", imGroupID, []string{model.GroupTodoStatusOpen, model.GroupTodoStatusDone}).
		Order("CASE WHEN status = 'open' THEN 0 ELSE 1 END ASC, updated_at DESC").
		Find(&rows).Error
	return rows, err
}

func (r *GroupCollabRepository) CreateTodo(row *model.GroupTodo) error {
	return r.db.Create(row).Error
}

func (r *GroupCollabRepository) FindTodo(id uint, imGroupID string) (*model.GroupTodo, error) {
	var row model.GroupTodo
	err := r.db.Where("id = ? AND im_group_id = ?", id, imGroupID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *GroupCollabRepository) UpdateTodo(id uint, fields map[string]any) error {
	return r.db.Model(&model.GroupTodo{}).Where("id = ?", id).Updates(fields).Error
}

func (r *GroupCollabRepository) ListEvents(imGroupID string, from time.Time) ([]model.GroupSharedEvent, error) {
	var rows []model.GroupSharedEvent
	err := r.db.
		Where("im_group_id = ? AND status = ? AND start_at >= ?", imGroupID, model.GroupCollabStatusActive, from).
		Order("start_at ASC").
		Limit(100).
		Find(&rows).Error
	return rows, err
}

func (r *GroupCollabRepository) CreateEvent(row *model.GroupSharedEvent) error {
	return r.db.Create(row).Error
}

func (r *GroupCollabRepository) FindEvent(id uint, imGroupID string) (*model.GroupSharedEvent, error) {
	var row model.GroupSharedEvent
	err := r.db.Where("id = ? AND im_group_id = ?", id, imGroupID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *GroupCollabRepository) SoftDeleteEvent(id uint) error {
	return r.db.Model(&model.GroupSharedEvent{}).
		Where("id = ?", id).
		Update("status", model.GroupCollabStatusDeleted).Error
}

func (r *GroupCollabRepository) ListLedgers(imGroupID string, limit int) ([]model.GroupSharedLedger, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var rows []model.GroupSharedLedger
	err := r.db.
		Where("im_group_id = ? AND status = ?", imGroupID, model.GroupCollabStatusActive).
		Order("record_date DESC, id DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *GroupCollabRepository) CreateLedger(row *model.GroupSharedLedger) error {
	return r.db.Create(row).Error
}

func (r *GroupCollabRepository) FindLedger(id uint, imGroupID string) (*model.GroupSharedLedger, error) {
	var row model.GroupSharedLedger
	err := r.db.Where("id = ? AND im_group_id = ?", id, imGroupID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *GroupCollabRepository) SoftDeleteLedger(id uint) error {
	return r.db.Model(&model.GroupSharedLedger{}).
		Where("id = ?", id).
		Update("status", model.GroupCollabStatusDeleted).Error
}

func (r *GroupCollabRepository) SumLedgers(imGroupID string) (expense float64, income float64, err error) {
	type sumRow struct {
		Type  string
		Total float64
	}
	var rows []sumRow
	err = r.db.Model(&model.GroupSharedLedger{}).
		Select("type, COALESCE(SUM(amount), 0) as total").
		Where("im_group_id = ? AND status = ?", imGroupID, model.GroupCollabStatusActive).
		Group("type").
		Scan(&rows).Error
	if err != nil {
		return 0, 0, err
	}
	for _, row := range rows {
		switch row.Type {
		case model.GroupLedgerTypeExpense:
			expense = row.Total
		case model.GroupLedgerTypeIncome:
			income = row.Total
		}
	}
	return expense, income, nil
}
