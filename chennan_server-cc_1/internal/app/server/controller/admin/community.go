package controller_admin

import (
	"errors"
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
)

// AdminCommunityAuthorResult 表示后台省钱社区作者信息。
type AdminCommunityAuthorResult struct {
	UserID                   uint   `json:"userID"`
	IMUserID                 string `json:"imUserID"`
	WananID                  string `json:"wananID"`
	Nickname                 string `json:"nickname"`
	AvatarURL                string `json:"avatarURL"`
	OwnerID                  uint   `json:"ownerID"`
	OwnerName                string `json:"ownerName"`
	CommunityPostDisabled    bool   `json:"communityPostDisabled"`
	CommunityCommentDisabled bool   `json:"communityCommentDisabled"`

	UserNo              string `json:"userNo"`
	SavingPostDisabled  bool   `json:"savingPostDisabled"`
	SavingCommentDisabled bool `json:"savingCommentDisabled"`
}

// AdminCommunityPostResult 表示后台帖子列表项。
type AdminCommunityPostResult struct {
	ID           uint                       `json:"id"`
	PostType     string                     `json:"postType"`
	Title        string                     `json:"title"`
	Content      string                     `json:"content"`
	LikeCount    int                        `json:"likeCount"`
	CommentCount int                        `json:"commentCount"`
	CreatedAt    string                     `json:"createdAt"`
	UpdatedAt    string                     `json:"updatedAt"`
	Author       AdminCommunityAuthorResult `json:"author"`

	ContentType string `json:"contentType"`
}

// AdminCommunityCommentResult 表示后台评论列表项。
type AdminCommunityCommentResult struct {
	ID        uint                       `json:"id"`
	PostID    uint                       `json:"postID"`
	Content   string                     `json:"content"`
	Status    string                     `json:"status"`
	CreatedAt string                     `json:"createdAt"`
	UpdatedAt string                     `json:"updatedAt"`
	Author    AdminCommunityAuthorResult `json:"author"`
}

type ListAdminCommunityPostsReq struct {
	Page          int
	PageSize      int
	ScopeOwnerIDs []uint
	OwnerID       uint
	Keyword       string
	PostType      string
	UserID        uint
}

type UpdateAdminCommunityPostReq struct {
	PostType string
	Title    string
	Content  string
}

type UpdateCommunityUserRestrictionReq struct {
	CommunityPostDisabled    *bool
	CommunityCommentDisabled *bool
}

func ListAdminCommunityPosts(req ListAdminCommunityPostsReq) (PageResult[AdminCommunityPostResult], error) {
	posts, total, err := repository.NewCommunityRepository().ListPostsPagedAdmin(
		req.Page,
		req.PageSize,
		req.ScopeOwnerIDs,
		req.OwnerID,
		strings.TrimSpace(req.Keyword),
		strings.TrimSpace(req.PostType),
		req.UserID,
	)
	if err != nil {
		return PageResult[AdminCommunityPostResult]{}, err
	}

	userMap, ownerMap, err := loadCommunityUserMaps(posts)
	if err != nil {
		return PageResult[AdminCommunityPostResult]{}, err
	}

	list := make([]AdminCommunityPostResult, 0, len(posts))
	for i := range posts {
		author := userMap[posts[i].UserID]
		list = append(list, AdminCommunityPostResult{
			ID:           posts[i].ID,
			PostType:     posts[i].PostType,
			Title:        posts[i].Title,
			Content:      posts[i].Content,
			LikeCount:    posts[i].LikeCount,
			CommentCount: posts[i].CommentCount,
			CreatedAt:    posts[i].CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:    posts[i].UpdatedAt.Format("2006-01-02 15:04:05"),
			Author:       buildAdminCommunityAuthor(author, ownerMap),
		})
	}

	return PageResult[AdminCommunityPostResult]{List: list, Total: total}, nil
}

func GetAdminCommunityComments(actorID uint, isAdmin bool, postID uint) ([]AdminCommunityCommentResult, error) {
	post, err := findManagedCommunityPost(actorID, isAdmin, postID)
	if err != nil {
		return nil, err
	}

	comments, err := repository.NewCommunityRepository().ListCommentsByPostAdmin(post.ID)
	if err != nil {
		return nil, err
	}

	users := make([]model.User, 0, len(comments))
	for i := range comments {
		user, userErr := repository.NewUserRepository().FindByID(comments[i].UserID)
		if userErr == nil {
			users = append(users, *user)
		}
	}
	ownerMap, err := loadCommunityOwnerMapFromUsers(users)
	if err != nil {
		return nil, err
	}

	userMap := make(map[uint]*model.User, len(users))
	for i := range users {
		userMap[users[i].ID] = &users[i]
	}

	result := make([]AdminCommunityCommentResult, 0, len(comments))
	for i := range comments {
		author := userMap[comments[i].UserID]
		result = append(result, AdminCommunityCommentResult{
			ID:        comments[i].ID,
			PostID:    comments[i].PostID,
			Content:   comments[i].Content,
			Status:    comments[i].Status,
			CreatedAt: comments[i].CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: comments[i].UpdatedAt.Format("2006-01-02 15:04:05"),
			Author:    buildAdminCommunityAuthor(author, ownerMap),
		})
	}

	return result, nil
}

