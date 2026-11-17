package repository

import (
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

// HabitRepository 处理 habits 与打卡记录的数据库操作。
type HabitRepository struct {
	db *gorm.DB
}

// NewHabitRepository 创建 HabitRepository。
func NewHabitRepository() *HabitRepository {
	return &HabitRepository{db: datasource.Datasource.Mysql.GetClient()}
}

// CreateHabit 创建用户习惯。
func (r *HabitRepository) CreateHabit(habit *model.UserHabit) error {
	return r.db.Create(habit).Error
}

// ListActiveHabitsByUser 查询用户所有启用中的习惯。
func (r *HabitRepository) ListActiveHabitsByUser(userID uint) ([]model.UserHabit, error) {
	var habits []model.UserHabit
	err := r.db.
		Where("user_id = ? AND status = 'active'", userID).
		Order("id ASC").
		Find(&habits).Error
	return habits, err
}

// FindHabitByIDAndUser 查询指定用户的习惯。
func (r *HabitRepository) FindHabitByIDAndUser(habitID, userID uint) (*model.UserHabit, error) {
	var habit model.UserHabit
	if err := r.db.
		Where("id = ? AND user_id = ?", habitID, userID).
		First(&habit).Error; err != nil {
		return nil, err
	}
	return &habit, nil
}

// FindHabitByID 查询单个习惯。
func (r *HabitRepository) FindHabitByID(habitID uint) (*model.UserHabit, error) {
	var habit model.UserHabit
	if err := r.db.First(&habit, habitID).Error; err != nil {
		return nil, err
	}
	return &habit, nil
}

// FindTodayCheckin 查询当天打卡记录。
func (r *HabitRepository) FindTodayCheckin(habitID uint, dayStart time.Time) (*model.HabitCheckin, error) {
	var checkin model.HabitCheckin
	if err := r.db.
		Where("user_habit_id = ? AND checkin_date = ?", habitID, dayStart).
		First(&checkin).Error; err != nil {
		return nil, err
	}
	return &checkin, nil
}

// SaveCheckin 创建或更新打卡记录。
func (r *HabitRepository) SaveCheckin(checkin *model.HabitCheckin) error {
	return r.db.Save(checkin).Error
}

// ListCheckinsByHabit 查询某习惯的全部已完成打卡记录。
func (r *HabitRepository) ListCheckinsByHabit(habitID uint) ([]model.HabitCheckin, error) {
	var checkins []model.HabitCheckin
	err := r.db.
		Where("user_habit_id = ? AND status = 'completed'", habitID).
		Order("checkin_date DESC").
		Find(&checkins).Error
	return checkins, err
}

// ListCheckinsByUserAndDate 查询某用户指定日期的所有打卡记录。
func (r *HabitRepository) ListCheckinsByUserAndDate(userID uint, dayStart time.Time) ([]model.HabitCheckin, error) {
	var checkins []model.HabitCheckin
	err := r.db.
		Where("user_id = ? AND checkin_date = ?", userID, dayStart).
		Find(&checkins).Error
	return checkins, err
}

// ListCompletedCheckinsByUsersUpToDate 查询一组用户截至指定日期的全部已完成打卡记录。
func (r *HabitRepository) ListCompletedCheckinsByUsersUpToDate(userIDs []uint, endDate time.Time) ([]model.HabitCheckin, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	var checkins []model.HabitCheckin
	err := r.db.
		Where("user_id IN ? AND status = 'completed' AND checkin_date <= ?", userIDs, endDate).
		Order("user_id ASC, checkin_date DESC, checked_in_at DESC").
		Find(&checkins).Error
	return checkins, err
}

// CountCompletedCheckinsInRange 统计时间范围内完成的打卡数。
func (r *HabitRepository) CountCompletedCheckinsInRange(userID uint, startDate, endDate time.Time) (int64, error) {
	var count int64
	err := r.db.
		Model(&model.HabitCheckin{}).
		Where("user_id = ? AND status = 'completed' AND checkin_date >= ? AND checkin_date <= ?", userID, startDate, endDate).
		Count(&count).Error
	return count, err
}

// CountCompletedCheckinsByHabitAndUsersInRange 统计指定习惯在一组用户中的完成打卡数。
func (r *HabitRepository) CountCompletedCheckinsByHabitAndUsersInRange(habitID uint, userIDs []uint, startDate, endDate time.Time) (int64, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}
	var count int64
	err := r.db.
		Model(&model.HabitCheckin{}).
		Where("user_habit_id = ? AND user_id IN ? AND status = 'completed' AND checkin_date >= ? AND checkin_date <= ?", habitID, userIDs, startDate, endDate).
		Count(&count).Error
	return count, err
}

// CountCompletedCheckinsByHabitAndUsersOnDay 统计指定习惯在一组用户中的当日完成打卡数。
func (r *HabitRepository) CountCompletedCheckinsByHabitAndUsersOnDay(habitID uint, userIDs []uint, day time.Time) (int64, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}
	var count int64
	err := r.db.
		Model(&model.HabitCheckin{}).
		Where("user_habit_id = ? AND user_id IN ? AND status = 'completed' AND checkin_date = ?", habitID, userIDs, day).
		Count(&count).Error
	return count, err
}
