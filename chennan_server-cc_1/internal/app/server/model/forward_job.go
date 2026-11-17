package model

import "time"

// Forward jobs are durable; a browser disconnect never deletes queued work.
type ForwardJob struct {
	ID               string    `gorm:"size:64;primaryKey" json:"id"`
	Sender           string    `gorm:"size:64;index;not null" json:"-"`
	RequestID        string    `gorm:"size:64;not null" json:"request_id"`
	Fingerprint      string    `gorm:"size:64;not null" json:"-"`
	CreatedAt        time.Time `json:"created_at"`
	Protocol         int       `gorm:"not null;default:0" json:"-"`
	State            string    `gorm:"size:16;index;not null;default:''" json:"state"`
	TargetCount      int64     `gorm:"not null;default:0" json:"target_count"`
	MessageCount     int64     `gorm:"not null;default:0" json:"message_count"`
	UploadedTargets  int64     `gorm:"not null;default:0" json:"uploaded_targets"`
	UploadedMessages int64     `gorm:"not null;default:0" json:"uploaded_messages"`
	Materialized     int64     `gorm:"not null;default:0" json:"materialized"`
	FillAt           int64     `gorm:"index;not null;default:0" json:"-"`
}

// Inputs are persisted once; their Cartesian product is generated gradually.
// Chunk size is a transport/backpressure detail, never a task-volume quota.
type ForwardTarget struct {
	JobID          string `gorm:"size:64;primaryKey;uniqueIndex:idx_forward_target_identity,priority:1"`
	Ordinal        int64  `gorm:"primaryKey"`
	Identity       string `gorm:"size:64;not null;uniqueIndex:idx_forward_target_identity,priority:2"`
	ConversationID string `gorm:"size:128;not null"`
	Name           string `gorm:"size:128"`
}
type ForwardMessage struct {
	JobID       string `gorm:"size:64;primaryKey"`
	Ordinal     int64  `gorm:"primaryKey"`
	Fingerprint string `gorm:"size:64;not null"`
	Body        string `gorm:"type:text;not null"`
	Preview     string `gorm:"size:256"`
}
type ForwardRetry struct {
	JobID     string `gorm:"size:64;primaryKey"`
	RetryID   string `gorm:"size:64;primaryKey"`
	CreatedAt time.Time
	Cursor    uint64 `gorm:"not null;default:0"`
	UpperID   uint64 `gorm:"not null;default:0"`
	Done      bool   `gorm:"index;not null;default:false"`
}

type ForwardDelivery struct {
	ID             uint64 `gorm:"primaryKey" json:"-"`
	JobID          string `gorm:"size:64;index:idx_forward_order,priority:1;index:idx_forward_status,priority:1;not null" json:"-"`
	Sender         string `gorm:"size:64;index;not null" json:"-"`
	ConversationID string `gorm:"size:128;index:idx_forward_order,priority:2;not null" json:"conversation_id"`
	Name           string `gorm:"size:128" json:"name"`
	Ordinal        int    `gorm:"index:idx_forward_order,priority:3" json:"index"`
	Preview        string `gorm:"size:256" json:"preview"`
	Message        string `gorm:"type:text;not null" json:"-"`
	Status         string `gorm:"size:16;index:idx_forward_due,priority:1;index:idx_forward_status,priority:2;not null" json:"status"`
	NextAt         int64  `gorm:"index:idx_forward_due,priority:2" json:"-"`
	LeaseUntil     int64  `gorm:"index" json:"-"`
	Claim          string `gorm:"size:64" json:"-"`
	Attempts       int    `json:"attempts"`
	Random         uint32 `json:"-"`
	Sequence       uint32 `json:"-"`
	Error          string `gorm:"size:256" json:"error,omitempty"`
	// Retain the latest definite diagnostic even after retry/success. These are
	// private operator fields; public progress uses the sanitized Error above.
	LastErrorCommand     string    `gorm:"size:64;not null;default:''" json:"-"`
	LastErrorCode        int       `gorm:"not null;default:0" json:"-"`
	LastErrorInfo        string    `gorm:"size:512;not null;default:''" json:"-"`
	LastTencentRequestID string    `gorm:"size:128;not null;default:''" json:"-"`
	Result               string    `gorm:"size:128" json:"message_id,omitempty"`
	CreatedAt            time.Time `json:"-"`
	UpdatedAt            time.Time `json:"-"`
}

// Gates are shared by all workers/processes using this database.
type ForwardGate struct {
	ID     string `gorm:"size:160;primaryKey"`
	NextAt int64
}
