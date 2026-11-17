package controller_admin

import (
	"errors"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"gorm.io/gorm"
)

type OfflineActivityResult struct {
	ID                uint   `json:"id"`
	Title             string `json:"title"`
	Subtitle          string `json:"subtitle"`
	Description       string `json:"description"`
	City              string `json:"city"`
	Venue             string `json:"venue"`
	StartAt           string `json:"startAt"`
	EndAt             string `json:"endAt"`
	Quota             int    `json:"quota"`
	Status            string `json:"status"`
	RegistrationCount int    `json:"registrationCount"`
	CreatedBy         uint   `json:"createdBy"`
	CreatedAt         string `json:"createdAt"`

	EventID                uint   `json:"eventID"`
	EventTitle             string `json:"eventTitle"`
	EventSubtitle          string `json:"eventSubtitle"`
	EventDescription       string `json:"eventDescription"`
	EventCity              string `json:"eventCity"`
	EventVenue             string `json:"eventVenue"`
	EventStartAt           string `json:"eventStartAt"`
	EventEndAt             string `json:"eventEndAt"`
	EventQuota             int    `json:"eventQuota"`
	EventStatus            string `json:"eventStatus"`
	EventRegistrationCount int    `json:"eventRegistrationCount"`
}

type OfflineActivityRegistrationResult struct {
	UserID   uint   `json:"userID"`
	IMUserID string `json:"imUserID"`
	Nickname string `json:"nickname"`
	Phone    string `json:"phone"`
	JoinedAt string `json:"joinedAt"`

	ParticipantUserID   uint   `json:"participantUserID"`
	ParticipantIMUserID string `json:"participantIMUserID"`
	ParticipantNickname string `json:"participantNickname"`
	ParticipantPhone    string `json:"participantPhone"`
	ParticipantJoinedAt string `json:"participantJoinedAt"`
}

type ListOfflineActivitiesReq struct {
	Page     int
	PageSize int
	Status   string
}

type CreateOfflineActivityReq struct {
	Title       string
	Subtitle    string
	Description string
	City        string
	Venue       string
	StartAt     string
	EndAt       string
	Quota       int
	Status      string
	CreatedBy   uint
}

type UpdateOfflineActivityReq struct {
	Title       string
	Subtitle    string
	Description string
	City        string
	Venue       string
	StartAt     string
	EndAt       string
	Quota       int
	Status      string
}

func ListOfflineActivities(req ListOfflineActivitiesReq) (PageResult[OfflineActivityResult], error) {
	repo := repository.NewOfflineActivityRepository()
	statuses := make([]string, 0, 1)
	if strings.TrimSpace(req.Status) != "" {
		statuses = append(statuses, strings.TrimSpace(req.Status))
	}
	items, total, err := repo.ListPaged(req.Page, req.PageSize, statuses)
	if err != nil {
		return PageResult[OfflineActivityResult]{}, err
	}
	list := make([]OfflineActivityResult, 0, len(items))
	for i := range items {
		list = append(list, toOfflineActivityResult(&items[i]))
	}
	return PageResult[OfflineActivityResult]{List: list, Total: total}, nil
}

func CreateOfflineActivity(req CreateOfflineActivityReq) (*OfflineActivityResult, error) {
	activity, err := buildOfflineActivityModel(
		0,
		req.Title,
		req.Subtitle,
		req.Description,
		req.City,
		req.Venue,
		req.StartAt,
		req.EndAt,
		req.Quota,
		req.Status,
		req.CreatedBy,
	)
	if err != nil {
		return nil, err
	}

	repo := repository.NewOfflineActivityRepository()
	if err := repo.Create(activity); err != nil {
		return nil, err
	}
	result := toOfflineActivityResult(activity)
	return &result, nil
}

func UpdateOfflineActivity(activityID uint, req UpdateOfflineActivityReq) (*OfflineActivityResult, error) {
	repo := repository.NewOfflineActivityRepository()
	existing, err := repo.FindByID(activityID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		return nil, err
	}

	updated, err := buildOfflineActivityModel(
		existing.ID,
		req.Title,
		req.Subtitle,
		req.Description,
		req.City,
		req.Venue,
		req.StartAt,
		req.EndAt,
		req.Quota,
		req.Status,
		existing.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	updated.RegistrationCount = existing.RegistrationCount
	updated.CreatedAt = existing.CreatedAt

	if err := repo.Update(updated); err != nil {
		return nil, err
	}
	result := toOfflineActivityResult(updated)
	return &result, nil
}

func CloseOfflineActivity(activityID uint) error {
	repo := repository.NewOfflineActivityRepository()
	activity, err := repo.FindByID(activityID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("活动不存在")
		}
		return err
	}
	activity.Status = "closed"
	return repo.Update(activity)
}

