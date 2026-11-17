package model

import "time"

// SystemSetting 系统级配置项。
type SystemSetting struct {
	SettingKey string `gorm:"column:setting_key;primaryKey;size:64"`
	Value      string `gorm:"type:text;not null"`
	UpdatedBy  *uint

	CreatedAt time.Time
	UpdatedAt time.Time
}
