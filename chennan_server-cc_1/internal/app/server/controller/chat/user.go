package controller_chat

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/pkg/ipgeo"
	"github.com/ixayda/ix-Go/pkg/logger"
	"gorm.io/gorm"
)

var ErrUserLoginIPForbidden = errors.New("forbidden to query target user login ip")

type UserBindOwnerErr string

func (e UserBindOwnerErr) Error() string { return string(e) }

// SearchUserResult 搜索结果条目。
type SearchUserResult struct {
	OwnerRevision    uint64 `json:"owner_revision"`
	OwnerState       string `json:"owner_state"`
	IsAssigned       bool   `json:"is_assigned"`
	IMUserID         string `json:"im_user_id"`
	WananID          string `json:"wanan_id,omitempty"`
	Phone            string `json:"phone,omitempty"`
	Nickname         string `json:"nickname"`
	UserTag          string `json:"user_tag,omitempty"`
	AvatarURL        string `json:"avatar_url"`
	OwnerID          uint   `json:"owner_id"`
	OwnerName        string `json:"owner_name,omitempty"`
	CanAddFriend     bool   `json:"can_add_friend"`
	AddBlockedReason string `json:"add_blocked_reason,omitempty"`
}

func toSearchUserResult(caller *model.User, u model.User) SearchUserResult {
	reason := FriendAddBlockedReason(caller, &u)
	summary := CurrentOwnerSummary(&u)
	return SearchUserResult{
		IMUserID:      u.IMUserID,
		WananID:       optionalString(u.WananID),
		Phone:         strings.TrimSpace(u.Phone),
		Nickname:      u.Nickname,
		UserTag:       strings.TrimSpace(u.UserTag),
		AvatarURL:     u.AvatarURL,
		OwnerID:       summary.OwnerID,
		OwnerRevision: u.OwnerRevision, OwnerState: summary.OwnerState, IsAssigned: summary.IsAssigned,
		OwnerName:        ownerNameForUser(u.OwnerID),
		CanAddFriend:     reason == "",
		AddBlockedReason: reason,
	}
}

func ownerNameForUser(ownerID uint) string {
	if ownerID == 0 {
		return ""
	}
	owner, err := repository.NewAdminUserRepository().FindByID(ownerID)
	if err != nil || owner == nil {
		return ""
	}
	return strings.TrimSpace(owner.DisplayName)
}

// UserTagResult 批量返回用户标签。
type UserTagResult struct {
	IMUserID string `json:"im_user_id"`
	UserTag  string `json:"user_tag,omitempty"`
}

// UserLastOnlineResult 批量返回用户最后在线时间。
type UserLastOnlineResult struct {
	IMUserID      string `json:"im_user_id"`
	LastOnlineAt  string `json:"last_online_at,omitempty"`
	LastOfflineAt string `json:"last_offline_at,omitempty"`
}

// UserLoginIPResult 返回单聊对象最近有效登录 IP 与 IP 归属地。
type UserLoginIPResult struct {
	IMUserID             string `json:"im_user_id"`
	CurrentLoginIP       string `json:"current_login_ip"`
	CurrentLoginLocation string `json:"current_login_location"`
}

func toVisibleSearchResults(caller *model.User, users []model.User) []SearchUserResult {
	results := make([]SearchUserResult, 0, len(users))
	for _, u := range users {
		if !CanSearchUser(caller, &u) {
			continue
		}
		results = append(results, toSearchUserResult(caller, u))
	}
	return results
}

// SearchUsers 按归属与全局搜索规则搜索用户。
func SearchUsers(ctx context.Context, imUserID, keyword string) ([]SearchUserResult, error) {
	repo := repository.NewUserRepository()

	caller, err := repo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	users, err := repo.SearchByOwner(keyword, imUserID)
	if err != nil {
		return nil, err
	}
	if scope := CurrentOwnerSummary(caller); scope.OwnerState == "unavailable" {
		return nil, UserBindOwnerErr(OwnerReasonMessage(scope.ReasonCode))
	}
	return toVisibleSearchResults(caller, users), nil
}

// SearchUsersExact 按手机号或弥络蓝序号精确搜索。
// 同归属可搜；超管/未归属只能搜到已开全局搜索的运营账号；助理/管理员不能跨空间互搜。
func SearchUsersExact(ctx context.Context, imUserID, keyword string) ([]SearchUserResult, error) {
	repo := repository.NewUserRepository()

	caller, err := repo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	users, err := repo.SearchByOwnerExact(strings.TrimSpace(keyword), imUserID)
	if err != nil {
		return nil, err
	}
	if scope := CurrentOwnerSummary(caller); scope.OwnerState == "unavailable" {
		return nil, UserBindOwnerErr(OwnerReasonMessage(scope.ReasonCode))
	}
	return toVisibleSearchResults(caller, users), nil
}

