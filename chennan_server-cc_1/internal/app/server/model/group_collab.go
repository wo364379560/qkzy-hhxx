package model

import "time"

const (
	GroupTodoStatusOpen      = "open"
	GroupTodoStatusDone      = "done"
	GroupCollabStatusActive  = "active"
	GroupCollabStatusDeleted = "deleted"
)

// GroupTodo 群待办（协作）。
type GroupTodo struct {
	ID               uint   `gorm:"primaryKey"`
	IMGroupID        string `gorm:"size:64;not null;index:idx_group_todos_group_status"`
	Title            string `gorm:"size:200;not null"`
	Description      string `gorm:"size:2000;not null;default:''"`
	ImageURLs        string `gorm:"type:text"` // JSON 字符串数组
	Status           string `gorm:"size:16;not null;default:'open';index:idx_group_todos_group_status"`
	CreatorIMUserID  string `gorm:"size:64;not null;default:''"`
	CreatorName      string `gorm:"size:128;not null;default:''"`
	AssigneeIMUserID string `gorm:"size:64;not null;default:'';index"`
	AssigneeName     string `gorm:"size:128;not null;default:''"`
	DoneAt           *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// GroupSharedEvent 群共享日程（协作）。
type GroupSharedEvent struct {
	ID              uint      `gorm:"primaryKey"`
	IMGroupID       string    `gorm:"size:64;not null;index:idx_group_events_group_status"`
	Title           string    `gorm:"size:200;not null"`
	Description     string    `gorm:"size:500;not null;default:''"`
	StartAt         time.Time `gorm:"not null;index"`
	EndAt           *time.Time
	Location        string `gorm:"size:200;not null;default:''"`
	Status          string `gorm:"size:16;not null;default:'active';index:idx_group_events_group_status"`
	CreatorIMUserID string `gorm:"size:64;not null;default:''"`
	CreatorName     string `gorm:"size:128;not null;default:''"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

const (
	GroupLedgerTypeExpense = "expense"
	GroupLedgerTypeIncome  = "income"
)

// GroupSharedLedger 群共享账本流水（协作）。
type GroupSharedLedger struct {
	ID              uint    `gorm:"primaryKey"`
	IMGroupID       string  `gorm:"size:64;not null;index:idx_group_ledgers_group_status"`
	Type            string  `gorm:"size:16;not null;default:'expense'"` // expense | income
	Amount          float64 `gorm:"not null;default:0"`
	Category        string  `gorm:"size:64;not null;default:''"`
	Note            string  `gorm:"size:500;not null;default:''"`
	RecordDate      string  `gorm:"size:16;not null;default:'';index"` // yyyy-MM-dd
	Status          string  `gorm:"size:16;not null;default:'active';index:idx_group_ledgers_group_status"`
	CreatorIMUserID string  `gorm:"size:64;not null;default:''"`
	CreatorName     string  `gorm:"size:128;not null;default:''"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
