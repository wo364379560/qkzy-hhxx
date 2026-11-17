package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

const (
	assignedRealNamePlaceholder = "***"
	assignedIDNumberMasked      = "******************"
)

// UserRealNameRepository handles user_real_names table operations.
type UserRealNameRepository struct {
	db *gorm.DB
}

func NewUserRealNameRepository() *UserRealNameRepository {
	return &UserRealNameRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func NewUserRealNameRepositoryWithDB(db *gorm.DB) *UserRealNameRepository {
	return &UserRealNameRepository{db: db}
}

func (r *UserRealNameRepository) FindByUserID(userID uint) (*model.UserRealName, error) {
	var record model.UserRealName
	if err := r.db.Where("user_id = ?", userID).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *UserRealNameRepository) FindByUserIDs(userIDs []uint) (map[uint]*model.UserRealName, error) {
	result := make(map[uint]*model.UserRealName, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	var rows []model.UserRealName
	if err := r.db.Where("user_id IN ?", userIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		result[rows[i].UserID] = &rows[i]
	}
	return result, nil
}

func (r *UserRealNameRepository) FindVerifiedByIDNumberHash(idNumberHash string) (*model.UserRealName, error) {
	if idNumberHash == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var record model.UserRealName
	if err := r.db.Where("id_number_hash = ? AND status = ?", idNumberHash, model.UserRealNameStatusVerified).
		First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *UserRealNameRepository) UpsertVerified(record *model.UserRealName) error {
	if record == nil {
		return errors.New("record is nil")
	}

	var existing model.UserRealName
	err := r.db.Where("user_id = ?", record.UserID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.Create(record).Error
	}
	if err != nil {
		return err
	}

	record.ID = existing.ID
	return r.db.Save(record).Error
}

func (r *UserRealNameRepository) SaveFailed(userID uint, idNumberHash, idNumberMasked string) error {
	var existing model.UserRealName
	err := r.db.Where("user_id = ?", userID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.Create(&model.UserRealName{
			UserID:         userID,
			IDNumberHash:   idNumberHash,
			IDNumberMasked: idNumberMasked,
			Status:         model.UserRealNameStatusFailed,
		}).Error
	}
	if err != nil {
		return err
	}

	return r.db.Model(&existing).Updates(map[string]any{
		"id_number_hash":   idNumberHash,
		"id_number_masked": idNumberMasked,
		"status":           model.UserRealNameStatusFailed,
		"verified_at":      nil,
		"aliyun_request_id": "",
	}).Error
}

func (r *UserRealNameRepository) Reset(userID uint) error {
	var existing model.UserRealName
	err := r.db.Where("user_id = ?", userID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	return r.db.Model(&existing).Updates(map[string]any{
		"real_name":          "",
		"id_number_hash":     "",
		"id_number_masked":   "",
		"status":             model.UserRealNameStatusNone,
		"verified_at":        nil,
		"aliyun_request_id":  "",
		"updated_at":         time.Now(),
	}).Error
}

// UpsertAssignedVerified 为已归属（非超管运营方）用户写入自动实名占位记录。
func (r *UserRealNameRepository) UpsertAssignedVerified(userID uint) error {
	existing, err := r.FindByUserID(userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if existing != nil && existing.Status == model.UserRealNameStatusVerified {
		return nil
	}
	now := time.Now()
	idNumber := fmt.Sprintf("assigned-%d", userID)
	return r.UpsertVerified(&model.UserRealName{
		UserID:         userID,
		RealName:       assignedRealNamePlaceholder,
		IDNumberHash:   hashAssignedIDNumber(idNumber),
		IDNumberMasked: assignedIDNumberMasked,
		Status:         model.UserRealNameStatusVerified,
		VerifiedAt:     &now,
	})
}

func hashAssignedIDNumber(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}
