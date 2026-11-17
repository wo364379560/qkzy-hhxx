package quietlog

import (
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
)

// ChatQuietHoursHitInput is the metadata recorded when a quiet-hours rule blocks an action.
type ChatQuietHoursHitInput struct {
	Action          string
	Source          string
	CallbackCommand string
	ActorIMUserID   string
	ActorPhone      string
	TargetAccount   string
	TargetGroupID   string
	IPAddress       string
	UserAgent       string
	Reason          string
}

// RecordChatQuietHoursHit persists a quiet-hours blocked-action log.
func RecordChatQuietHoursHit(input ChatQuietHoursHitInput) error {
	log := &model.ChatQuietHoursHitLog{
		Action:          strings.TrimSpace(input.Action),
		Source:          strings.TrimSpace(input.Source),
		CallbackCommand: strings.TrimSpace(input.CallbackCommand),
		IMUserID:        strings.TrimSpace(input.ActorIMUserID),
		Phone:           strings.TrimSpace(input.ActorPhone),
		TargetAccount:   strings.TrimSpace(input.TargetAccount),
		TargetGroupID:   strings.TrimSpace(input.TargetGroupID),
		IPAddress:       strings.TrimSpace(input.IPAddress),
		UserAgent:       strings.TrimSpace(input.UserAgent),
		Reason:          strings.TrimSpace(input.Reason),
	}

	if log.IMUserID != "" {
		if user, err := repository.NewUserRepository().FindByIMUserID(log.IMUserID); err == nil && user != nil {
			log.UserID = &user.ID
			log.OwnerID = &user.OwnerID
			if log.Phone == "" {
				log.Phone = user.Phone
			}
		}
	}

	return repository.NewChatQuietHoursHitLogRepository().Create(log)
}
