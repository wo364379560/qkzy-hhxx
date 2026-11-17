package controller_admin

import (
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
)

// ReportResult 举报信息响应。
type ReportResult struct {
	ID           uint   `json:"id"`
	ReporterID   uint   `json:"reporterID"`
	ReporterName string `json:"reporterName"`
	TargetID     uint   `json:"targetID"`
	TargetType   string `json:"targetType"`
	TargetName   string `json:"targetName"`
	Category     string `json:"category"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`

	ContentTargetType string `json:"contentTargetType"`
	ContentTargetName string `json:"contentTargetName"`
}

func toReportResult(r *model.Report) ReportResult {
	return ReportResult{
		ID:           r.ID,
		ReporterID:   r.ReporterID,
		ReporterName: r.ReporterName,
		TargetID:     r.TargetID,
		TargetType:   r.TargetType,
		TargetName:   r.TargetName,
		Category:     r.Category,
		Description:  r.Description,
		Status:       r.Status,
		CreatedAt:    r.CreatedAt.Format("2006-01-02 15:04:05"),
		ContentTargetType: mapReportTargetType(r.TargetType),
		ContentTargetName: r.TargetName,
	}
}

func mapReportTargetType(targetType string) string {
	switch targetType {
	case "community_post":
		return "saving_post"
	case "community_comment":
		return "saving_comment"
	default:
		return targetType
	}
}

// ListReportsReq 查询参数。
type ListReportsReq struct {
	Page          int
	PageSize      int
	ScopeOwnerIDs []uint
	OwnerID       uint   // 0 表示不限制（管理员）
	Status        string // "" 表示全部
}

// ListReports 分页列出举报。
func ListReports(req ListReportsReq) (PageResult[ReportResult], error) {
	repo := repository.NewReportRepository()
	reports, total, err := repo.ListPaged(req.Page, req.PageSize, req.ScopeOwnerIDs, req.OwnerID, req.Status)
	if err != nil {
		return PageResult[ReportResult]{}, err
	}
	list := make([]ReportResult, 0, len(reports))
	for i := range reports {
		list = append(list, toReportResult(&reports[i]))
	}
	return PageResult[ReportResult]{List: list, Total: total}, nil
}

// ReviewReport 标记举报为已处理。
func ReviewReport(actorID uint, isAdmin bool, reportID uint) error {
	repo := repository.NewReportRepository()
	if _, err := findManagedReport(actorID, isAdmin, reportID); err != nil {
		return err
	}
	return repo.UpdateStatus(reportID, "reviewed")
}

// DismissReport 忽略举报。
func DismissReport(actorID uint, isAdmin bool, reportID uint) error {
	repo := repository.NewReportRepository()
	if _, err := findManagedReport(actorID, isAdmin, reportID); err != nil {
		return err
	}
	return repo.UpdateStatus(reportID, "dismissed")
}

// CreateReportReq 用户 App 端提交举报（内部调用）。
type CreateReportReq struct {
	ReporterID   uint
	ReporterName string
	TargetID     uint   // 被举报对象 DB ID（用户/内容）
	TargetIMID   string // IM 侧 ID（userID/groupID）或内容字符串 ID
	TargetType   string // "user" | "group" | "community_post" | "community_comment"
	TargetName   string
	Category     string
	Description  string
	OwnerID      uint
}

// CreateReport 创建举报记录（由 chat-server 调用或内部使用）。
func CreateReport(req CreateReportReq) error {
	report := &model.Report{
		ReporterID:   req.ReporterID,
		ReporterName: req.ReporterName,
		TargetID:     req.TargetID,
		TargetIMID:   req.TargetIMID,
		TargetType:   req.TargetType,
		TargetName:   req.TargetName,
		Category:     req.Category,
		Description:  req.Description,
		Status:       "pending",
		OwnerID:      req.OwnerID,
		CreatedAt:    time.Now(),
	}
	return repository.NewReportRepository().Create(report)
}
