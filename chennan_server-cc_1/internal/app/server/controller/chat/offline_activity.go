package controller_chat

import (
	"errors"

	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"gorm.io/gorm"
)

type OfflineActivityAppResult struct {
	ID                uint   `json:"id"`
	Title             string `json:"title"`
	Subtitle          string `json:"subtitle"`
	Description       string `json:"description"`
	City              string `json:"city"`
	Venue             string `json:"venue"`
	StartAt           string `json:"start_at"`
	EndAt             string `json:"end_at"`
	Quota             int    `json:"quota"`
	Status            string `json:"status"`
	RegistrationCount int    `json:"registration_count"`
	IsRegistered      bool   `json:"is_registered"`
}

type OfflineActivityListResult struct {
	Items []OfflineActivityAppResult `json:"items"`
}

type ToggleOfflineActivityRegistrationResult struct {
	IsRegistered      bool `json:"is_registered"`
	RegistrationCount int  `json:"registration_count"`
}

func ListOfflineActivities(imUserID string) (*OfflineActivityListResult, error) {
	userRepo := repository.NewUserRepository()
	user, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	repo := repository.NewOfflineActivityRepository()
	items, err := repo.ListPublished()
	if err != nil {
		return nil, err
	}

	activityIDs := make([]uint, 0, len(items))
	for _, item := range items {
		activityIDs = append(activityIDs, item.ID)
	}
	registeredMap, err := repo.ListRegisteredActivityIDs(user.ID, activityIDs)
	if err != nil {
		return nil, err
	}

	results := make([]OfflineActivityAppResult, 0, len(items))
	for _, item := range items {
		_, isRegistered := registeredMap[item.ID]
		results = append(results, OfflineActivityAppResult{
			ID:                item.ID,
			Title:             item.Title,
			Subtitle:          item.Subtitle,
			Description:       item.Description,
			City:              item.City,
			Venue:             item.Venue,
			StartAt:           item.StartAt.Format("2006-01-02 15:04:05"),
			EndAt:             item.EndAt.Format("2006-01-02 15:04:05"),
			Quota:             item.Quota,
			Status:            item.Status,
			RegistrationCount: item.RegistrationCount,
			IsRegistered:      isRegistered,
		})
	}

	return &OfflineActivityListResult{Items: results}, nil
}

func ToggleOfflineActivityRegistration(imUserID string, activityID uint) (*ToggleOfflineActivityRegistrationResult, error) {
	userRepo := repository.NewUserRepository()
	user, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	repo := repository.NewOfflineActivityRepository()
	if _, err := repo.FindByID(activityID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		return nil, err
	}

	isRegistered, registrationCount, err := repo.ToggleRegistration(activityID, user.ID)
	if err != nil {
		return nil, err
	}

	return &ToggleOfflineActivityRegistrationResult{
		IsRegistered:      isRegistered,
		RegistrationCount: registrationCount,
	}, nil
}
