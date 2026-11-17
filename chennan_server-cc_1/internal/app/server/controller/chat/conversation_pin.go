package controller_chat

import (
	"errors"
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/repository"
)

type ConversationCustomPinResult struct {
	ConversationID string `json:"conversationID"`
	PinOrder       int    `json:"pinOrder"`
	PinnedAt       string `json:"pinnedAt"`
}

func ListConversationCustomPins(imUserID string) ([]ConversationCustomPinResult, error) {
	userRepo := repository.NewUserRepository()
	user, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return nil, err
	}

	rows, err := repository.NewUserConversationCustomPinRepository().ListByOwner(user.ID)
	if err != nil {
		return nil, err
	}

	result := make([]ConversationCustomPinResult, 0, len(rows))
	for _, row := range rows {
		result = append(result, ConversationCustomPinResult{
			ConversationID: row.ConversationID,
			PinOrder:       row.PinOrder,
			PinnedAt:       row.PinnedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return result, nil
}

func PinConversationCustom(imUserID, conversationID string, pinOrder int) error {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return errors.New("conversationID不能为空")
	}
	if pinOrder <= 0 {
		pinOrder = 1
	}

	userRepo := repository.NewUserRepository()
	user, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return err
	}

	return repository.NewUserConversationCustomPinRepository().Upsert(user.ID, conversationID, pinOrder)
}

func UnpinConversationCustom(imUserID, conversationID string) error {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return errors.New("conversationID不能为空")
	}

	userRepo := repository.NewUserRepository()
	user, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return err
	}

	return repository.NewUserConversationCustomPinRepository().Delete(user.ID, conversationID)
}

func ReorderConversationCustomPins(imUserID string, conversationIDs []string) error {
	userRepo := repository.NewUserRepository()
	user, err := userRepo.FindByIMUserID(imUserID)
	if err != nil {
		return err
	}

	filtered := make([]string, 0, len(conversationIDs))
	seen := make(map[string]struct{}, len(conversationIDs))
	for _, item := range conversationIDs {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		filtered = append(filtered, item)
	}

	return repository.NewUserConversationCustomPinRepository().Reorder(user.ID, filtered)
}
