package repository

import (
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

// ChatKeywordRepository 处理 chat_keywords 表的数据库操作。
type ChatKeywordRepository struct {
	db *gorm.DB
}

// NewChatKeywordRepository 创建关键词仓储实例。
func NewChatKeywordRepository() *ChatKeywordRepository {
	return &ChatKeywordRepository{db: datasource.Datasource.Mysql.GetClient()}
}

// ListPaged 分页查询关键词规则。
func (r *ChatKeywordRepository) ListPaged(page, pageSize int, keyword string) ([]model.ChatKeyword, int64, error) {
	q := r.db.Model(&model.ChatKeyword{})
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("keyword LIKE ? OR replacement LIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.ChatKeyword
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&rows).Error
	return rows, total, err
}

// ListEnabled 查询所有启用中的关键词。
func (r *ChatKeywordRepository) ListEnabled() ([]model.ChatKeyword, error) {
	var rows []model.ChatKeyword
	err := r.db.Where("enabled = ?", true).Order("id DESC").Find(&rows).Error
	return rows, err
}

// ListAll 查询全部关键词规则。
func (r *ChatKeywordRepository) ListAll() ([]model.ChatKeyword, error) {
	var rows []model.ChatKeyword
	err := r.db.Order("id DESC").Find(&rows).Error
	return rows, err
}

// FindByID 按主键查询关键词。
func (r *ChatKeywordRepository) FindByID(id uint) (*model.ChatKeyword, error) {
	var row model.ChatKeyword
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// FindByKeyword 按关键词精确查询。
func (r *ChatKeywordRepository) FindByKeyword(keyword string) (*model.ChatKeyword, error) {
	var row model.ChatKeyword
	if err := r.db.Where("keyword = ?", strings.TrimSpace(keyword)).Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Create 新增关键词规则。
func (r *ChatKeywordRepository) Create(row *model.ChatKeyword) error {
	return r.db.Create(row).Error
}

// Update 更新关键词规则字段。
func (r *ChatKeywordRepository) Update(id uint, fields map[string]any) error {
	return r.db.Model(&model.ChatKeyword{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除关键词规则。
func (r *ChatKeywordRepository) Delete(id uint) error {
	return r.db.Delete(&model.ChatKeyword{}, id).Error
}
