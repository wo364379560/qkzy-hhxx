package repository

import (
	"errors"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OfflineActivityRepository 处理线下活动及报名。
type OfflineActivityRepository struct {
	db *gorm.DB
}

func NewOfflineActivityRepository() *OfflineActivityRepository {
	return &OfflineActivityRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func (r *OfflineActivityRepository) ListPaged(page, pageSize int, statuses []string) ([]model.OfflineActivity, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	q := r.db.Model(&model.OfflineActivity{})
	if len(statuses) > 0 {
		q = q.Where("status IN ?", statuses)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.OfflineActivity
	err := q.Order("start_at ASC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	return items, total, err
}

func (r *OfflineActivityRepository) ListPublished() ([]model.OfflineActivity, error) {
	var items []model.OfflineActivity
	err := r.db.
		Where("status = ?", "published").
		Order("start_at ASC, id DESC").
		Find(&items).Error
	return items, err
}

func (r *OfflineActivityRepository) Create(activity *model.OfflineActivity) error {
	return r.db.Create(activity).Error
}

func (r *OfflineActivityRepository) Update(activity *model.OfflineActivity) error {
	return r.db.Save(activity).Error
}

func (r *OfflineActivityRepository) FindByID(activityID uint) (*model.OfflineActivity, error) {
	var activity model.OfflineActivity
	if err := r.db.First(&activity, activityID).Error; err != nil {
		return nil, err
	}
	return &activity, nil
}

func (r *OfflineActivityRepository) ListRegisteredActivityIDs(userID uint, activityIDs []uint) (map[uint]struct{}, error) {
	if len(activityIDs) == 0 {
		return map[uint]struct{}{}, nil
	}
	var rows []model.OfflineActivityRegistration
	if err := r.db.
		Where("user_id = ? AND status = 'registered' AND activity_id IN ?", userID, activityIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[uint]struct{}, len(rows))
	for _, row := range rows {
		result[row.ActivityID] = struct{}{}
	}
	return result, nil
}

func (r *OfflineActivityRepository) ToggleRegistration(activityID, userID uint) (bool, int, error) {
	var registered bool
	var registrationCount int

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var activity model.OfflineActivity
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&activity, activityID).Error; err != nil {
			return err
		}

		var registration model.OfflineActivityRegistration
		err := tx.
			Where("activity_id = ? AND user_id = ?", activityID, userID).
			First(&registration).Error

		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			if activity.Status != "published" {
				return errors.New("活动当前不可报名")
			}
			if activity.Quota > 0 && activity.RegistrationCount >= activity.Quota {
				return errors.New("报名人数已满")
			}
			now := time.Now()
			registration = model.OfflineActivityRegistration{
				ActivityID: activityID,
				UserID:     userID,
				Status:     "registered",
				JoinedAt:   now,
			}
			if err := tx.Create(&registration).Error; err != nil {
				return err
			}
			if err := tx.Model(&activity).
				UpdateColumn("registration_count", gorm.Expr("registration_count + ?", 1)).Error; err != nil {
				return err
			}
			registered = true
		case err != nil:
			return err
		default:
			if registration.Status == "registered" {
				now := time.Now()
				registration.Status = "cancelled"
				registration.CancelledAt = &now
				if err := tx.Save(&registration).Error; err != nil {
					return err
				}
				if err := tx.Model(&activity).
					UpdateColumn("registration_count", gorm.Expr("GREATEST(registration_count - 1, 0)")).Error; err != nil {
					return err
				}
				registered = false
			} else {
				if activity.Status != "published" {
					return errors.New("活动当前不可报名")
				}
				if activity.Quota > 0 && activity.RegistrationCount >= activity.Quota {
					return errors.New("报名人数已满")
				}
				registration.Status = "registered"
				registration.CancelledAt = nil
				registration.JoinedAt = time.Now()
				if err := tx.Save(&registration).Error; err != nil {
					return err
				}
				if err := tx.Model(&activity).
					UpdateColumn("registration_count", gorm.Expr("registration_count + ?", 1)).Error; err != nil {
					return err
				}
				registered = true
			}
		}

		if err := tx.Model(&model.OfflineActivity{}).
			Select("registration_count").
			Where("id = ?", activityID).
			Scan(&registrationCount).Error; err != nil {
			return err
		}
		return nil
	})

	return registered, registrationCount, err
}

func (r *OfflineActivityRepository) ListRegistrations(activityID uint) ([]model.OfflineActivityRegistration, error) {
	var rows []model.OfflineActivityRegistration
	err := r.db.
		Where("activity_id = ? AND status = 'registered'", activityID).
		Order("joined_at ASC, id ASC").
		Find(&rows).Error
	return rows, err
}
