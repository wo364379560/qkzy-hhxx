package model

import "time"

// CommunityPost 表示社区帖子。
type CommunityPost struct {
	ID                uint   `gorm:"primaryKey"`
	UserID            uint   `gorm:"index;not null"`
	PostType          string `gorm:"size:32;index;not null"` // checkin_share / challenge_recruit / experience_share
	Title             string `gorm:"size:128"`
	Content           string `gorm:"size:3000;not null"`
	LinkedUserHabitID *uint  `gorm:"index"`
	LinkedChallengeID *uint  `gorm:"index"`
	Status            string `gorm:"size:16;default:'published'"` // published / hidden / deleted
	LikeCount         int    `gorm:"default:0"`
	CommentCount      int    `gorm:"default:0"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// CommunityPostLike 表示帖子点赞关系。
type CommunityPostLike struct {
	ID     uint `gorm:"primaryKey"`
	PostID uint `gorm:"uniqueIndex:idx_post_user_like;not null"`
	UserID uint `gorm:"uniqueIndex:idx_post_user_like;not null"`

	CreatedAt time.Time
}

// CommunityPostComment 表示帖子评论。
type CommunityPostComment struct {
	ID      uint   `gorm:"primaryKey"`
	PostID  uint   `gorm:"index;not null"`
	UserID  uint   `gorm:"index;not null"`
	Content string `gorm:"size:1000;not null"`
	Status  string `gorm:"size:16;default:'active'"` // active / deleted

	CreatedAt time.Time
	UpdatedAt time.Time
}
