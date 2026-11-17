package repository

import (
	"errors"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CommunityRepository 处理社区帖子、点赞和评论。
type CommunityRepository struct {
	db *gorm.DB
}

// NewCommunityRepository 创建 CommunityRepository。
func NewCommunityRepository() *CommunityRepository {
	return &CommunityRepository{db: datasource.Datasource.Mysql.GetClient()}
}

// ListPosts 查询当前用户可见的社区帖子列表。
func (r *CommunityRepository) ListPosts(feed string, viewerUserID uint, limit int) ([]model.CommunityPost, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	q := r.db.Where("status = ? AND user_id = ?", "published", viewerUserID)
	switch feed {
	case "recommended":
		q = q.Order("like_count DESC, comment_count DESC, id DESC")
	default:
		q = q.Order("created_at DESC, id DESC")
	}

	var posts []model.CommunityPost
	err := q.Limit(limit).Find(&posts).Error
	return posts, err
}

// CreatePost 创建社区帖子。
func (r *CommunityRepository) CreatePost(post *model.CommunityPost) error {
	return r.db.Create(post).Error
}

// ListPostsPagedAdmin 按管理范围分页查询社区帖子。
func (r *CommunityRepository) ListPostsPagedAdmin(page, pageSize int, scopeOwnerIDs []uint, ownerID uint, keyword, postType string, userID uint) ([]model.CommunityPost, int64, error) {
	q := r.db.Model(&model.CommunityPost{}).
		Joins("JOIN users ON users.id = community_posts.user_id")

	if ownerID > 0 {
		q = q.Where("users.owner_id = ?", ownerID)
	} else if len(scopeOwnerIDs) > 0 {
		q = q.Where("users.owner_id IN ?", scopeOwnerIDs)
	}
	if userID > 0 {
		q = q.Where("community_posts.user_id = ?", userID)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("community_posts.title LIKE ? OR community_posts.content LIKE ? OR users.nickname LIKE ? OR users.im_user_id LIKE ? OR users.wanan_id LIKE ?", like, like, like, like, like)
	}
	if postType != "" {
		q = q.Where("community_posts.post_type = ?", postType)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var posts []model.CommunityPost
	offset := (page - 1) * pageSize
	err := q.Select("community_posts.*").
		Order("community_posts.id DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&posts).Error
	return posts, total, err
}

// DeletePost 软删除帖子。
func (r *CommunityRepository) DeletePost(postID uint) error {
	return r.db.Model(&model.CommunityPost{}).
		Where("id = ?", postID).
		Updates(map[string]any{
			"status": "deleted",
		}).Error
}

// FindPostByID 查询当前用户可见的单个帖子。
func (r *CommunityRepository) FindPostByID(postID, viewerUserID uint) (*model.CommunityPost, error) {
	var post model.CommunityPost
	if err := r.db.
		Where("id = ? AND status = 'published' AND user_id = ?", postID, viewerUserID).
		First(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

// FindPostByIDAnyStatus 查询单个帖子（后台使用，不限制状态）。
func (r *CommunityRepository) FindPostByIDAnyStatus(postID uint) (*model.CommunityPost, error) {
	var post model.CommunityPost
	if err := r.db.Where("id = ?", postID).First(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

// ListCommentsByPost 查询帖子的评论列表。
func (r *CommunityRepository) ListCommentsByPost(postID uint) ([]model.CommunityPostComment, error) {
	var comments []model.CommunityPostComment
	err := r.db.
		Where("post_id = ? AND status = 'active'", postID).
		Order("created_at ASC, id ASC").
		Find(&comments).Error
	return comments, err
}

// ListCommentsByPostAdmin 查询帖子的全部评论（后台管理）。
func (r *CommunityRepository) ListCommentsByPostAdmin(postID uint) ([]model.CommunityPostComment, error) {
	var comments []model.CommunityPostComment
	err := r.db.
		Where("post_id = ?", postID).
		Order("created_at ASC, id ASC").
		Find(&comments).Error
	return comments, err
}

// FindCommentByID 查询单个评论。
func (r *CommunityRepository) FindCommentByID(commentID uint) (*model.CommunityPostComment, error) {
	var comment model.CommunityPostComment
	if err := r.db.Where("id = ?", commentID).First(&comment).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

// CreateComment 创建评论并回写帖子评论数。
func (r *CommunityRepository) CreateComment(comment *model.CommunityPostComment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(comment).Error; err != nil {
			return err
		}
		return tx.Model(&model.CommunityPost{}).
			Where("id = ?", comment.PostID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).Error
	})
}

// UpdatePost 更新帖子内容。
func (r *CommunityRepository) UpdatePost(postID uint, fields map[string]any) error {
	return r.db.Model(&model.CommunityPost{}).Where("id = ?", postID).Updates(fields).Error
}

// DeleteCommentPhysical 物理删除评论并回写帖子评论数。
func (r *CommunityRepository) DeleteCommentPhysical(commentID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var comment model.CommunityPostComment
		if err := tx.Where("id = ?", commentID).First(&comment).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", commentID).Delete(&model.CommunityPostComment{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.CommunityPost{}).
			Where("id = ?", comment.PostID).
			UpdateColumn("comment_count", gorm.Expr("GREATEST(comment_count - 1, 0)")).Error; err != nil {
			return err
		}
		return tx.Where("target_type = ? AND target_id = ?", "community_comment", commentID).
			Delete(&model.Report{}).Error
	})
}

// DeletePostPhysical 物理删除帖子及其评论、点赞、相关举报。
func (r *CommunityRepository) DeletePostPhysical(postID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("post_id = ?", postID).Delete(&model.CommunityPostLike{}).Error; err != nil {
			return err
		}
		var commentIDs []uint
		if err := tx.Model(&model.CommunityPostComment{}).Where("post_id = ?", postID).Pluck("id", &commentIDs).Error; err != nil {
			return err
		}
		if len(commentIDs) > 0 {
			if err := tx.Where("target_type = ? AND target_id IN ?", "community_comment", commentIDs).
				Delete(&model.Report{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("post_id = ?", postID).Delete(&model.CommunityPostComment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("target_type = ? AND target_id = ?", "community_post", postID).
			Delete(&model.Report{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", postID).Delete(&model.CommunityPost{}).Error
	})
}

// ListLikedPostIDsByUser 查询用户已点赞的帖子 ID。
func (r *CommunityRepository) ListLikedPostIDsByUser(userID uint, postIDs []uint) (map[uint]struct{}, error) {
	if len(postIDs) == 0 {
		return map[uint]struct{}{}, nil
	}

	var likes []model.CommunityPostLike
	if err := r.db.
		Where("user_id = ? AND post_id IN ?", userID, postIDs).
		Find(&likes).Error; err != nil {
		return nil, err
	}

	result := make(map[uint]struct{}, len(likes))
	for _, like := range likes {
		result[like.PostID] = struct{}{}
	}
	return result, nil
}

// ToggleLike 切换当前用户自己帖子上的点赞状态并返回最新结果。
func (r *CommunityRepository) ToggleLike(postID, userID uint) (bool, int, error) {
	var liked bool
	var likeCount int

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var post model.CommunityPost
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status = 'published' AND user_id = ?", postID, userID).
			First(&post).Error; err != nil {
			return err
		}

		var existing model.CommunityPostLike
		err := tx.Where("post_id = ? AND user_id = ?", postID, userID).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := tx.Create(&model.CommunityPostLike{PostID: postID, UserID: userID}).Error; err != nil {
				return err
			}
			if err := tx.Model(&post).
				UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil {
				return err
			}
			liked = true
		case err != nil:
			return err
		default:
			if err := tx.Delete(&existing).Error; err != nil {
				return err
			}
			if err := tx.Model(&post).
				UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error; err != nil {
				return err
			}
			liked = false
		}

		if err := tx.Model(&model.CommunityPost{}).
			Select("like_count").
			Where("id = ?", postID).
			Scan(&likeCount).Error; err != nil {
			return err
		}
		return nil
	})

	return liked, likeCount, err
}