// GetUserBriefs 按 IM UserID 批量返回资料（含注册手机号、弥络蓝序号），供好友资料页展示。
func GetUserBriefs(ctx context.Context, imUserID string, targetIMUserIDs []string) ([]SearchUserResult, error) {
	repo := repository.NewUserRepository()

	if _, err := repo.FindByIMUserID(imUserID); err != nil {
		return nil, err
	}

	normalizedIDs := normalizeIMUserIDs(targetIMUserIDs)
	if len(normalizedIDs) == 0 {
		return []SearchUserResult{}, nil
	}
	if len(normalizedIDs) > 50 {
		normalizedIDs = normalizedIDs[:50]
	}

	users, err := repo.FindByIMUserIDs(normalizedIDs)
	if err != nil {
		return nil, err
	}

	userMap := make(map[string]model.User, len(users))
	for _, user := range users {
		if user.Status != "active" {
			continue
		}
		userMap[user.IMUserID] = user
	}

	results := make([]SearchUserResult, 0, len(normalizedIDs))
	for _, targetID := range normalizedIDs {
		user, ok := userMap[targetID]
		if !ok {
			continue
		}
		summary := CurrentOwnerSummary(&user)
		results = append(results, SearchUserResult{
			IMUserID:      user.IMUserID,
			WananID:       optionalString(user.WananID),
			Phone:         strings.TrimSpace(user.Phone),
			Nickname:      user.Nickname,
			UserTag:       strings.TrimSpace(user.UserTag),
			AvatarURL:     user.AvatarURL,
			OwnerID:       summary.OwnerID,
			OwnerRevision: user.OwnerRevision, OwnerState: summary.OwnerState, IsAssigned: summary.IsAssigned,
			OwnerName: ownerNameForUser(user.OwnerID),
		})
	}
	return results, nil
}

type FriendOwnerResult struct {
	User       *SearchUserResult      `json:"user"`
	Assignment *model.OwnerAssignment `json:"assignment"`
}

func BindFriendOwnerFromMe(ctx context.Context, imUserID, targetIMUserID string, requests ...string) (*FriendOwnerResult, error) {
	imUserID = strings.TrimSpace(imUserID)
	targetIMUserID = strings.TrimSpace(targetIMUserID)
	if imUserID == "" || targetIMUserID == "" || imUserID == targetIMUserID {
		return nil, UserBindOwnerErr("参数错误")
	}
	request := ""
	if len(requests) > 0 {
		request = requests[0]
	}
	assignment, err := SyncFriendOwnership(ctx, imUserID, targetIMUserID, request)
	if err != nil {
		return nil, err
	}
	repo := repository.NewUserRepository()
	caller, err := repo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}
	target, err := repo.FindByIMUserID(targetIMUserID)
	if err != nil {
		return nil, err
	}
	user := toSearchUserResult(caller, *target)
	return &FriendOwnerResult{User: &user, Assignment: assignment}, nil
}

func areTencentFriends(a, b string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := tencentim.NewClient().CheckBilateral(ctx, a, []string{b})
	return result[b], err
}

// GetUserTags 批量查询用户标签，用于聊天与群场景展示。
func GetUserTags(ctx context.Context, imUserID string, targetIMUserIDs []string) ([]UserTagResult, error) {
	repo := repository.NewUserRepository()

	if _, err := repo.FindByIMUserID(imUserID); err != nil {
		return nil, err
	}

	normalizedIDs := normalizeIMUserIDs(targetIMUserIDs)
	if len(normalizedIDs) == 0 {
		return []UserTagResult{}, nil
	}

	users, err := repo.FindByIMUserIDs(normalizedIDs)
	if err != nil {
		return nil, err
	}

	userMap := make(map[string]SearchUserResult, len(users))
	for _, user := range users {
		userMap[user.IMUserID] = SearchUserResult{
			IMUserID:  user.IMUserID,
			WananID:   optionalString(user.WananID),
			Nickname:  user.Nickname,
			UserTag:   strings.TrimSpace(user.UserTag),
			AvatarURL: user.AvatarURL,
		}
	}

	results := make([]UserTagResult, 0, len(normalizedIDs))
	for _, targetID := range normalizedIDs {
		user, ok := userMap[targetID]
		if !ok {
			continue
		}
		results = append(results, UserTagResult{
			IMUserID: user.IMUserID,
			UserTag:  user.UserTag,
		})
	}
	return results, nil
}

