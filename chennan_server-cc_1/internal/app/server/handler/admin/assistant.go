package handler_admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/handler/authlimit"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

func setAssistantAuditChangeSet(c *gin.Context, changeSet map[string]any) {
	if len(changeSet) == 0 {
		return
	}
	if payload, err := json.Marshal(changeSet); err == nil {
		middleware.SetAdminAccessChangeSet(c, string(payload))
	}
}

// GetAssistantMe 获取当前助理自己的详情（含邀请码）。
func GetAssistantMe(c *gin.Context) {
	if c.GetString("admin_role") != "assistant" {
		response.Forbidden(c, "仅助理可访问")
		return
	}
	adminID := c.GetUint("admin_id")
	result, err := controller_admin.GetAssistantMe(adminID)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// ListAssistants 获取助理列表（仅管理员）。
func ListAssistants(c *gin.Context) {
	role := c.GetString("admin_role")
	if !controller_admin.HasManagerPrivileges(role) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}
	page, pageSize := parsePagination(c)
	result, err := controller_admin.ListAssistants(controller_admin.ListAssistantsReq{
		ActorID:   c.GetUint("admin_id"),
		ActorRole: role,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// ListAssistantIPWhitelistOverview 获取助理白名单策略概览（仅管理员）。
func ListAssistantIPWhitelistOverview(c *gin.Context) {
	role := c.GetString("admin_role")
	if !controller_admin.HasManagerPrivileges(role) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}

	page, pageSize := parsePagination(c)
	var enabled *bool
	switch c.Query("enabled") {
	case "true":
		value := true
		enabled = &value
	case "false":
		value := false
		enabled = &value
	default:
		value := true
		enabled = &value
	}

	result, err := controller_admin.ListAssistantIPWhitelistOverview(controller_admin.ListAssistantIPWhitelistOverviewReq{
		ActorID:   c.GetUint("admin_id"),
		ActorRole: role,
		Page:      page,
		PageSize:  pageSize,
		Keyword:   c.Query("keyword"),
		Status:    c.Query("status"),
		Enabled:   enabled,
	})
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// ListIPWhitelistOverview 获取后台账号白名单策略概览（仅管理员）。
func ListIPWhitelistOverview(c *gin.Context) {
	role := c.GetString("admin_role")
	if !controller_admin.HasManagerPrivileges(role) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}

	page, pageSize := parsePagination(c)
	var enabled *bool
	switch c.Query("enabled") {
	case "true":
		value := true
		enabled = &value
	case "false":
		value := false
		enabled = &value
	default:
		value := true
		enabled = &value
	}

	result, err := controller_admin.ListIPWhitelistOverview(controller_admin.ListIPWhitelistOverviewReq{
		ActorID:   c.GetUint("admin_id"),
		ActorRole: role,
		Page:      page,
		PageSize:  pageSize,
		Keyword:   c.Query("keyword"),
		Status:    c.Query("status"),
		Enabled:   enabled,
	})
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

type createAssistantReq struct {
	Username    string `json:"username" binding:"required,min=3,max=32"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name"`
}

func (req *createAssistantReq) normalize() error {
	req.Username = strings.TrimSpace(req.Username)
	if n := utf8.RuneCountInString(req.Username); n < 3 || n > 32 {
		return errors.New("用户名需为 3–32 个字符")
	}
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.DisplayName == "" {
		// 兼容仅有用户名和初始密码的旧版后台页面。
		req.DisplayName = req.Username
	}
	if utf8.RuneCountInString(req.DisplayName) > 64 {
		return errors.New("昵称最多 64 个字符")
	}
	return nil
}

func assistantCreateValidationMessage(err error) string {
	var fields validator.ValidationErrors
	if errors.As(err, &fields) {
		for _, field := range fields {
			switch field.Field() {
			case "Username":
				return "请填写 3–32 个字符的用户名"
			case "Password":
				return "请填写至少 8 位的初始密码"
			}
		}
	}
	return "请求格式不正确，请检查用户名和密码"
}

type updateAssistantProfileReq struct {
	Username    *string `json:"username"`
	DisplayName string  `json:"display_name" binding:"required"`
}

// CreateAssistant 创建助理（仅管理员）。
func CreateAssistant(c *gin.Context) {
	createAssistant(c, controller_admin.CreateAssistant)
}

func createAssistant(c *gin.Context, create func(controller_admin.CreateAssistantReq) (*controller_admin.AssistantResult, error)) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}
	var req createAssistantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, assistantCreateValidationMessage(err))
		return
	}
	if err := req.normalize(); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	result, err := create(controller_admin.CreateAssistantReq{
		Username:    req.Username,
		Password:    req.Password,
		DisplayName: req.DisplayName,
		CreatedBy:   adminID,
	})
	if err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "assistant", &result.ID, result.Username, result.DisplayName)
	middleware.SetAdminAccessAction(c, "assistant.create", "创建助理")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("创建助理账号；username=%s；display_name=%s", result.Username, result.DisplayName))
	response.OK(c, result)
}

// UpdateAssistantProfile 修改助理昵称。
func UpdateAssistantProfile(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}
	assistantID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	var req updateAssistantProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	result, err := controller_admin.UpdateAssistantProfile(controller_admin.UpdateAssistantProfileReq{
		Username:    req.Username,
		ActorID:     c.GetUint("admin_id"),
		AssistantID: assistantID,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "assistant", &result.ID, result.Username, result.DisplayName)
	middleware.SetAdminAccessAction(c, "assistant.update_profile", "修改助理账号资料")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("修改助理账号资料；username_before=%s；username_after=%s；nickname_before=%s；nickname_after=%s", result.PreviousUsername, result.Username, result.PreviousDisplayName, result.DisplayName))
	response.OK(c, result)
}

