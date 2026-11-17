package model

import "time"

// Credentials remain in the server/COS signature; no signed URLs are persisted.
type ChatMultipartUpload struct {
	ID          string    `gorm:"size:64;primaryKey" json:"-"`
	FileID      string    `gorm:"size:64;uniqueIndex;not null" json:"file_id"`
	Owner       string    `gorm:"size:64;index;not null" json:"-"`
	Fingerprint string    `gorm:"size:64;not null" json:"content_hash"`
	UploadID    string    `gorm:"size:256;not null" json:"-"`
	PartSize    int64     `gorm:"not null" json:"part_size"`
	State       string    `gorm:"size:16;index;not null" json:"state"`
	LeaseUntil  int64     `gorm:"index" json:"-"`
	Attempts    int       `json:"-"`
	ExpiresAt   time.Time `gorm:"index" json:"expires_at"`
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}
