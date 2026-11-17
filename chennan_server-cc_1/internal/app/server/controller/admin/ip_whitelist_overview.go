package controller_admin

import (
	"errors"

	"github.com/ixayda/ix-Go/internal/app/server/repository"
)

// ListIPWhitelistOverviewReq 查询后台账号白名单策略概览参数。
type ListIPWhitelistOverviewReq struct {
	ActorID   uint
	ActorRole string
	Page      int
	PageSize  int
	Keyword   string
	Status    string
	Enabled   *bool
}

// IPWhitelistOverviewResult 是后台账号白名单策略概览响应。
type IPWhitelistOverviewResult struct {
	ID                 uint     `json:"id"`
	Username           string   `json:"username"`
	Role               string   `json:"role"`
	Status             string   `json:"status"`
	ManagedCount       int64    `json:"managedCount"`
	IPWhitelistEnabled bool     `json:"ipWhitelistEnabled"`
	RuleCount          int      `json:"ruleCount"`
	RulePreview        []string `json:"rulePreview"`
	LastUpdatedAt      *string  `json:"lastUpdatedAt"`
}

// ListIPWhitelistOverview 分页列出后台账号白名单策略概览。
func ListIPWhitelistOverview(req ListIPWhitelistOverviewReq) (PageResult[IPWhitelistOverviewResult], error) {
	actor, err := BuildActorContext(req.ActorID)
	if err != nil {
		return PageResult[IPWhitelistOverviewResult]{}, err
	}
	if !HasManagerPrivileges(actor.Role) {
		return PageResult[IPWhitelistOverviewResult]{}, errors.New("仅超级管理员或管理员可操作")
	}

	adminRepo := repository.NewAdminUserRepository()
	roles := []string{RoleAssistant}
	var parentAdminID *uint
	if IsSuperAdminRole(actor.Role) {
		roles = []string{RoleAdmin, RoleAssistant}
	} else {
		parentAdminID = &actor.AdminID
	}

	admins, total, err := adminRepo.ListWhitelistAccountsPaged(req.Page, req.PageSize, roles, req.Keyword, req.Status, req.Enabled, parentAdminID)
	if err != nil {
		return PageResult[IPWhitelistOverviewResult]{}, err
	}

	list := make([]IPWhitelistOverviewResult, 0, len(admins))
	if len(admins) == 0 {
		return PageResult[IPWhitelistOverviewResult]{List: list, Total: total}, nil
	}

	ids := make([]uint, len(admins))
	for i, admin := range admins {
		ids[i] = admin.ID
	}

	userCounts, err := repository.NewUserRepository().CountByOwners(ids)
	if err != nil {
		return PageResult[IPWhitelistOverviewResult]{}, err
	}
	assistantCountMap, err := adminRepo.CountAssistantsByParents(ids)
	if err != nil {
		return PageResult[IPWhitelistOverviewResult]{}, err
	}

	rules, err := repository.NewAdminUserIPWhitelistRepository().ListByAdminUserIDs(ids)
	if err != nil {
		return PageResult[IPWhitelistOverviewResult]{}, err
	}

	ruleCountMap := make(map[uint]int, len(ids))
	rulePreviewMap := make(map[uint][]string, len(ids))
	lastUpdatedMap := make(map[uint]string, len(ids))
	for _, rule := range rules {
		ruleCountMap[rule.AdminUserID]++
		if len(rulePreviewMap[rule.AdminUserID]) < 3 {
			rulePreviewMap[rule.AdminUserID] = append(rulePreviewMap[rule.AdminUserID], rule.CIDR)
		}
		last := rule.UpdatedAt.Format("2006-01-02 15:04:05")
		if current, ok := lastUpdatedMap[rule.AdminUserID]; !ok || last > current {
			lastUpdatedMap[rule.AdminUserID] = last
		}
	}

	for _, admin := range admins {
		var lastUpdatedAt *string
		if value, ok := lastUpdatedMap[admin.ID]; ok {
			lastUpdatedAt = &value
		}

		managedCount := userCounts[admin.ID]
		if admin.Role == RoleAdmin {
			managedCount = assistantCountMap[admin.ID]
		}

		list = append(list, IPWhitelistOverviewResult{
			ID:                 admin.ID,
			Username:           admin.Username,
			Role:               admin.Role,
			Status:             admin.Status,
			ManagedCount:       managedCount,
			IPWhitelistEnabled: admin.IPWhitelistEnabled,
			RuleCount:          ruleCountMap[admin.ID],
			RulePreview:        rulePreviewMap[admin.ID],
			LastUpdatedAt:      lastUpdatedAt,
		})
	}

	return PageResult[IPWhitelistOverviewResult]{List: list, Total: total}, nil
}
