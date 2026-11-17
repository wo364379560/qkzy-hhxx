package repository

import (
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

type GroupMessagePinRepository struct {
	db *gorm.DB
}

func NewGroupMessagePinRepository() *GroupMessagePinRepository {
	return &GroupMessagePinRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func NewGroupMessagePinRepositoryWithDB(db *gorm.DB) *GroupMessagePinRepository {
	return &GroupMessagePinRepository{db: db}
}

func (r *GroupMessagePinRepository) FindActiveByGroupID(groupID uint) (*model.GroupMessagePin, error) {
	var pin model.GroupMessagePin
	if err := r.db.Where("group_id = ? AND status = 'active'", groupID).Order("id DESC").First(&pin).Error; err != nil {
		return nil, err
	}
	return &pin, nil
}

func (r *GroupMessagePinRepository) ReplaceActive(pin *model.GroupMessagePin) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.GroupMessagePin{}).
			Where("group_id = ? AND status = 'active'", pin.GroupID).
			Updates(map[string]any{
				"status":     "removed",
				"updated_at": time.Now(),
			}).Error; err != nil {
			return err
		}
		return tx.Create(pin).Error
	})
}

func (r *GroupMessagePinRepository) RemoveActiveByGroupID(groupID uint) error {
	return r.db.Model(&model.GroupMessagePin{}).
		Where("group_id = ? AND status = 'active'", groupID).
		Updates(map[string]any{
			"status":     "removed",
			"updated_at": time.Now(),
		}).Error
}

func (r *GroupMessagePinRepository) RemoveActiveByMessageID(imGroupID, messageID string) error {
	return r.db.Model(&model.GroupMessagePin{}).
		Where("im_group_id = ? AND message_id = ? AND status = 'active'", imGroupID, messageID).
		Updates(map[string]any{
			"status":     "removed",
			"updated_at": time.Now(),
		}).Error
}