// DeleteAssistant 删除助理（仅管理员）。
func DeleteAssistant(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}
	assistantID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	if err = controller_admin.DeleteAssistant(c.GetUint("admin_id"), assistantID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "assistant", &assistantID, "", "")
	middleware.SetAdminAccessAction(c, "assistant.delete", "删除助理")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("删除助理账号；assistant_id=%d", assistantID))
	response.OK(c, nil)
}

// GenerateAssistantInviteCode 为助理生成邀请码（仅管理员）。
func GenerateAssistantInviteCode(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}
	assistantID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	result, err := controller_admin.GenerateAssistantInviteCode(c.GetUint("admin_id"), assistantID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "assistant", &assistantID, "", "助理邀请码")
	middleware.SetAdminAccessAction(c, "assistant.generate_invite_code", "生成助理邀请码")
	setAssistantAuditChangeSet(c, map[string]any{
		"invite_code": map[string]any{
			"before": "old",
			"after":  "new",
		},
	})
	response.OK(c, result)
}

// BanAssistant 禁用助理账号（仅管理员）。
func BanAssistant(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}
	assistantID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	result, err := controller_admin.BanAssistant(c.GetUint("admin_id"), assistantID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "assistant", &assistantID, "", "")
	middleware.SetAdminAccessAction(c, "assistant.ban", "禁用助理")
	middleware.SetAdminAccessActionDetail(
		c,
		fmt.Sprintf(
			"禁用助理账号；assistant_id=%d；affected_user_count=%d",
			assistantID,
			result.AffectedUserCount,
		),
	)
	setAssistantAuditChangeSet(c, map[string]any{
		"status": map[string]any{
			"before": "active",
			"after":  "banned",
		},
	})
	response.OK(c, result)
}

// UnbanAssistant 解禁助理账号（仅管理员）。
func UnbanAssistant(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}
	assistantID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	result, err := controller_admin.UnbanAssistant(c.GetUint("admin_id"), assistantID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "assistant", &assistantID, "", "")
	middleware.SetAdminAccessAction(c, "assistant.unban", "启用助理")
	middleware.SetAdminAccessActionDetail(
		c,
		fmt.Sprintf(
			"启用助理账号；assistant_id=%d；affected_user_count=%d",
			assistantID,
			result.AffectedUserCount,
		),
	)
	setAssistantAuditChangeSet(c, map[string]any{
		"status": map[string]any{
			"before": "banned",
			"after":  "active",
		},
	})
	response.OK(c, result)
}

type resetAssistantPasswordReq struct {
	Password string `json:"password" binding:"required,min=8"`
}

type saveAssistantIPWhitelistReq struct {
	Rules []saveAssistantIPWhitelistRule `json:"rules"`
}

type saveAssistantIPWhitelistRule struct {
	CIDR string `json:"cidr" binding:"required"`
	Note string `json:"note"`
}

// ResetAssistantPassword 重置助理密码（仅管理员）。
func ResetAssistantPassword(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}
	assistantID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	var req resetAssistantPasswordReq
	if err = c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	if err = controller_admin.ResetAssistantPassword(controller_admin.ResetAssistantPasswordReq{
		ActorID:     c.GetUint("admin_id"),
		AssistantID: assistantID,
		NewPassword: req.Password,
	}); err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "assistant", &assistantID, "", "")
	middleware.SetAdminAccessAction(c, "assistant.reset_password", "重置助理密码")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("重置助理密码；assistant_id=%d", assistantID))
	response.OK(c, nil)
}

// ResolveAssistantLoginCooldown 解除助理后台登录冷却（仅管理员）。
func ResolveAssistantLoginCooldown(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}
	assistantID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	if err = controller_admin.ResolveAssistantLoginCooldown(c.GetUint("admin_id"), assistantID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "assistant", &assistantID, "", "")
	middleware.SetAdminAccessAction(c, "assistant.resolve_login_cooldown", "解除助理登录冷却")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("解除助理登录冷却；assistant_id=%d", assistantID))
	response.OK(c, nil)
}

// GetAssistantIPWhitelist 获取单个助理的 IP 白名单配置（仅管理员）。
func GetAssistantIPWhitelist(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}
	assistantID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	result, err := controller_admin.GetAssistantIPWhitelist(c.GetUint("admin_id"), assistantID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// SaveAssistantIPWhitelist 保存单个助理的 IP 白名单配置（仅管理员）。
func SaveAssistantIPWhitelist(c *gin.Context) {
	if !controller_admin.HasManagerPrivileges(c.GetString("admin_role")) {
		response.Forbidden(c, "仅超级管理员或管理员可操作")
		return
	}
	assistantID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}

	var req saveAssistantIPWhitelistReq
	if err = c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	rules := make([]controller_admin.SaveAssistantIPWhitelistRuleReq, 0, len(req.Rules))
	for _, rule := range req.Rules {
		rules = append(rules, controller_admin.SaveAssistantIPWhitelistRuleReq{
			CIDR: rule.CIDR,
			Note: rule.Note,
		})
	}

	result, err := controller_admin.SaveAssistantIPWhitelist(controller_admin.SaveAssistantIPWhitelistReq{
		ActorID:     c.GetUint("admin_id"),
		AssistantID: assistantID,
		Rules:       rules,
		UpdatedBy:   c.GetUint("admin_id"),
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "assistant", &assistantID, "", "")
	middleware.SetAdminAccessAction(c, "assistant.save_ip_whitelist", "保存助理白名单")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("保存助理白名单；assistant_id=%d；rule_count=%d", assistantID, len(rules)))
	setAssistantAuditChangeSet(c, map[string]any{
		"rule_count": map[string]any{
			"before": "未知",
			"after":  len(rules),
		},
	})
	response.OK(c, result)
}