func UpdateAdminCommunityPost(actorID uint, isAdmin bool, postID uint, req UpdateAdminCommunityPostReq) error {
	post, err := findManagedCommunityPost(actorID, isAdmin, postID)
	if err != nil {
		return err
	}

	req.PostType = strings.TrimSpace(req.PostType)
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)

	if req.PostType == "" {
		req.PostType = post.PostType
	}
	if !isValidAdminCommunityPostType(req.PostType) {
		return errors.New("不支持的内容类型")
	}
	if req.Content == "" {
		return errors.New("内容不能为空")
	}
	if len([]rune(req.Title)) > 64 {
		return errors.New("标题不能超过 64 个字符")
	}
	if len([]rune(req.Content)) > 2000 {
		return errors.New("内容不能超过 2000 个字符")
	}

	return repository.NewCommunityRepository().UpdatePost(post.ID, map[string]any{
		"post_type": req.PostType,
		"title":     req.Title,
		"content":   req.Content,
	})
}

func DeleteAdminCommunityPost(actorID uint, isAdmin bool, postID uint) error {
	post, err := findManagedCommunityPost(actorID, isAdmin, postID)
	if err != nil {
		return err
	}
	return repository.NewCommunityRepository().DeletePostPhysical(post.ID)
}

func DeleteAdminCommunityComment(actorID uint, isAdmin bool, commentID uint) error {
	comment, err := findManagedCommunityComment(actorID, isAdmin, commentID)
	if err != nil {
		return err
	}
	return repository.NewCommunityRepository().DeleteCommentPhysical(comment.ID)
}

func UpdateCommunityUserRestriction(actorID uint, isAdmin bool, userID uint, req UpdateCommunityUserRestrictionReq) error {
	user, err := findManagedUser(actorID, isAdmin, userID)
	if err != nil {
		return err
	}

	updates := map[string]any{}
	if req.CommunityPostDisabled != nil {
		updates["community_post_disabled"] = *req.CommunityPostDisabled
	}
	if req.CommunityCommentDisabled != nil {
		updates["community_comment_disabled"] = *req.CommunityCommentDisabled
	}
	if len(updates) == 0 {
		return errors.New("未提供可更新的限制项")
	}

	return repository.NewUserRepository().UpdateFields(user.ID, updates)
}

func buildAdminCommunityAuthor(user *model.User, ownerMap map[uint]*model.AdminUser) AdminCommunityAuthorResult {
	if user == nil {
		return AdminCommunityAuthorResult{}
	}
	ownerName := ""
	if owner, ok := ownerMap[user.OwnerID]; ok {
		ownerName = owner.Username
	}
	return AdminCommunityAuthorResult{
		UserID:                   user.ID,
		IMUserID:                 user.IMUserID,
		WananID:                  optionalString(user.WananID),
		Nickname:                 user.Nickname,
		AvatarURL:                user.AvatarURL,
		OwnerID:                  user.OwnerID,
		OwnerName:                ownerName,
		CommunityPostDisabled:    user.CommunityPostDisabled,
		CommunityCommentDisabled: user.CommunityCommentDisabled,
		UserNo:                   optionalString(user.WananID),
		SavingPostDisabled:       user.CommunityPostDisabled,
		SavingCommentDisabled:    user.CommunityCommentDisabled,
	}
}

func loadCommunityUserMaps(posts []model.CommunityPost) (map[uint]*model.User, map[uint]*model.AdminUser, error) {
	userRepo := repository.NewUserRepository()
	users := make([]model.User, 0, len(posts))
	for i := range posts {
		user, err := userRepo.FindByID(posts[i].UserID)
		if err == nil {
			users = append(users, *user)
		}
	}
	ownerMap, err := loadCommunityOwnerMapFromUsers(users)
	if err != nil {
		return nil, nil, err
	}
	userMap := make(map[uint]*model.User, len(users))
	for i := range users {
		userMap[users[i].ID] = &users[i]
	}
	return userMap, ownerMap, nil
}

func loadCommunityOwnerMapFromUsers(users []model.User) (map[uint]*model.AdminUser, error) {
	ownerIDs := make([]uint, 0)
	seen := make(map[uint]struct{})
	for i := range users {
		if users[i].OwnerID == 0 {
			continue
		}
		if _, ok := seen[users[i].OwnerID]; ok {
			continue
		}
		seen[users[i].OwnerID] = struct{}{}
		ownerIDs = append(ownerIDs, users[i].OwnerID)
	}
	owners, err := repository.NewAdminUserRepository().FindByIDs(ownerIDs)
	if err != nil {
		return nil, err
	}
	ownerMap := make(map[uint]*model.AdminUser, len(owners))
	for i := range owners {
		ownerMap[owners[i].ID] = &owners[i]
	}
	return ownerMap, nil
}

func isValidAdminCommunityPostType(postType string) bool {
	switch postType {
	case "checkin_share", "challenge_recruit", "experience_share":
		return true
	default:
		return false
	}
}