func ListOfflineActivityRegistrations(activityID uint) ([]OfflineActivityRegistrationResult, error) {
	repo := repository.NewOfflineActivityRepository()
	if _, err := repo.FindByID(activityID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		return nil, err
	}

	rows, err := repo.ListRegistrations(activityID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []OfflineActivityRegistrationResult{}, nil
	}

	userRepo := repository.NewUserRepository()
	results := make([]OfflineActivityRegistrationResult, 0, len(rows))
	for _, row := range rows {
		user, err := userRepo.FindByID(row.UserID)
		if err != nil {
			continue
		}
		results = append(results, OfflineActivityRegistrationResult{
			UserID:   user.ID,
			IMUserID: user.IMUserID,
			Nickname: user.Nickname,
			Phone:    user.Phone,
			JoinedAt: row.JoinedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return results, nil
}

func toOfflineActivityResult(activity *model.OfflineActivity) OfflineActivityResult {
	return OfflineActivityResult{
		ID:                activity.ID,
		Title:             activity.Title,
		Subtitle:          activity.Subtitle,
		Description:       activity.Description,
		City:              activity.City,
		Venue:             activity.Venue,
		StartAt:           activity.StartAt.Format("2006-01-02 15:04:05"),
		EndAt:             activity.EndAt.Format("2006-01-02 15:04:05"),
		Quota:             activity.Quota,
		Status:            activity.Status,
		RegistrationCount: activity.RegistrationCount,
		CreatedBy:         activity.CreatedBy,
		CreatedAt:         activity.CreatedAt.Format("2006-01-02 15:04:05"),
		EventID:                activity.ID,
		EventTitle:             activity.Title,
		EventSubtitle:          activity.Subtitle,
		EventDescription:       activity.Description,
		EventCity:              activity.City,
		EventVenue:             activity.Venue,
		EventStartAt:           activity.StartAt.Format("2006-01-02 15:04:05"),
		EventEndAt:             activity.EndAt.Format("2006-01-02 15:04:05"),
		EventQuota:             activity.Quota,
		EventStatus:            activity.Status,
		EventRegistrationCount: activity.RegistrationCount,
	}
}

func buildOfflineActivityModel(
	id uint,
	title, subtitle, description, city, venue, startAt, endAt string,
	quota int,
	status string,
	createdBy uint,
) (*model.OfflineActivity, error) {
	title = strings.TrimSpace(title)
	subtitle = strings.TrimSpace(subtitle)
	description = strings.TrimSpace(description)
	city = strings.TrimSpace(city)
	venue = strings.TrimSpace(venue)
	status = strings.TrimSpace(status)

	if title == "" {
		return nil, errors.New("活动标题不能为空")
	}
	if city == "" {
		return nil, errors.New("城市不能为空")
	}
	if venue == "" {
		return nil, errors.New("活动地点不能为空")
	}
	if len([]rune(title)) > 64 {
		return nil, errors.New("活动标题不能超过 64 个字符")
	}
	if len([]rune(subtitle)) > 120 {
		return nil, errors.New("副标题不能超过 120 个字符")
	}
	if len([]rune(description)) > 1000 {
		return nil, errors.New("活动说明不能超过 1000 个字符")
	}
	if quota < 0 {
		return nil, errors.New("报名人数不能为负数")
	}
	if status == "" {
		status = "draft"
	}
	switch status {
	case "draft", "published", "closed", "cancelled":
	default:
		return nil, errors.New("活动状态不支持")
	}

	start, err := time.ParseInLocation("2006-01-02 15:04:05", startAt, time.Local)
	if err != nil {
		return nil, errors.New("开始时间格式必须为 YYYY-MM-DD HH:mm:ss")
	}
	end, err := time.ParseInLocation("2006-01-02 15:04:05", endAt, time.Local)
	if err != nil {
		return nil, errors.New("结束时间格式必须为 YYYY-MM-DD HH:mm:ss")
	}
	if !end.After(start) {
		return nil, errors.New("结束时间必须晚于开始时间")
	}

	return &model.OfflineActivity{
		ID:          id,
		Title:       title,
		Subtitle:    subtitle,
		Description: description,
		City:        city,
		Venue:       venue,
		StartAt:     start,
		EndAt:       end,
		Quota:       quota,
		Status:      status,
		CreatedBy:   createdBy,
	}, nil
}
