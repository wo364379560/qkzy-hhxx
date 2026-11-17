package controller_admin

import (
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/repository"
)

type SMSSendLogResult struct {
	ID           uint   `json:"id"`
	Phone        string `json:"phone"`
	IPAddress    string `json:"ipAddress"`
	DeviceID     string `json:"deviceID"`
	Scene        string `json:"scene"`
	Status       string `json:"status"`
	RejectReason string `json:"rejectReason"`
	Provider     string `json:"provider"`
	CreatedAt    string `json:"createdAt"`
}

type ListSMSSendLogsResult struct {
	List  []SMSSendLogResult `json:"list"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Size  int                `json:"size"`
}

func ListSMSSendLogs(page, pageSize int, keyword, scene, status string) (*ListSMSSendLogsResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	logs, total, err := repository.NewSMSSendLogRepository().ListPaged(page, pageSize, keyword, scene, status)
	if err != nil {
		return nil, err
	}

	items := make([]SMSSendLogResult, 0, len(logs))
	for _, item := range logs {
		items = append(items, SMSSendLogResult{
			ID:           item.ID,
			Phone:        item.Phone,
			IPAddress:    item.IPAddress,
			DeviceID:     item.DeviceID,
			Scene:        item.Scene,
			Status:       item.Status,
			RejectReason: strings.TrimSpace(item.RejectReason),
			Provider:     item.Provider,
			CreatedAt:    item.CreatedAt.Format(time.DateTime),
		})
	}

	return &ListSMSSendLogsResult{
		List:  items,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}
