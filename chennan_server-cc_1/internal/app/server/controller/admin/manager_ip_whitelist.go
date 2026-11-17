package controller_admin

// SaveManagerIPWhitelistReq 是管理员 IP 白名单保存参数。
type SaveManagerIPWhitelistReq struct {
	ActorID   uint
	ManagerID uint
	Rules     []SaveAdminIPWhitelistRuleReq
	UpdatedBy uint
}

// GetManagerIPWhitelist 获取单个管理员的 IP 白名单配置。
func GetManagerIPWhitelist(actorID, managerID uint) (*AdminIPWhitelistResult, error) {
	manager, err := findManagedManager(actorID, managerID)
	if err != nil {
		return nil, err
	}

	return getAdminUserIPWhitelist(manager)
}

// SaveManagerIPWhitelist 保存单个管理员的 IP 白名单配置。
func SaveManagerIPWhitelist(req SaveManagerIPWhitelistReq) (*AdminIPWhitelistResult, error) {
	manager, err := findManagedManager(req.ActorID, req.ManagerID)
	if err != nil {
		return nil, err
	}

	return saveAdminUserIPWhitelist(manager, req.UpdatedBy, req.Rules)
}
