package controller_chat

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"regexp"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
)

var wananIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{5,19}$`)

// ProfileErr is a user-facing profile update error.
type ProfileErr string

func (e ProfileErr) Error() string { return string(e) }

type ProfileSyncError struct {
	AvatarURL string
	Cause     error
}

func (e *ProfileSyncError) Error() string {
	return "业务资料已保存，但聊天资料同步未完成，请重试资料同步（无需重新上传）"
}
func (e *ProfileSyncError) Unwrap() error { return e.Cause }

func syncSavedProfile(imUserID, avatarURL string, items []tencentim.ProfileItemDetail, configured bool, send func(string, []tencentim.ProfileItemDetail) error) error {
	if len(items) == 0 {
		return nil
	}
	if !configured {
		return &ProfileSyncError{AvatarURL: avatarURL, Cause: errors.New("IM未配置")}
	}
	if err := send(imUserID, items); err != nil {
		return &ProfileSyncError{AvatarURL: avatarURL, Cause: err}
	}
	return nil
}

// ProfileResult 返回给 App 的用户资料。
type ProfileResult struct {
	OwnerRevision  uint64 `json:"owner_revision"`
	OwnerState     string `json:"owner_state"`
	OwnerRole      string `json:"owner_role"`
	OwnerReason    string `json:"owner_reason_code,omitempty"`
	IMUserID       string `json:"im_user_id"`
	Phone          string `json:"phone"`
	WananID        string `json:"wanan_id"`
	Nickname       string `json:"nickname"`
	UserTag        string `json:"user_tag"`
	AvatarURL      string `json:"avatar_url"`
	SelfSignature  string `json:"self_signature"`
	Gender         int8   `json:"gender"`
	Birthday       int    `json:"birthday"` // YYYYMMDD，0 表示未设置
	AllowType      int8   `json:"allow_type"`
	CanCreateGroup bool   `json:"can_create_group"`
	InviteCode     string `json:"invite_code"`
	CanBindInvite  bool   `json:"can_bind_invite_code"`
	OwnerID        uint   `json:"owner_id"` // 前端归属标识：未归属/超管名下为 0，管理员/助理名下为实际 ID
	OwnerName      string `json:"owner_name,omitempty"`
	IsAssigned     bool   `json:"is_assigned"` // 归属管理员/助理为 true；未归属或归属超管（含 feng88）为 false
}

// GetProfile 获取当前用户资料。
func GetProfile(ctx context.Context, imUserID string) (*ProfileResult, error) {
	repo := repository.NewUserRepository()
	user, err := repo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}
	// 历史建号只写了 DB AllowType，未同步到 IM（默认 AllowAny）。拉取资料时尽力对齐。
	go syncAllowTypeToIM(imUserID, user.AllowType)
	return toProfileResult(user)
}

func syncAllowTypeToIM(imUserID string, allowType int8) {
	im := tencentim.NewClient()
	if !im.IsConfigured() {
		return
	}
	_ = im.SetAllowType(imUserID, allowType)
}

// UpdateProfileReq 可更新的字段，nil 表示不修改。
type UpdateProfileReq struct {
	WananID       *string
	Nickname      *string
	AvatarURL     *string
	SelfSignature *string
	Gender        *int8
	Birthday      *int // YYYYMMDD
	AllowType     *int8
}

// UpdateProfile 更新 DB + 同步到腾讯云 IM。
func UpdateProfile(ctx context.Context, imUserID string, req UpdateProfileReq) (*ProfileResult, error) {
	repo := repository.NewUserRepository()
	user, err := repo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	// 1. 收集变更字段，同步更新 model 和 IM ProfileItem 列表
	var items []tencentim.ProfileItemDetail

	if req.WananID != nil {
		wananID, err := normalizeWananID(*req.WananID)
		if err != nil {
			return nil, ProfileErr(err.Error())
		}

		phoneExists, err := repo.ExistsByPhone(wananID)
		if err != nil {
			return nil, err
		}
		if phoneExists {
			return nil, ProfileErr("弥洛蓝序号不能与手机号相同，请更换后重试！")
		}

		currentWananID := optionalString(user.WananID)
		if currentWananID != wananID {
			exists, err := repo.ExistsByWananID(wananID)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, ProfileErr("当前弥洛蓝序号已被使用，请更换后重试！")
			}
		}

		user.WananID = &wananID
	}
	if req.Nickname != nil {
		nickname := strings.TrimSpace(*req.Nickname)
		if nickname == "" {
			return nil, ProfileErr("昵称不能为空")
		}

		user.Nickname = nickname
		items = append(items, tencentim.ProfileItemDetail{Tag: tencentim.TagNick, Value: nickname})
		// 首次更新资料（设置了昵称）时自动激活账号
		if !user.IsActivated && nickname != "" {
			user.IsActivated = true
		}
	}
	if req.AvatarURL != nil {
		user.AvatarURL = *req.AvatarURL
		items = append(items, tencentim.ProfileItemDetail{Tag: tencentim.TagImage, Value: *req.AvatarURL})
	}
	if req.SelfSignature != nil {
		user.SelfSignature = *req.SelfSignature
		items = append(items, tencentim.ProfileItemDetail{Tag: tencentim.TagSelfSignature, Value: *req.SelfSignature})
	}
	if req.Gender != nil {
		user.Gender = *req.Gender
		items = append(items, tencentim.ProfileItemDetail{Tag: tencentim.TagGender, Value: genderTag(*req.Gender)})
	}
	if req.Birthday != nil && *req.Birthday > 0 {
		user.Birthday = birthdayToTime(*req.Birthday)
		items = append(items, tencentim.ProfileItemDetail{Tag: tencentim.TagBirthDay, Value: uint32(*req.Birthday)})
	}
	if req.AllowType != nil {
		user.AllowType = *req.AllowType
		items = append(items, tencentim.ProfileItemDetail{Tag: tencentim.TagAllowType, Value: allowTypeTag(*req.AllowType)})
	}

	// 2. 持久化到 DB
	if err := repo.Update(user); err != nil {
		if profileError := mapProfilePersistenceError(err); profileError != nil {
			return nil, *profileError
		}
		return nil, err
	}

	// The desired business profile is saved, but a failed IM update is not success.
	// Repeating the same profile patch retries synchronization without uploading again.
	im := tencentim.NewClient()
	if err := syncSavedProfile(imUserID, user.AvatarURL, items, im.IsConfigured(), im.PortraitSet); err != nil {
		return nil, err
	}

	return toProfileResult(user)
}

// ── helpers ──────────────────────────────────────────────────────────────────

func toProfileResult(u *model.User) (*ProfileResult, error) {
	return toProfileResultWithOwnerLookup(u, func(id uint) (*model.AdminUser, error) {
		return repository.NewAdminUserRepository().FindByID(id)
	})
}

func toProfileResultWithOwnerLookup(u *model.User, findOwner func(uint) (*model.AdminUser, error)) (*ProfileResult, error) {
	r := &ProfileResult{
		IMUserID:       u.IMUserID,
		Phone:          strings.TrimSpace(u.Phone),
		WananID:        optionalString(u.WananID),
		Nickname:       u.Nickname,
		UserTag:        strings.TrimSpace(u.UserTag),
		AvatarURL:      u.AvatarURL,
		SelfSignature:  u.SelfSignature,
		Gender:         u.Gender,
		AllowType:      u.AllowType,
		CanCreateGroup: u.CanCreateGroup,
		InviteCode:     strings.TrimSpace(u.InviteCode),
		CanBindInvite:  strings.TrimSpace(u.InviteCode) == "",
		OwnerID:        u.OwnerID,
		OwnerRevision:  u.OwnerRevision,
	}
	var owner *model.AdminUser
	if u.OwnerID != 0 {
		var err error
		owner, err = findOwner(u.OwnerID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if owner != nil {
			r.OwnerName = strings.TrimSpace(owner.DisplayName)
		}
	}
	summary := model.SummarizeOwner(u, owner)
	r.OwnerID = summary.OwnerID
	r.IsAssigned = summary.IsAssigned
	r.OwnerState = summary.OwnerState
	r.OwnerRole = summary.OwnerRole
	r.OwnerReason = summary.ReasonCode
	if summary.OwnerState == "pool" {
		r.OwnerName = ""
	}

	if u.Birthday != nil {
		t := *u.Birthday
		r.Birthday = t.Year()*10000 + int(t.Month())*100 + t.Day()
	}
	return r, nil
}

func genderTag(g int8) string {
	switch g {
	case 1:
		return "Gender_Type_Male"
	case 2:
		return "Gender_Type_Female"
	default:
		return "Gender_Type_Unknown"
	}
}

func allowTypeTag(a int8) string {
	return tencentim.AllowTypeTag(a)
}

func birthdayToTime(yyyymmdd int) *time.Time {
	t := time.Date(
		yyyymmdd/10000,
		time.Month((yyyymmdd/100)%100),
		yyyymmdd%100,
		0, 0, 0, 0, time.Local,
	)
	return &t
}

func normalizeWananID(raw string) (string, error) {
	wananID := strings.ToLower(strings.TrimSpace(raw))
	if wananID == "" {
		return "", ProfileErr("无忧号不能为空")
	}
	if !wananIDPattern.MatchString(wananID) {
		return "", ProfileErr("无忧号需为 6-20 位字母或数字，可含下划线或短横线")
	}
	return wananID, nil
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func mapProfilePersistenceError(err error) *ProfileErr {
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "duplicate entry") && strings.Contains(message, "wanan_id"):
		profileErr := ProfileErr("当前弥洛蓝序号已被使用，请更换后重试！")
		return &profileErr
	default:
		return nil
	}
}
