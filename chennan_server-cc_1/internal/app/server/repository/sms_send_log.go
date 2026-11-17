package repository

import (
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

type SMSSendLogRepository struct {
	db *gorm.DB
}

func NewSMSSendLogRepository() *SMSSendLogRepository {
	return &SMSSendLogRepository{db: datasource.Datasource.Mysql.GetClient()}
}

func NewSMSSendLogRepositoryWithDB(db *gorm.DB) *SMSSendLogRepository {
	return &SMSSendLogRepository{db: db}
}

func (r *SMSSendLogRepository) Create(log *model.SMSSendLog) error {
	return r.db.Create(log).Error
}

func (r *SMSSendLogRepository) CountByPhoneSince(phone string, since time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&model.SMSSendLog{}).
		Where("phone = ? AND created_at >= ? AND status = ?", phone, since, "success").
		Count(&count).Error
	return count, err
}

func (r *SMSSendLogRepository) CountByIPSince(ip string, since time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&model.SMSSendLog{}).
		Where("ip_address = ? AND created_at >= ? AND status = ?", ip, since, "success").
		Count(&count).Error
	return count, err
}

func (r *SMSSendLogRepository) CountByDeviceSince(deviceID string, since time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&model.SMSSendLog{}).
		Where("device_id = ? AND created_at >= ? AND status = ?", deviceID, since, "success").
		Count(&count).Error
	return count, err
}

func (r *SMSSendLogRepository) FindLatestSuccessByPhone(phone string) (*model.SMSSendLog, error) {
	var log model.SMSSendLog
	if err := r.db.
		Where("phone = ? AND status = ?", phone, "success").
		Order("created_at DESC").
		First(&log).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *SMSSendLogRepository) CountAllSince(since time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&model.SMSSendLog{}).
		Where("created_at >= ?", since).
		Count(&count).Error
	return count, err
}

func (r *SMSSendLogRepository) CountBySceneSince(scene string, since time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&model.SMSSendLog{}).
		Where("scene = ? AND created_at >= ?", scene, since).
		Count(&count).Error
	return count, err
}

func (r *SMSSendLogRepository) ListPaged(page, pageSize int, keyword, scene, status string) ([]model.SMSSendLog, int64, error) {
	q := r.db.Model(&model.SMSSendLog{})
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("phone LIKE ? OR ip_address LIKE ? OR device_id LIKE ? OR reject_reason LIKE ?", like, like, like, like)
	}
	if scene = strings.TrimSpace(scene); scene != "" {
		q = q.Where("scene = ?", scene)
	}
	if status = strings.TrimSpace(status); status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []model.SMSSendLog
	offset := (page - 1) * pageSize
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
