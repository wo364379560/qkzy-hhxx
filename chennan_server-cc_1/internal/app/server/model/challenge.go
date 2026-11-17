package model

import "time"

// Challenge 表示围绕某个习惯发起的监督挑战。
type Challenge struct {
	ID                uint      `gorm:"primaryKey"`
	OwnerID           uint      `gorm:"index;not null"`
	CreatorUserID     uint      `gorm:"index;not null"`
	Name              string    `gorm:"size:128;not null"`
	Description       string    `gorm:"size:255"`
	LinkedUserHabitID uint      `gorm:"index;not null"`
	StartDate         time.Time `gorm:"type:date;not null"`
	EndDate           time.Time `gorm:"type:date;not null"`
	Status            string    `gorm:"type:enum('active','completed','archived');default:'active'"`
	IMGroupID         string    `gorm:"size:64;index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// ChallengeMember 表示挑战参与人。
type ChallengeMember struct {
	ID          uint      `gorm:"primaryKey"`
	ChallengeID uint      `gorm:"uniqueIndex:idx_challenge_member;not null"`
	UserID      uint      `gorm:"uniqueIndex:idx_challenge_member;not null"`
	Role        string    `gorm:"size:16;default:'member'"` // owner / member
	JoinStatus  string    `gorm:"size:16;default:'active'"` // active / left
	JoinedAt    time.Time `gorm:"not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
