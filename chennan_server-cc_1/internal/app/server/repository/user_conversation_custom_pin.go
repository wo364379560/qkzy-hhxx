package repository

import (
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

type UserConversationCustomPinRepository struct {
	db *gorm.DB
}

func NewUserConversationCustomPinRepository() *UserConversationCustomPinRepository {
	return &UserConversationCustomPinRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func (r *UserConversationCustomPinRepository) ListByOwner(ownerUserID uint) ([]model.UserConversationCustomPin, error) {
	var rows []model.UserConversationCustomPin
	err := r.db.
		Where("owner_user_id = ?", ownerUserID).
		Order("pin_order ASC, pinned_at ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *UserConversationCustomPinRepository) Upsert(ownerUserID uint, conversationID string, pinOrder int) error {
	now := time.Now()
	record := &model.UserConversationCustomPin{
		OwnerUserID:    ownerUserID,
		ConversationID: conversationID,
		PinOrder:       pinOrder,
		PinnedAt:       now,
	}
	return r.db.Where("owner_user_id = ? AND conversation_id = ?", ownerUserID, conversationID).
		Assign(map[string]any{
			"pin_order": pinOrder,
			"pinned_at": now,
		}).
		FirstOrCreate(record).Error
}

func (r *UserConversationCustomPinRepository) Delete(ownerUserID uint, conversationID string) error {
	return r.db.
		Where("owner_user_id = ? AND conversation_id = ?", ownerUserID, conversationID).
		Delete(&model.UserConversationCustomPin{}).Error
}

func (r *UserConversationCustomPinRepository) Reorder(ownerUserID uint, conversationIDs []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		for index, conversationID := range conversationIDs {
			if err := tx.Model(&model.UserConversationCustomPin{}).
				Where("owner_user_id = ? AND conversation_id = ?", ownerUserID, conversationID).
				Updates(map[string]any{
					"pin_order": index + 1,
					"pinned_at": now,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
