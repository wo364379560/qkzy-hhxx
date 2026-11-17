package handler_admin

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/handler/authlimit"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type createManagerReq struct {
	Username    string `json:"username" binding:"required,min=3,max=32"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name"`
}

func (req *createManagerReq) normalizeDisplayName() {
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.DisplayName == "" {
		req.DisplayName = strings.TrimSpace(req.Username)
	}
}

type updateManagerProfileReq struct {
	Username    *string `json:"username"`
	DisplayName string  `json:"display_name" binding:"required"`
}

type resetManagerPasswordReq struct {
	Password string `json:"password" binding:"required,min=8"`
}

type saveManagerIPWhitelistReq struct {
	Rules []saveManagerIPWhitelistRule `json:"rules"`
}

type saveManagerIPWhitelistRule struct {
	CIDR string `json:"cidr" binding:"required"`
	Note string `json:"note"`
}

// ListManagers 获取管理员列表（仅超级管理员）。
func ListManagers(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	page, pageSize := parsePagination(c)
	result, err := controller_admin.ListManagers(controller_admin.ListManagersReq{
		ActorID:  c.GetUint("admin_id"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// CreateManager 创建管理员（仅超级管理员）。
func CreateManager(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}

	var req createManagerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	// 兼容只有用户名和密码的旧版管理后台。
	req.normalizeDisplayName()
	result, err := controller_admin.CreateManager(controller_admin.CreateManagerReq{
		Username:    req.Username,
		Password:    req.Password,
		DisplayName: req.DisplayName,
		CreatedBy:   c.GetUint("admin_id"),
	})
	if err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "admin_user", &result.ID, result.Username, result.DisplayName)
	middleware.SetAdminAccessAction(c, "admin_user.create", "创建管理员")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("创建管理员账号；username=%s；display_name=%s", result.Username, result.DisplayName))
	response.OK(c, result)
}

// UpdateManagerProfile 修改管理员昵称（仅超级管理员）。
func UpdateManagerProfile(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}
	managerID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	var req updateManagerProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	result, err := controller_admin.UpdateManagerProfile(controller_admin.UpdateManagerProfileReq{
		Username:    req.Username,
		ActorID:     c.GetUint("admin_id"),
		ManagerID:   managerID,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "admin_user", &result.ID, result.Username, result.DisplayName)
	middleware.SetAdminAccessAction(c, "admin_user.update_profile", "修改管理员账号资料")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("修改管理员账号资料；username_before=%s；username_after=%s；nickname_before=%s；nickname_after=%s", result.PreviousUsername, result.Username, result.PreviousDisplayName, result.DisplayName))
	response.OK(c, result)
}

// BanManager 禁用管理员（仅超级管理员）。
func BanManager(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}
	managerID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	result, err := controller_admin.BanManager(c.GetUint("admin_id"), managerID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "admin_user", &managerID, result.Username, result.Username)
	middleware.SetAdminAccessAction(c, "admin_user.ban", "禁用管理员")
	middleware.SetAdminAccessChangeSet(c, `{"status":{"before":"active","after":"banned"}}`)
	response.OK(c, result)
}

// UnbanManager 启用管理员（仅超级管理员）。
func UnbanManager(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}
	managerID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	result, err := controller_admin.UnbanManager(c.GetUint("admin_id"), managerID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "admin_user", &managerID, result.Username, result.Username)
	middleware.SetAdminAccessAction(c, "admin_user.unban", "启用管理员")
	middleware.SetAdminAccessChangeSet(c, `{"status":{"before":"banned","after":"active"}}`)
	response.OK(c, result)
}

// DeleteManager 删除管理员（仅超级管理员）。
func DeleteManager(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}
	managerID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	if err := controller_admin.DeleteManager(c.GetUint("admin_id"), managerID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "admin_user", &managerID, "", "")
	middleware.SetAdminAccessAction(c, "admin_user.delete", "删除管理员")
	middleware.SetAdminAccessActionDetail(c, "删除管理员账号")
	response.OK(c, nil)
}

// ResetManagerPassword 重置管理员密码（仅超级管理员）。
func ResetManagerPassword(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}
	managerID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	var req resetManagerPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	if err := controller_admin.ResetManagerPassword(controller_admin.ResetManagerPasswordReq{
		ActorID:     c.GetUint("admin_id"),
		ManagerID:   managerID,
		NewPassword: req.Password,
	}); err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "admin_user", &managerID, "", "")
	middleware.SetAdminAccessAction(c, "admin_user.reset_password", "重置管理员密码")
	middleware.SetAdminAccessActionDetail(c, "重置管理员密码")
	response.OK(c, nil)
}

// ResetManagerTOTP 重置管理员 2FA（仅超级管理员）。
func ResetManagerTOTP(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}
	managerID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	if err := controller_admin.ResetManagerTOTP(c.GetUint("admin_id"), managerID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "admin_user", &managerID, "", "")
	middleware.SetAdminAccessAction(c, "admin_user.reset_totp", "重置管理员2FA")
	middleware.SetAdminAccessActionDetail(c, "重置管理员2FA")
	response.OK(c, nil)
}

// ResolveManagerLoginCooldown 解除管理员后台登录冷却（仅超级管理员）。
func ResolveManagerLoginCooldown(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}
	managerID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	if err := controller_admin.ResolveManagerLoginCooldown(c.GetUint("admin_id"), managerID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "admin_user", &managerID, "", "")
	middleware.SetAdminAccessAction(c, "admin_user.resolve_login_cooldown", "解除管理员登录冷却")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("解除管理员登录冷却；admin_id=%d", managerID))
	response.OK(c, nil)
}

// GetManagerIPWhitelist 获取单个管理员的 IP 白名单配置（仅超级管理员）。
func GetManagerIPWhitelist(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}
	managerID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	result, err := controller_admin.GetManagerIPWhitelist(c.GetUint("admin_id"), managerID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// SaveManagerIPWhitelist 保存单个管理员的 IP 白名单配置（仅超级管理员）。
func SaveManagerIPWhitelist(c *gin.Context) {
	if !controller_admin.IsSuperAdminRole(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员可操作")
		return
	}
	managerID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	var req saveManagerIPWhitelistReq
	if err = c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	rules := make([]controller_admin.SaveAdminIPWhitelistRuleReq, 0, len(req.Rules))
	for _, rule := range req.Rules {
		rules = append(rules, controller_admin.SaveAdminIPWhitelistRuleReq{
			CIDR: rule.CIDR,
			Note: rule.Note,
		})
	}

	result, err := controller_admin.SaveManagerIPWhitelist(controller_admin.SaveManagerIPWhitelistReq{
		ActorID:   c.GetUint("admin_id"),
		ManagerID: managerID,
		Rules:     rules,
		UpdatedBy: c.GetUint("admin_id"),
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "admin_user", &managerID, "", "")
	middleware.SetAdminAccessAction(c, "admin_user.save_ip_whitelist", "保存管理员白名单")
	middleware.SetAdminAccessActionDetail(c, "保存管理员 IP 白名单")
	middleware.SetAdminAccessChangeSet(c, `{"rule_count":{"before":"未知","after":"已更新"}}`)
	response.OK(c, result)
}
