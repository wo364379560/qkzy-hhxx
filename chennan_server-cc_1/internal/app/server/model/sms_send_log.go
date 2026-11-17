package model

import "time"

// SMSSendLog 记录短信发送请求日志，用于频控和审计。
type SMSSendLog struct {
	ID           uint      `gorm:"primaryKey"`
	Phone        string    `gorm:"size:32;not null;index:idx_sms_send_logs_phone_created_at"`
	IPAddress    string    `gorm:"size:64;not null;default:'';index:idx_sms_send_logs_ip_created_at"`
	DeviceID     string    `gorm:"size:128;not null;default:'';index:idx_sms_send_logs_device_created_at"`
	Scene        string    `gorm:"size:32;not null;default:'general';index:idx_sms_send_logs_scene_created_at"`
	Status       string    `gorm:"size:16;not null;index:idx_sms_send_logs_status_created_at"`
	RejectReason string    `gorm:"size:128;not null;default:''"`
	Provider     string    `gorm:"size:32;not null;default:''"`
	CreatedAt    time.Time `gorm:"index"`
}