// GetUserLastOnline 在调用者同归属下批量查询用户最后在线时间。
func GetUserLastOnline(ctx context.Context, imUserID string, targetIMUserIDs []string) ([]UserLastOnlineResult, error) {
	repo := repository.NewUserRepository()

	caller, err := repo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	normalizedIDs := normalizeIMUserIDs(targetIMUserIDs)
	if len(normalizedIDs) == 0 {
		return []UserLastOnlineResult{}, nil
	}

	users, err := repo.FindByIMUserIDs(normalizedIDs)
	if err != nil {
		return nil, err
	}

	// SDK presence polling on the Web does not update our database. Reconcile
	// previously-online peers here so a missing/delayed offline callback cannot
	// make the label count from login. Already-offline users need no IM query.
	var checkIDs []string
	for _, user := range users {
		if user.OwnerID == caller.OwnerID && user.Status == "active" && user.IMUserID != caller.IMUserID && user.LastPresenceStatus != "offline" {
			checkIDs = append(checkIDs, user.IMUserID)
		}
	}
	if len(checkIDs) > 0 && tencentim.NewClient().IsConfigured() {
		checkedAt := time.Now()
		states, statusErr := tencentim.BatchQueryUserStatus(checkIDs)
		if statusErr == nil {
			var onlineIDs, offlineIDs []string
			for _, state := range states {
				id, _ := state["im_user_id"].(string)
				if state["status"] == "offline" {
					offlineIDs = append(offlineIDs, id)
				}
				if state["status"] == "online" {
					onlineIDs = append(onlineIDs, id)
				}
			}
			if statusErr = repo.BatchRecordPresenceStatusByIMUserIDs(onlineIDs, offlineIDs, checkedAt); statusErr == nil {
				users, err = repo.FindByIMUserIDs(normalizedIDs)
				if err != nil {
					return nil, err
				}
			}
		}
		if statusErr != nil {
			logger.Warnf("reconcile last offline presence failed: %v", statusErr)
		}
	}

	userMap := make(map[string]UserLastOnlineResult, len(users))
	for _, user := range users {
		if user.OwnerID != caller.OwnerID || user.Status != "active" || user.IMUserID == caller.IMUserID {
			continue
		}

		result := userLastOnlineResult(user)
		userMap[user.IMUserID] = result
	}

	results := make([]UserLastOnlineResult, 0, len(normalizedIDs))
	for _, targetID := range normalizedIDs {
		user, ok := userMap[targetID]
		if !ok {
			continue
		}
		results = append(results, user)
	}
	return results, nil
}

func userLastOnlineResult(user model.User) UserLastOnlineResult {
	result := UserLastOnlineResult{IMUserID: user.IMUserID}
	lastSeen := user.LastOnlineAt
	if user.LastOfflineAt != nil {
		result.LastOfflineAt = user.LastOfflineAt.UTC().Format(time.RFC3339)
		if lastSeen == nil || user.LastOfflineAt.After(*lastSeen) {
			lastSeen = user.LastOfflineAt
		}
	}
	// Preserve the legacy field for older clients, using the newer offline time.
	if lastSeen != nil {
		result.LastOnlineAt = lastSeen.UTC().Format(time.RFC3339)
	}
	return result
}

// GetUserLoginIP 查询同归属单聊对象最近有效登录 IP 与归属地。
func GetUserLoginIP(ctx context.Context, imUserID, targetIMUserID string) (*UserLoginIPResult, error) {
	imUserID = strings.TrimSpace(imUserID)
	targetIMUserID = strings.TrimSpace(targetIMUserID)
	if imUserID == "" || targetIMUserID == "" || imUserID == targetIMUserID {
		return nil, ErrUserLoginIPForbidden
	}

	repo := repository.NewUserRepository()
	caller, err := repo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}
	target, err := repo.FindByIMUserID(targetIMUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserLoginIPForbidden
		}
		return nil, err
	}
	if target.OwnerID != caller.OwnerID || target.Status != "active" {
		return nil, ErrUserLoginIPForbidden
	}

	currentLoginIP := ""
	session, err := repository.NewUserRefreshSessionRepository().FindLatestActiveByUserID(target.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if session != nil {
		currentLoginIP = strings.TrimSpace(session.IP)
	}

	currentLoginLocation := ""
	if currentLoginIP != "" {
		if location, err := ipgeo.Lookup(ctx, currentLoginIP); err == nil {
			currentLoginLocation = location.Text()
		} else {
			logger.Warnf("chat user login ip location lookup failed: im_user_id=%s ip=%s err=%v", target.IMUserID, currentLoginIP, err)
		}
	}

	return &UserLoginIPResult{
		IMUserID:             target.IMUserID,
		CurrentLoginIP:       currentLoginIP,
		CurrentLoginLocation: currentLoginLocation,
	}, nil
}

func normalizeIMUserIDs(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	results := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		results = append(results, normalized)
	}
	return results
}
