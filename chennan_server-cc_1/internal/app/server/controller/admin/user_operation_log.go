package controller_admin

import (
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
)

const (
	UserOperationAccountDelete = "account_delete"
)

// UserOperationLogResult is the admin response for mobile user operation logs.
type UserOperationLogResult struct {
	ID              uint   `json:"id"`
	UserID          *uint  `json:"userId"`
	IMUserID        string `json:"imUserId"`
	Phone           string `json:"phone"`
	Nickname        string `json:"nickname"`
	OwnerID         *uint  `json:"ownerId"`
	OwnerName       string `json:"ownerName"`
	OperationType   string `json:"operationType"`
	OperationName   string `json:"operationName"`
	OperationDetail string `json:"operationDetail"`
	IPAddress       string `json:"ipAddress"`
	UserAgent       string `json:"userAgent"`
	DeviceID        string `json:"deviceId"`
	DeviceName      string `json:"deviceName"`
	Platform        string `json:"platform"`
	AppVersion      string `json:"appVersion"`
	Status          string `json:"status"`
	FailureReason   string `json:"failureReason"`
	CreatedAt       string `json:"createdAt"`
}

// ListUserOperationLogsReq contains admin query filters.
type ListUserOperationLogsReq struct {
	Page          int
	PageSize      int
	Keyword       string
	OperationType string
	Status        string
	Phone         string
	IMUserID      string
	OwnerID       uint
	IPAddress     string
	StartTime     *time.Time
	EndTime       *time.Time
	ScopeOwnerIDs []uint
}

// ListUserOperationLogs returns paginated mobile user operation logs.
func ListUserOperationLogs(req ListUserOperationLogsReq) (PageResult[UserOperationLogResult], error) {
	rows, total, err := repository.NewUserOperationLogRepository().ListPaged(
		req.Page,
		req.PageSize,
		repository.ListUserOperationLogFilter{
			Keyword:       req.Keyword,
			OperationType: req.OperationType,
			Status:        req.Status,
			Phone:         req.Phone,
			IMUserID:      req.IMUserID,
			OwnerID:       req.OwnerID,
			IPAddress:     req.IPAddress,
			StartTime:     req.StartTime,
			EndTime:       req.EndTime,
			ScopeOwnerIDs: req.ScopeOwnerIDs,
		},
	)
	if err != nil {
		return PageResult[UserOperationLogResult]{}, err
	}

	list := make([]UserOperationLogResult, 0, len(rows))
	for _, row := range rows {
		list = append(list, toUserOperationLogResult(row))
	}
	return PageResult[UserOperationLogResult]{List: list, Total: total}, nil
}

func toUserOperationLogResult(row model.UserOperationLog) UserOperationLogResult {
	return UserOperationLogResult{
		ID:              row.ID,
		UserID:          row.UserID,
		IMUserID:        row.IMUserID,
		Phone:           row.Phone,
		Nickname:        row.Nickname,
		OwnerID:         row.OwnerID,
		OwnerName:       row.OwnerName,
		OperationType:   row.OperationType,
		OperationName:   row.OperationName,
		OperationDetail: row.OperationDetail,
		IPAddress:       row.IPAddress,
		UserAgent:       row.UserAgent,
		DeviceID:        row.DeviceID,
		DeviceName:      row.DeviceName,
		Platform:        row.Platform,
		AppVersion:      row.AppVersion,
		Status:          row.Status,
		FailureReason:   row.FailureReason,
		CreatedAt:       row.CreatedAt.Format(time.DateTime),
	}
}
