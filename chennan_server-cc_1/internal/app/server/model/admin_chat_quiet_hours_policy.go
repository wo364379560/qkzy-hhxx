package model

import "time"

// AdminChatQuietHoursPolicy 表示后台账号级别的静默时段策略。
type AdminChatQuietHoursPolicy struct {
	ID               uint   `gorm:"primaryKey"`
	AdminUserID      uint   `gorm:"uniqueIndex;not null"`
	Enabled          bool   `gorm:"default:false"`
	TimeZone         string `gorm:"size:64;not null"`
	Start            string `gorm:"size:5;not null"`
	End              string `gorm:"size:5;not null"`
	BlockRegister    bool   `gorm:"default:true"`
	BlockSendMessage bool   `gorm:"default:true"`
	BlockCreateGroup bool   `gorm:"default:true"`
	BlockAddFriend   bool   `gorm:"default:true"`
	UpdatedBy        uint

	CreatedAt time.Time
	UpdatedAt time.Time
}
