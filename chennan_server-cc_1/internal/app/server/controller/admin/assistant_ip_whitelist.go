package controller_admin

// AssistantIPWhitelistRuleResult 是助理 IP 白名单规则响应。
type AssistantIPWhitelistRuleResult struct {
	ID   uint   `json:"id"`
	CIDR string `json:"cidr"`
	Note string `json:"note"`
}

// AssistantIPWhitelistResult 是助理 IP 白名单详情响应。
type AssistantIPWhitelistResult struct {
	AssistantID uint                             `json:"assistantId"`
	Enabled     bool                             `json:"enabled"`
	Rules       []AssistantIPWhitelistRuleResult `json:"rules"`
}

// SaveAssistantIPWhitelistRuleReq 是白名单规则保存参数。
type SaveAssistantIPWhitelistRuleReq struct {
	CIDR string
	Note string
}

// SaveAssistantIPWhitelistReq 是助理 IP 白名单保存参数。
type SaveAssistantIPWhitelistReq struct {
	ActorID     uint
	AssistantID uint
	Rules       []SaveAssistantIPWhitelistRuleReq
	UpdatedBy   uint
}

// GetAssistantIPWhitelist 获取单个助理的 IP 白名单配置。
func GetAssistantIPWhitelist(actorID, assistantID uint) (*AssistantIPWhitelistResult, error) {
	assistant, err := findManagedAssistant(actorID, assistantID)
	if err != nil {
		return nil, err
	}

	result, err := getAdminUserIPWhitelist(assistant)
	if err != nil {
		return nil, err
	}
	return toAssistantIPWhitelistResult(result), nil
}

// SaveAssistantIPWhitelist 保存单个助理的 IP 白名单配置。
func SaveAssistantIPWhitelist(req SaveAssistantIPWhitelistReq) (*AssistantIPWhitelistResult, error) {
	assistant, err := findManagedAssistant(req.ActorID, req.AssistantID)
	if err != nil {
		return nil, err
	}

	result, err := saveAdminUserIPWhitelist(assistant, req.UpdatedBy, toAdminIPWhitelistRuleReqs(req.Rules))
	if err != nil {
		return nil, err
	}
	return toAssistantIPWhitelistResult(result), nil
}

func toAdminIPWhitelistRuleReqs(rules []SaveAssistantIPWhitelistRuleReq) []SaveAdminIPWhitelistRuleReq {
	result := make([]SaveAdminIPWhitelistRuleReq, 0, len(rules))
	for _, rule := range rules {
		result = append(result, SaveAdminIPWhitelistRuleReq{
			CIDR: rule.CIDR,
			Note: rule.Note,
		})
	}
	return result
}

func toAssistantIPWhitelistResult(result *AdminIPWhitelistResult) *AssistantIPWhitelistResult {
	resultRules := make([]AssistantIPWhitelistRuleResult, 0, len(result.Rules))
	for _, rule := range result.Rules {
		resultRules = append(resultRules, AssistantIPWhitelistRuleResult{
			ID:   rule.ID,
			CIDR: rule.CIDR,
			Note: rule.Note,
		})
	}
	return &AssistantIPWhitelistResult{
		AssistantID: result.AdminUserID,
		Enabled:     result.Enabled,
		Rules:       resultRules,
	}
}
