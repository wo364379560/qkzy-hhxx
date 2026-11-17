package controller_admin

import "github.com/ixayda/ix-Go/internal/app/server/repository"

// StatsResult 仪表盘统计数据。
type StatsResult struct {
	UserCount          int64 `json:"userCount"`
	InviteCodeCount    int64 `json:"inviteCodeCount"`
	GroupCount         int64 `json:"groupCount"`
	AssistantCount     int64 `json:"assistantCount"`
	ReportCount        int64 `json:"reportCount"`
	PendingReportCount int64 `json:"pendingReportCount"`

	UserTotal           int64 `json:"userTotal"`
	InviteCodeTotal     int64 `json:"inviteCodeTotal"`
	GroupTotal          int64 `json:"groupTotal"`
	AssistantTotal      int64 `json:"assistantTotal"`
	ReportTotal         int64 `json:"reportTotal"`
	PendingReviewCount  int64 `json:"pendingReviewCount"`
}

// GetStats 获取管理员仪表盘统计（全量）。
func GetStats() (StatsResult, error) {
	userRepo := repository.NewUserRepository()
	userCount, err := userRepo.Count()
	if err != nil {
		return StatsResult{}, err
	}

	icRepo := repository.NewInviteCodeRepository()
	icCount, err := icRepo.Count()
	if err != nil {
		return StatsResult{}, err
	}

	groupRepo := repository.NewGroupRepository()
	groupCount, err := groupRepo.Count()
	if err != nil {
		return StatsResult{}, err
	}

	adminRepo := repository.NewAdminUserRepository()
	assistantCount, err := adminRepo.CountAssistants()
	if err != nil {
		return StatsResult{}, err
	}

	reportRepo := repository.NewReportRepository()
	reportCount, err := reportRepo.CountByStatus("")
	if err != nil {
		return StatsResult{}, err
	}
	pendingReportCount, err := reportRepo.CountByStatus("pending")
	if err != nil {
		return StatsResult{}, err
	}

	return StatsResult{
		UserCount:          userCount,
		InviteCodeCount:    icCount,
		GroupCount:         groupCount,
		AssistantCount:     assistantCount,
		ReportCount:        reportCount,
		PendingReportCount: pendingReportCount,
		UserTotal:          userCount,
		InviteCodeTotal:    icCount,
		GroupTotal:         groupCount,
		AssistantTotal:     assistantCount,
		ReportTotal:        reportCount,
		PendingReviewCount: pendingReportCount,
	}, nil
}

// GetAssistantStats 助理仪表盘统计（仅自己名下数据）。
func GetAssistantStats(assistantID uint) (StatsResult, error) {
	userRepo := repository.NewUserRepository()
	userCount, err := userRepo.CountByOwner(assistantID)
	if err != nil {
		return StatsResult{}, err
	}

	icRepo := repository.NewInviteCodeRepository()
	icCount, err := icRepo.CountByOwner(assistantID)
	if err != nil {
		return StatsResult{}, err
	}

	reportRepo := repository.NewReportRepository()
	reportCount, err := reportRepo.CountByOwnerAndStatus(assistantID, "")
	if err != nil {
		return StatsResult{}, err
	}
	pendingReportCount, err := reportRepo.CountByOwnerAndStatus(assistantID, "pending")
	if err != nil {
		return StatsResult{}, err
	}

	return StatsResult{
		UserCount:          userCount,
		InviteCodeCount:    icCount,
		ReportCount:        reportCount,
		PendingReportCount: pendingReportCount,
		UserTotal:          userCount,
		InviteCodeTotal:    icCount,
		ReportTotal:        reportCount,
		PendingReviewCount: pendingReportCount,
	}, nil
}

// GetManagerStats 超级管理员/管理员按作用域返回仪表盘统计。
func GetManagerStats(adminID uint) (StatsResult, error) {
	actor, err := BuildActorContext(adminID)
	if err != nil {
		return StatsResult{}, err
	}
	if IsSuperAdminRole(actor.Role) {
		return GetStats()
	}

	userRepo := repository.NewUserRepository()
	userCounts, err := userRepo.CountByOwners(actor.ScopeAdminIDs)
	if err != nil {
		return StatsResult{}, err
	}
	var userCount int64
	for _, count := range userCounts {
		userCount += count
	}

	icRepo := repository.NewInviteCodeRepository()
	icCount, err := icRepo.CountByOwners(actor.ScopeAdminIDs)
	if err != nil {
		return StatsResult{}, err
	}

	groupRepo := repository.NewGroupRepository()
	groupCount, err := groupRepo.CountByOwners(actor.ScopeAdminIDs)
	if err != nil {
		return StatsResult{}, err
	}

	assistantCount := int64(0)
	if actor.Role == RoleAdmin {
		assistants, err := repository.NewAdminUserRepository().ListByParentAdminID(actor.AdminID)
		if err != nil {
			return StatsResult{}, err
		}
		assistantCount = int64(len(assistants))
	}

	reportRepo := repository.NewReportRepository()
	reportCount, err := reportRepo.CountByOwnersAndStatus(actor.ScopeAdminIDs, "")
	if err != nil {
		return StatsResult{}, err
	}
	pendingReportCount, err := reportRepo.CountByOwnersAndStatus(actor.ScopeAdminIDs, "pending")
	if err != nil {
		return StatsResult{}, err
	}

	return StatsResult{
		UserCount:          userCount,
		InviteCodeCount:    icCount,
		GroupCount:         groupCount,
		AssistantCount:     assistantCount,
		ReportCount:        reportCount,
		PendingReportCount: pendingReportCount,
		UserTotal:          userCount,
		InviteCodeTotal:    icCount,
		GroupTotal:         groupCount,
		AssistantTotal:     assistantCount,
		ReportTotal:        reportCount,
		PendingReviewCount: pendingReportCount,
	}, nil
}
