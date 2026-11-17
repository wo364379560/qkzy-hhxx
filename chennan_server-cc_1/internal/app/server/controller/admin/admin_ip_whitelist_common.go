package controller_admin

import (
	"errors"
	"strings"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/security"
)

const maxAdminIPWhitelistRules = 50

// AdminIPWhitelistRuleResult 是后台账号 IP 白名单规则响应。
type AdminIPWhitelistRuleResult struct {
	ID   uint   `json:"id"`
	CIDR string `json:"cidr"`
	Note string `json:"note"`
}

// AdminIPWhitelistResult 是后台账号 IP 白名单详情响应。
type AdminIPWhitelistResult struct {
	AdminUserID uint                         `json:"adminUserId"`
	Enabled     bool                         `json:"enabled"`
	Rules       []AdminIPWhitelistRuleResult `json:"rules"`
}

// SaveAdminIPWhitelistRuleReq 是后台账号白名单规则保存参数。
type SaveAdminIPWhitelistRuleReq struct {
	CIDR string
	Note string
}

func getAdminUserIPWhitelist(admin *model.AdminUser) (*AdminIPWhitelistResult, error) {
	ruleRepo := repository.NewAdminUserIPWhitelistRepository()
	rules, err := ruleRepo.ListByAdminUserID(admin.ID)
	if err != nil {
		return nil, err
	}

	admin.IPWhitelistEnabled = len(rules) > 0
	return buildAdminIPWhitelistResult(admin, rules), nil
}

func saveAdminUserIPWhitelist(admin *model.AdminUser, updatedBy uint, rules []SaveAdminIPWhitelistRuleReq) (*AdminIPWhitelistResult, error) {
	normalizedRules, err := normalizeAdminIPWhitelistRules(admin.ID, updatedBy, rules)
	if err != nil {
		return nil, err
	}
	enabled := len(normalizedRules) > 0

	ruleRepo := repository.NewAdminUserIPWhitelistRepository()
	if err := ruleRepo.ReplaceRules(admin.ID, normalizedRules); err != nil {
		return nil, err
	}
	if err := repository.NewAdminUserRepository().UpdateIPWhitelistEnabled(admin.ID, enabled); err != nil {
		return nil, err
	}

	admin.IPWhitelistEnabled = enabled
	return buildAdminIPWhitelistResult(admin, normalizedRules), nil
}

func normalizeAdminIPWhitelistRules(adminUserID, updatedBy uint, rules []SaveAdminIPWhitelistRuleReq) ([]model.AdminUserIPWhitelist, error) {
	if len(rules) > maxAdminIPWhitelistRules {
		return nil, errors.New("白名单规则最多 50 条")
	}

	seen := make(map[string]struct{}, len(rules))
	result := make([]model.AdminUserIPWhitelist, 0, len(rules))
	for _, rule := range rules {
		cidr, err := security.NormalizeIPWhitelistRule(rule.CIDR)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[cidr]; ok {
			continue
		}
		seen[cidr] = struct{}{}
		result = append(result, model.AdminUserIPWhitelist{
			AdminUserID: adminUserID,
			CIDR:        cidr,
			Note:        strings.TrimSpace(rule.Note),
			CreatedBy:   updatedBy,
		})
	}
	return result, nil
}

func buildAdminIPWhitelistResult(admin *model.AdminUser, rules []model.AdminUserIPWhitelist) *AdminIPWhitelistResult {
	resultRules := make([]AdminIPWhitelistRuleResult, 0, len(rules))
	for _, rule := range rules {
		resultRules = append(resultRules, AdminIPWhitelistRuleResult{
			ID:   rule.ID,
			CIDR: rule.CIDR,
			Note: rule.Note,
		})
	}

	return &AdminIPWhitelistResult{
		AdminUserID: admin.ID,
		Enabled:     admin.IPWhitelistEnabled,
		Rules:       resultRules,
	}
}
