package controller_admin

import (
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/repository"
)

// ChatQuietHoursHitLogResult is the admin response for a quiet-hours blocked-action hit.
type ChatQuietHoursHitLogResult struct {
	ID              uint   `json:"id"`
	Action          string `json:"action"`
	Source          string `json:"source"`
	CallbackCommand string `json:"callbackCommand"`
	UserID          *uint  `json:"userId"`
	OwnerID         *uint  `json:"ownerId"`
	IMUserID        string `json:"imUserId"`
	Phone           string `json:"phone"`
	TargetAccount   string `json:"targetAccount"`
	TargetGroupID   string `json:"targetGroupId"`
	IPAddress       string `json:"ipAddress"`
	UserAgent       string `json:"userAgent"`
	Reason          string `json:"reason"`
	CreatedAt       string `json:"createdAt"`
}

// ListChatQuietHoursHitLogsReq contains paging/filter params for quiet-hours logs.
type ListChatQuietHoursHitLogsReq struct {
	Page          int
	PageSize      int
	Keyword       string
	Action        string
	Source        string
	ScopeOwnerIDs []uint
}

// ListChatQuietHoursHitLogs pages quiet-hours hit logs.
func ListChatQuietHoursHitLogs(req ListChatQuietHoursHitLogsReq) (PageResult[ChatQuietHoursHitLogResult], error) {
	rows, total, err := repository.NewChatQuietHoursHitLogRepository().ListPaged(
		req.Page,
		req.PageSize,
		req.Keyword,
		req.Action,
		req.Source,
		req.ScopeOwnerIDs,
	)
	if err != nil {
		return PageResult[ChatQuietHoursHitLogResult]{}, err
	}

	list := make([]ChatQuietHoursHitLogResult, 0, len(rows))
	for _, row := range rows {
		list = append(list, ChatQuietHoursHitLogResult{
			ID:              row.ID,
			Action:          row.Action,
			Source:          row.Source,
			CallbackCommand: row.CallbackCommand,
			UserID:          row.UserID,
			OwnerID:         row.OwnerID,
			IMUserID:        row.IMUserID,
			Phone:           row.Phone,
			TargetAccount:   row.TargetAccount,
			TargetGroupID:   row.TargetGroupID,
			IPAddress:       row.IPAddress,
			UserAgent:       row.UserAgent,
			Reason:          row.Reason,
			CreatedAt:       row.CreatedAt.Format(time.DateTime),
		})
	}

	return PageResult[ChatQuietHoursHitLogResult]{List: list, Total: total}, nil
}
