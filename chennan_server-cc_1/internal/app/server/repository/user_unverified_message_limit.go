package repository

import (
	"errors"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserUnverifiedMessageLimitRepository handles cumulative send quota for unverified users.
type UserUnverifiedMessageLimitRepository struct {
	db *gorm.DB
}

func NewUserUnverifiedMessageLimitRepository() *UserUnverifiedMessageLimitRepository {
	return &UserUnverifiedMessageLimitRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func NewUserUnverifiedMessageLimitRepositoryWithDB(db *gorm.DB) *UserUnverifiedMessageLimitRepository {
	return &UserUnverifiedMessageLimitRepository{db: db}
}

func (r *UserUnverifiedMessageLimitRepository) CountByUserID(userID uint) (int64, error) {
	if userID == 0 {
		return 0, nil
	}
	var row model.UserUnverifiedMessageLimit
	err := r.db.Where("user_id = ?", userID).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return row.SentCount, nil
}

func (r *UserUnverifiedMessageLimitRepository) UpsertCountAtLeast(user *model.User, count int64, now time.Time) error {
	if user == nil || user.ID == 0 || count <= 0 {
		return nil
	}
	imUserID := strings.TrimSpace(user.IMUserID)
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoNothing: true,
		}).Create(&model.UserUnverifiedMessageLimit{
			UserID:      user.ID,
			IMUserID:    imUserID,
			SentCount:   count,
			FirstSentAt: &now,
			LastSentAt:  &now,
		}).Error; err != nil {
			return err
		}

		var row model.UserUnverifiedMessageLimit
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", user.ID).
			First(&row).Error
		if err != nil {
			return err
		}
		updates := map[string]any{
			"im_user_id":   imUserID,
			"last_sent_at": now,
			"sent_count":   gorm.Expr("GREATEST(sent_count, ?)", count),
		}
		if row.FirstSentAt == nil {
			updates["first_sent_at"] = now
		}
		return tx.Model(&row).Updates(updates).Error
	})
}

func (r *UserUnverifiedMessageLimitRepository) TryConsume(user *model.User, limit int64, now time.Time) (bool, int64, error) {
	if user == nil || user.ID == 0 {
		return true, 0, nil
	}
	if limit <= 0 {
		return false, 0, nil
	}
	imUserID := strings.TrimSpace(user.IMUserID)
	var savedCount int64
	allowed := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoNothing: true,
		}).Create(&model.UserUnverifiedMessageLimit{
			UserID:    user.ID,
			IMUserID:  imUserID,
			SentCount: 0,
		}).Error; err != nil {
			return err
		}

		var row model.UserUnverifiedMessageLimit
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", user.ID).
			First(&row).Error
		if err != nil {
			return err
		}
		if row.SentCount >= limit {
			savedCount = row.SentCount
			allowed = false
			return nil
		}
		row.SentCount++
		row.IMUserID = imUserID
		if row.FirstSentAt == nil {
			row.FirstSentAt = &now
		}
		row.LastSentAt = &now
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		savedCount = row.SentCount
		allowed = true
		return nil
	})
	return allowed, savedCount, err
}

func (r *UserUnverifiedMessageLimitRepository) ResetAllSentCounts() error {
	return r.db.Exec("UPDATE user_unverified_message_limits SET sent_count = 0").Error
}
