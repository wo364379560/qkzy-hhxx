package controller_chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"gorm.io/gorm"
)

const (
	communityPostTypeCheckinShare     = "checkin_share"
	communityPostTypeChallengeRecruit = "challenge_recruit"
	communityPostTypeExperienceShare  = "experience_share"
)

// CommunityAuthorResult 表示帖子作者信息。
type CommunityAuthorResult struct {
	UserID    string `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

// CommunityPostResult 表示社区帖子摘要。
type CommunityPostResult struct {
	ID                   uint                  `json:"id"`
	PostType             string                `json:"post_type"`
	Title                string                `json:"title"`
	Content              string                `json:"content"`
	LinkedHabitID        uint                  `json:"linked_habit_id,omitempty"`
	LinkedHabitName      string                `json:"linked_habit_name,omitempty"`
	LinkedChallengeID    uint                  `json:"linked_challenge_id,omitempty"`
	LinkedChallengeTitle string                `json:"linked_challenge_title,omitempty"`
	LikeCount            int                   `json:"like_count"`
	CommentCount         int                   `json:"comment_count"`
	IsLiked              bool                  `json:"is_liked"`
	IsMine               bool                  `json:"is_mine"`
	CreatedAt            string                `json:"created_at"`
	Author               CommunityAuthorResult `json:"author"`

	ContentType            string                `json:"content_type"`
	LinkedLedgerID         uint                  `json:"linked_ledger_id,omitempty"`
	LinkedLedgerName       string                `json:"linked_ledger_name,omitempty"`
	LinkedSplitRecordID    uint                  `json:"linked_split_record_id,omitempty"`
	LinkedSplitRecordTitle string                `json:"linked_split_record_title,omitempty"`
}

// CommunityFeedResult 表示社区广场列表。
type CommunityFeedResult struct {
	Feed  string                `json:"feed"`
	Items []CommunityPostResult `json:"items"`

	ContentFeed string                `json:"content_feed"`
	Posts       []CommunityPostResult `json:"posts"`
}

// CommunityCommentResult 表示帖子评论。
type CommunityCommentResult struct {
	ID        uint                  `json:"id"`
	Content   string                `json:"content"`
	CreatedAt string                `json:"created_at"`
	Author    CommunityAuthorResult `json:"author"`
}

// CommunityPostDetailResult 表示帖子详情。
type CommunityPostDetailResult struct {
	Post     CommunityPostResult      `json:"post"`
	Comments []CommunityCommentResult `json:"comments"`

	Content CommunityPostResult      `json:"content"`
	Replies []CommunityCommentResult `json:"replies"`
}

// CreateCommunityPostReq 表示创建帖子请求。
type CreateCommunityPostReq struct {
	PostType          string
	Title             string
	Content           string
	LinkedHabitID     uint
	LinkedChallengeID uint
}

// CreateCommunityPostResult 表示创建帖子结果。
type CreateCommunityPostResult struct {
	ID uint `json:"id"`

	PostID uint `json:"post_id"`
}

// CreateCommunityCommentResult 表示评论结果。
type CreateCommunityCommentResult struct {
	Comment CommunityCommentResult `json:"comment"`

	Reply CommunityCommentResult `json:"reply"`
}

// ToggleCommunityPostLikeResult 表示点赞切换结果。
type ToggleCommunityPostLikeResult struct {
	IsLiked   bool `json:"is_liked"`
	LikeCount int  `json:"like_count"`

	Liked bool `json:"liked"`
}

// DeleteCommunityPost 删除自己的帖子。
func DeleteCommunityPost(_ context.Context, imUserID string, postID uint) error {
	userRepo := repository.NewUserRepository()
	caller, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return err
	}

	communityRepo := repository.NewCommunityRepository()
	post, err := communityRepo.FindPostByID(postID, caller.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("内容不存在")
		}
		return err
	}
	if post.UserID != caller.ID {
		return errors.New("只能删除自己的内容")
	}

	return communityRepo.DeletePost(postID)
}

// ListCommunityPosts 查询社区帖子流。
func ListCommunityPosts(_ context.Context, imUserID, feed string) (*CommunityFeedResult, error) {
	feed = normalizeCommunityFeed(feed)

	userRepo := repository.NewUserRepository()
	viewer, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	communityRepo := repository.NewCommunityRepository()
	posts, err := communityRepo.ListPosts(feed, viewer.ID, 20)
	if err != nil {
		return nil, err
	}

	items, err := buildCommunityPostResults(viewer.ID, posts)
	if err != nil {
		return nil, err
	}

	return &CommunityFeedResult{
		Feed:        feed,
		Items:       items,
		ContentFeed: feed,
		Posts:       items,
	}, nil
}

// CreateCommunityPost 创建社区帖子。
func CreateCommunityPost(_ context.Context, imUserID string, req CreateCommunityPostReq) (*CreateCommunityPostResult, error) {
	req.PostType = strings.TrimSpace(req.PostType)
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)

	if !isValidCommunityPostType(req.PostType) {
		return nil, errors.New("不支持的内容类型")
	}
	if req.Content == "" {
		return nil, errors.New("内容不能为空")
	}
	if len([]rune(req.Title)) > 64 {
		return nil, errors.New("标题不能超过 64 个字符")
	}
	if len([]rune(req.Content)) > 2000 {
		return nil, errors.New("内容不能超过 2000 个字符")
	}

	userRepo := repository.NewUserRepository()
	caller, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}
	if caller.CommunityPostDisabled {
		return nil, errors.New("当前账号已被限制发布内容")
	}

	post := &model.CommunityPost{
		UserID:   caller.ID,
		PostType: req.PostType,
		Title:    req.Title,
		Content:  req.Content,
		Status:   "published",
	}

	habitRepo := repository.NewHabitRepository()
	challengeRepo := repository.NewChallengeRepository()

	if req.LinkedHabitID > 0 {
		habit, err := habitRepo.FindHabitByIDAndUser(req.LinkedHabitID, caller.ID)
		if err != nil {
			return nil, errors.New("关联账本不存在")
		}
		post.LinkedUserHabitID = &habit.ID
		if post.Title == "" && req.PostType == communityPostTypeCheckinShare {
			post.Title = fmt.Sprintf("今天完成了 %s", habit.Name)
		}
	}

	if req.LinkedChallengeID > 0 {
		challenge, err := challengeRepo.FindByID(req.LinkedChallengeID)
		if err != nil {
			return nil, errors.New("关联分账不存在")
		}
		if challenge.OwnerID != caller.OwnerID {
			return nil, errors.New("无权关联该分账")
		}
		post.LinkedChallengeID = &challenge.ID
		if post.Title == "" && req.PostType == communityPostTypeChallengeRecruit {
			post.Title = fmt.Sprintf("一起加入 %s", challenge.Name)
		}
	}

	if post.Title == "" {
		switch req.PostType {
		case communityPostTypeExperienceShare:
			post.Title = "我的省钱心得"
		case communityPostTypeCheckinShare:
			post.Title = "今日记账"
		case communityPostTypeChallengeRecruit:
			post.Title = "分账招募"
		}
	}

	communityRepo := repository.NewCommunityRepository()
	if err := communityRepo.CreatePost(post); err != nil {
		return nil, err
	}

	return &CreateCommunityPostResult{ID: post.ID, PostID: post.ID}, nil
}

// GetCommunityPostDetail 获取帖子详情。
func GetCommunityPostDetail(_ context.Context, imUserID string, postID uint) (*CommunityPostDetailResult, error) {
	userRepo := repository.NewUserRepository()
	viewer, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	communityRepo := repository.NewCommunityRepository()
	post, err := communityRepo.FindPostByID(postID, viewer.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("内容不存在")
		}
		return nil, err
	}

	postItems, err := buildCommunityPostResults(viewer.ID, []model.CommunityPost{*post})
	if err != nil {
		return nil, err
	}
	if len(postItems) == 0 {
		return nil, errors.New("内容不存在")
	}

	comments, err := communityRepo.ListCommentsByPost(postID)
	if err != nil {
		return nil, err
	}

	commentResults := make([]CommunityCommentResult, 0, len(comments))
	for _, comment := range comments {
		author, err := userRepo.FindByID(comment.UserID)
		if err != nil {
			continue
		}
		commentResults = append(commentResults, CommunityCommentResult{
			ID:        comment.ID,
			Content:   comment.Content,
			CreatedAt: comment.CreatedAt.Format(time.RFC3339),
			Author:    buildCommunityAuthor(author),
		})
	}

	return &CommunityPostDetailResult{
		Post:     postItems[0],
		Comments: commentResults,
		Content:  postItems[0],
		Replies:  commentResults,
	}, nil
}

// CreateCommunityComment 创建帖子评论。
func CreateCommunityComment(_ context.Context, imUserID string, postID uint, content string) (*CreateCommunityCommentResult, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("评论内容不能为空")
	}
	if len([]rune(content)) > 500 {
		return nil, errors.New("评论不能超过 500 个字符")
	}

	userRepo := repository.NewUserRepository()
	caller, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}
	if caller.CommunityCommentDisabled {
		return nil, errors.New("当前账号已被限制评论")
	}

	communityRepo := repository.NewCommunityRepository()
	if _, err := communityRepo.FindPostByID(postID, caller.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("内容不存在")
		}
		return nil, err
	}

	comment := &model.CommunityPostComment{
		PostID:  postID,
		UserID:  caller.ID,
		Content: content,
		Status:  "active",
	}
	if err := communityRepo.CreateComment(comment); err != nil {
		return nil, err
	}

	return &CreateCommunityCommentResult{
		Comment: CommunityCommentResult{
			ID:        comment.ID,
			Content:   comment.Content,
			CreatedAt: comment.CreatedAt.Format(time.RFC3339),
			Author:    buildCommunityAuthor(caller),
		},
		Reply: CommunityCommentResult{
			ID:        comment.ID,
			Content:   comment.Content,
			CreatedAt: comment.CreatedAt.Format(time.RFC3339),
			Author:    buildCommunityAuthor(caller),
		},
	}, nil
}

// ToggleCommunityPostLike 切换帖子点赞状态。
func ToggleCommunityPostLike(_ context.Context, imUserID string, postID uint) (*ToggleCommunityPostLikeResult, error) {
	userRepo := repository.NewUserRepository()
	caller, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	communityRepo := repository.NewCommunityRepository()
	liked, likeCount, err := communityRepo.ToggleLike(postID, caller.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("内容不存在")
		}
		return nil, err
	}

	return &ToggleCommunityPostLikeResult{
		IsLiked:   liked,
		LikeCount: likeCount,
		Liked:     liked,
	}, nil
}

func buildCommunityPostResults(viewerID uint, posts []model.CommunityPost) ([]CommunityPostResult, error) {
	if len(posts) == 0 {
		return []CommunityPostResult{}, nil
	}

	communityRepo := repository.NewCommunityRepository()
	userRepo := repository.NewUserRepository()
	habitRepo := repository.NewHabitRepository()
	challengeRepo := repository.NewChallengeRepository()

	postIDs := make([]uint, 0, len(posts))
	for _, post := range posts {
		postIDs = append(postIDs, post.ID)
	}

	likedMap, err := communityRepo.ListLikedPostIDsByUser(viewerID, postIDs)
	if err != nil {
		return nil, err
	}

	results := make([]CommunityPostResult, 0, len(posts))
	for _, post := range posts {
		author, err := userRepo.FindByID(post.UserID)
		if err != nil {
			continue
		}

		item := CommunityPostResult{
			ID:           post.ID,
			PostType:     post.PostType,
			Title:        post.Title,
			Content:      post.Content,
			LikeCount:    post.LikeCount,
			CommentCount: post.CommentCount,
			CreatedAt:    post.CreatedAt.Format(time.RFC3339),
			Author:       buildCommunityAuthor(author),
			IsMine:       post.UserID == viewerID,
		}
		_, item.IsLiked = likedMap[post.ID]

		if post.LinkedUserHabitID != nil {
			item.LinkedHabitID = *post.LinkedUserHabitID
			if habit, err := habitRepo.FindHabitByID(*post.LinkedUserHabitID); err == nil {
				item.LinkedHabitName = habit.Name
			}
		}

		if post.LinkedChallengeID != nil {
			item.LinkedChallengeID = *post.LinkedChallengeID
			if challenge, err := challengeRepo.FindByID(*post.LinkedChallengeID); err == nil {
				item.LinkedChallengeTitle = challenge.Name
			}
		}

		results = append(results, item)
	}

	return results, nil
}

func buildCommunityAuthor(user *model.User) CommunityAuthorResult {
	return CommunityAuthorResult{
		UserID:    user.IMUserID,
		Nickname:  user.Nickname,
		AvatarURL: user.AvatarURL,
	}
}

func isValidCommunityPostType(postType string) bool {
	switch postType {
	case communityPostTypeCheckinShare, communityPostTypeChallengeRecruit, communityPostTypeExperienceShare:
		return true
	default:
		return false
	}
}

func normalizeCommunityFeed(feed string) string {
	switch strings.TrimSpace(feed) {
	case "latest":
		return "latest"
	default:
		return "recommended"
	}
}
