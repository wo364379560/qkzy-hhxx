package model

import "time"

// UserHabit 表示用户创建的习惯实例。
type UserHabit struct {
	ID           uint   `gorm:"primaryKey"`
	UserID       uint   `gorm:"index;not null"`
	Name         string `gorm:"size:64;not null"`
	Icon         string `gorm:"size:32"`
	GoalType     string `gorm:"size:16;not null"` // binary / numeric / timer
	GoalTarget   string `gorm:"size:64"`
	ReminderTime string `gorm:"size:8"` // HH:mm
	Status       string `gorm:"type:enum('active','archived');default:'active'"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// HabitCheckin 表示习惯在某一天的打卡记录。
type HabitCheckin struct {
	ID          uint      `gorm:"primaryKey"`
	UserHabitID uint      `gorm:"not null;uniqueIndex:idx_habit_checkin_day"`
	UserID      uint      `gorm:"index;not null"`
	CheckinDate time.Time `gorm:"type:date;not null;uniqueIndex:idx_habit_checkin_day"`
	Value       string    `gorm:"size:64"`
	Status      string    `gorm:"size:16;default:'completed'"`
	CheckedInAt time.Time `gorm:"not null"`
	Source      string    `gorm:"size:16;default:'manual'"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
