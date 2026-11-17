package handler_admin

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/handler/authlimit"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

// ListUsers 获取用户列表。
func ListUsers(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	role := c.GetString("admin_role")
	actor, err := controller_admin.BuildActorContext(adminID)
	if err != nil {
		response.InternalError(c)
		return
	}

	page, pageSize := parsePagination(c)

	// 权限范围：助理只能查自己名下；管理员只能查自己作用域；超级管理员可查全量或按归属过滤
	var ownerID uint
	if controller_admin.IsAssistantRole(role) {
		ownerID = adminID
	} else if oid := c.Query("ownerID"); oid != "" {
		if id, err := strconv.ParseUint(oid, 10, 64); err == nil {
			if controller_admin.IsSuperAdminRole(role) || controller_admin.CanManageOwnerID(actor, uint(id)) {
				ownerID = uint(id)
			}
		}
	}

	scopeOwnerIDs := []uint(nil)
	if !controller_admin.IsSuperAdminRole(role) {
		scopeOwnerIDs = actor.ScopeAdminIDs
	}

	var isActivated *bool
	var ipWhitelistEnabled *bool
	statusFilter := c.Query("status")
	if statusFilter == "unactivated" {
		// 未激活 = status=active AND is_activated=false
		statusFilter = "active"
		b := false
		isActivated = &b
	} else if v := c.Query("isActivated"); v == "true" {
		b := true
		isActivated = &b
	} else if v == "false" {
		b := false
		isActivated = &b
	}
	if v := c.Query("ipWhitelistEnabled"); v == "true" {
		b := true
		ipWhitelistEnabled = &b
	} else if v == "false" {
		b := false
		ipWhitelistEnabled = &b
	}
	currentLoginIP := strings.TrimSpace(c.Query("currentLoginIP"))
	if currentLoginIP != "" && net.ParseIP(currentLoginIP) == nil {
		response.BadRequest(c, "请输入合法 IP")
		return
	}

	result, err := controller_admin.ListUsers(controller_admin.ListUsersReq{
		Page:               page,
		PageSize:           pageSize,
		OwnerID:            ownerID,
		ScopeOwnerIDs:      scopeOwnerIDs,
		Keyword:            c.Query("keyword"),
		Status:             statusFilter,
		CurrentLoginIP:     currentLoginIP,
		IsActivated:        isActivated,
		IPWhitelistEnabled: ipWhitelistEnabled,
	})
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// ListUserOwners 获取用户管理页可用的归属人列表。
func ListUserOwners(c *gin.Context) {
	result, err := controller_admin.ListUserOwners(
		c.GetUint("admin_id"),
		controller_admin.HasManagerPrivileges(c.GetString("admin_role")),
	)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, result)
}

// ExportUsers 导出当前筛选结果的用户资料 CSV。
func ExportUsers(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	role := c.GetString("admin_role")
	actor, err := controller_admin.BuildActorContext(adminID)
	if err != nil {
		response.InternalError(c)
		return
	}

	var req exportUsersReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	req.CurrentLoginIP = strings.TrimSpace(req.CurrentLoginIP)
	if req.CurrentLoginIP != "" && net.ParseIP(req.CurrentLoginIP) == nil {
		response.BadRequest(c, "请输入合法 IP")
		return
	}

	var ownerID uint
	if controller_admin.IsAssistantRole(role) {
		ownerID = adminID
	} else if req.OwnerID > 0 {
		if controller_admin.IsSuperAdminRole(role) || controller_admin.CanManageOwnerID(actor, req.OwnerID) {
			ownerID = req.OwnerID
		}
	}

	scopeOwnerIDs := []uint(nil)
	if !controller_admin.IsSuperAdminRole(role) {
		scopeOwnerIDs = actor.ScopeAdminIDs
	}

	rows, err := controller_admin.ExportUsers(controller_admin.ExportUsersReq{
		OwnerID:            ownerID,
		ScopeOwnerIDs:      scopeOwnerIDs,
		Keyword:            req.Keyword,
		Status:             req.Status,
		CurrentLoginIP:     req.CurrentLoginIP,
		IsActivated:        req.IsActivated,
		IPWhitelistEnabled: req.IPWhitelistEnabled,
	})
	if err != nil {
		middleware.SetAdminAccessFailureReason(c, err.Error())
		response.BadRequest(c, err.Error())
		return
	}

	var buffer bytes.Buffer
	buffer.WriteString("\uFEFF")
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{
		"用户名",
		"UID",
		"无忧号",
		"手机号",
		"当前登录IP",
		"最后在线时间",
		"最后离线时间",
		"归属",
		"归属角色",
		"账号状态",
		"登录状态",
		"是否激活",
		"邀请码",
		"注册时间",
	}); err != nil {
		response.InternalError(c)
		return
	}

	for _, row := range rows {
		if err := writer.Write([]string{
			safeCSVCell(row.Nickname),
			safeCSVCell(row.IMUserID),
			safeCSVCell(row.WananID),
			safeCSVCell(row.Phone),
			safeCSVCell(row.CurrentLoginIP),
			safeCSVCell(row.LastOnlineAt),
			safeCSVCell(row.LastOfflineAt),
			safeCSVCell(row.OwnerName),
			safeCSVCell(row.OwnerRole),
			safeCSVCell(row.Status),
			safeCSVCell(row.LoginStatus),
			boolToYesNo(row.IsActivated),
			safeCSVCell(row.InviteCode),
			safeCSVCell(row.CreatedAt),
		}); err != nil {
			response.InternalError(c)
			return
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		response.InternalError(c)
		return
	}

	middleware.SetAdminAccessActionDetail(
		c,
		fmt.Sprintf(
			"导出当前筛选结果；数量:%d；关键词:%s；状态:%s；激活:%v；归属ID:%d",
			len(rows),
			req.Keyword,
			req.Status,
			req.IsActivated,
			ownerID,
		),
	)

	filename := fmt.Sprintf("users_export_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buffer.Bytes())
}

func boolToYesNo(value bool) string {
	if value {
		return "是"
	}
	return "否"
}

func safeCSVCell(value string) string {
	if value == "" {
		return ""
	}
	switch value[0] {
	case '=', '+', '-', '@':
		return "'" + value
	default:
		return value
	}
}

func setUserAuditTarget(c *gin.Context, adminID uint, role string, userID uint) {
	result, err := controller_admin.GetUser(userID, adminID, role)
	if err != nil {
		middleware.SetAdminAccessTarget(c, "user", &userID, "", "")
		return
	}

	imUserID, _ := result["im_user_id"].(string)
	nickname, _ := result["nickname"].(string)
	middleware.SetAdminAccessTarget(c, "user", &userID, imUserID, nickname)
}

func setUserAuditChangeSet(c *gin.Context, changeSet map[string]any) {
	if len(changeSet) == 0 {
		return
	}
	if payload, err := json.Marshal(changeSet); err == nil {
		middleware.SetAdminAccessChangeSet(c, string(payload))
	}
}

type createUserReq struct {
	Phone     string `json:"phone"      binding:"required,max=32"`
	Password  string `json:"password"   binding:"required,min=8"`
	Nickname  string `json:"nickname"   binding:"omitempty,max=32"`
	AvatarURL string `json:"avatar_url" binding:"omitempty,url"`
	OwnerID   uint   `json:"owner_id"`
}

type exportUsersReq struct {
	Keyword            string `json:"keyword"`
	Status             string `json:"status"`
	CurrentLoginIP     string `json:"currentLoginIP"`
	IsActivated        *bool  `json:"isActivated"`
	IPWhitelistEnabled *bool  `json:"ipWhitelistEnabled"`
	OwnerID            uint   `json:"ownerID"`
}

// CreateUser 创建单个用户。
func CreateUser(c *gin.Context) {
	var req createUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	result, err := controller_admin.CreateUser(c.Request.Context(), adminID, isAdmin, controller_admin.CreateUserReq{
		Phone:     req.Phone,
		Password:  req.Password,
		Nickname:  req.Nickname,
		AvatarURL: req.AvatarURL,
		OwnerID:   req.OwnerID,
		CreatedBy: adminID,
	})
	if err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "user", &result.ID, result.IMUserID, result.Nickname)
	middleware.SetAdminAccessAction(c, "user.create", "创建用户")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("创建用户；phone=%s；nickname=%s；owner_id=%d", req.Phone, req.Nickname, req.OwnerID))
	response.OK(c, result)
}

type batchCreateUserItem struct {
	Phone     string `json:"phone"      binding:"required,max=32"`
	Password  string `json:"password"   binding:"required,min=8"`
	Nickname  string `json:"nickname"   binding:"omitempty,max=32"`
	AvatarURL string `json:"avatar_url" binding:"omitempty,url"`
	OwnerID   uint   `json:"owner_id"`
}

type batchCreateUsersReq struct {
	Users []batchCreateUserItem `json:"users" binding:"required,min=1,max=200"`
}

type generatePhoneTemplateReq struct {
	Prefix      string `json:"prefix"`
	MiddleMode  string `json:"middle_mode"`
	MiddleStart int    `json:"middle_start"`
	Suffix      string `json:"suffix"`
}

type generateNicknameTemplateReq struct {
	Prefix  string `json:"prefix"`
	Start   int    `json:"start"`
	Padding int    `json:"padding"`
}

type generatePasswordTemplateReq struct {
	Mode         string `json:"mode"`
	FixedValue   string `json:"fixed_value"`
	Prefix       string `json:"prefix"`
	RandomLength int    `json:"random_length"`
}

type generateUsersReq struct {
	Count            int                         `json:"count" binding:"required"`
	OwnerID          uint                        `json:"owner_id"`
	RealNameVerified *bool                       `json:"real_name_verified"` // nil/true=已实名（默认），false=未实名
	CanCreateGroup   *bool                       `json:"can_create_group"`   // nil/false=不允许建群（默认），true=允许
	PhoneTemplate    generatePhoneTemplateReq    `json:"phone_template" binding:"required"`
	NicknameTemplate generateNicknameTemplateReq `json:"nickname_template" binding:"required"`
	PasswordTemplate generatePasswordTemplateReq `json:"password_template" binding:"required"`
}

// BatchCreateUsers 批量创建用户。
func BatchCreateUsers(c *gin.Context) {
	var req batchCreateUsersReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	reqs := make([]controller_admin.CreateUserReq, len(req.Users))
	for i, u := range req.Users {
		reqs[i] = controller_admin.CreateUserReq{
			Phone:     u.Phone,
			Password:  u.Password,
			Nickname:  u.Nickname,
			AvatarURL: u.AvatarURL,
			OwnerID:   u.OwnerID,
			CreatedBy: adminID,
		}
	}
	results, err := controller_admin.BatchCreateUsers(c.Request.Context(), adminID, isAdmin, reqs)
	if err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.InternalError(c)
		return
	}
	middleware.SetAdminAccessAction(c, "user.batch_create", "批量创建用户")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("批量创建用户；count=%d", len(req.Users)))
	response.OK(c, results)
}

// GenerateUsers 按模板批量生成用户。
func GenerateUsers(c *gin.Context) {
	var req generateUsersReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	result, err := controller_admin.GenerateUsers(c.Request.Context(), adminID, isAdmin, controller_admin.GenerateUsersReq{
		Count:            req.Count,
		OwnerID:          req.OwnerID,
		RealNameVerified: req.RealNameVerified,
		CanCreateGroup:   req.CanCreateGroup,
		PhoneTemplate: controller_admin.GeneratePhoneTemplateReq{
			Prefix:      req.PhoneTemplate.Prefix,
			MiddleMode:  req.PhoneTemplate.MiddleMode,
			MiddleStart: req.PhoneTemplate.MiddleStart,
			Suffix:      req.PhoneTemplate.Suffix,
		},
		NicknameTemplate: controller_admin.GenerateNicknameTemplateReq{
			Prefix:  req.NicknameTemplate.Prefix,
			Start:   req.NicknameTemplate.Start,
			Padding: req.NicknameTemplate.Padding,
		},
		PasswordTemplate: controller_admin.GeneratePasswordTemplateReq{
			Mode:         req.PasswordTemplate.Mode,
			FixedValue:   req.PasswordTemplate.FixedValue,
			Prefix:       req.PasswordTemplate.Prefix,
			RandomLength: req.PasswordTemplate.RandomLength,
		},
	})
	if err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		middleware.SetAdminAccessFailureReason(c, err.Error())
		response.BadRequest(c, err.Error())
		return
	}

	passwordModeLabel := "完全随机"
	switch req.PasswordTemplate.Mode {
	case "fixed":
		passwordModeLabel = "统一固定密码"
	case "prefix_random":
		passwordModeLabel = "固定前缀+随机后缀"
	}
	effectiveOwnerID := adminID
	if isAdmin && req.OwnerID > 0 {
		effectiveOwnerID = req.OwnerID
	}
	middleware.SetAdminAccessActionDetail(
		c,
		fmt.Sprintf(
			"数量:%d；成功:%d；失败:%d；归属:%d；自动实名:%v；允许建群:%v；手机号:%s-中4位%s-%s；昵称:%s+序号；密码:%s",
			result.Requested,
			result.Success,
			result.Failed,
			effectiveOwnerID,
			req.RealNameVerified == nil || (req.RealNameVerified != nil && *req.RealNameVerified),
			req.CanCreateGroup != nil && *req.CanCreateGroup,
			req.PhoneTemplate.Prefix,
			req.PhoneTemplate.MiddleMode,
			req.PhoneTemplate.Suffix,
			req.NicknameTemplate.Prefix,
			passwordModeLabel,
		),
	)
	response.OK(c, result)
}

// BanUser 封禁用户。
func BanUser(c *gin.Context) {
	userID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	before, _ := controller_admin.GetUser(uint(userID), adminID, c.GetString("admin_role"))
	if err = controller_admin.BanUser(c.Request.Context(), adminID, isAdmin, userID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	setUserAuditTarget(c, adminID, c.GetString("admin_role"), userID)
	middleware.SetAdminAccessAction(c, "user.ban", "封禁用户")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("封禁用户；user_id=%d", userID))
	if before != nil {
		setUserAuditChangeSet(c, map[string]any{
			"status": map[string]any{
				"before": before["status"],
				"after":  "banned",
			},
			"login_status": map[string]any{
				"before": before["login_status"],
				"after":  before["login_status"],
			},
		})
	}
	response.OK(c, nil)
}

// UnbanUser 解封用户。
func UnbanUser(c *gin.Context) {
	userID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	before, _ := controller_admin.GetUser(uint(userID), adminID, c.GetString("admin_role"))
	if err = controller_admin.UnbanUser(c.Request.Context(), adminID, isAdmin, userID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	setUserAuditTarget(c, adminID, c.GetString("admin_role"), userID)
	middleware.SetAdminAccessAction(c, "user.unban", "解封用户")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("解封用户；user_id=%d", userID))
	if before != nil {
		setUserAuditChangeSet(c, map[string]any{
			"status": map[string]any{
				"before": before["status"],
				"after":  "active",
			},
			"login_status": map[string]any{
				"before": before["login_status"],
				"after":  before["login_status"],
			},
		})
	}
	response.OK(c, nil)
}

// KickUser 踢出用户（强制下线）。
func KickUser(c *gin.Context) {
	userID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	before, _ := controller_admin.GetUser(uint(userID), adminID, c.GetString("admin_role"))
	if err = controller_admin.KickUser(c.Request.Context(), adminID, isAdmin, userID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	setUserAuditTarget(c, adminID, c.GetString("admin_role"), userID)
	middleware.SetAdminAccessAction(c, "user.kick", "踢出用户")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("踢出用户；user_id=%d", userID))
	if before != nil {
		setUserAuditChangeSet(c, map[string]any{
			"status": map[string]any{
				"before": before["status"],
				"after":  before["status"],
			},
			"login_status": map[string]any{
				"before": before["login_status"],
				"after":  "offline",
			},
		})
	}
	response.OK(c, nil)
}

// DeleteUser 注销用户。
func DeleteUser(c *gin.Context) {
	userID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	before, _ := controller_admin.GetUser(uint(userID), adminID, c.GetString("admin_role"))
	if err = controller_admin.DeleteUser(c.Request.Context(), adminID, isAdmin, userID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	setUserAuditTarget(c, adminID, c.GetString("admin_role"), userID)
	middleware.SetAdminAccessAction(c, "user.delete", "注销用户")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("软注销用户；user_id=%d；本地业务数据和腾讯 IM 账号未删除", userID))
	if before != nil {
		setUserAuditChangeSet(c, map[string]any{
			"status": map[string]any{
				"before": before["status"],
				"after":  "deleted",
			},
			"login_status": map[string]any{
				"before": before["login_status"],
				"after":  "deleted",
			},
		})
	}
	response.OK(c, nil)
}

// PreviewDeleteUser 获取注销用户前的影响预览。
func PreviewDeleteUser(c *gin.Context) {
	userID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	result, err := controller_admin.PreviewDeleteUser(c.Request.Context(), adminID, isAdmin, userID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// EnableUserTOTP 打开用户 Web 端 Google 验证。
func EnableUserTOTP(c *gin.Context) {
	userID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	user, err := controller_admin.EnableUserTOTPByAdmin(c.GetUint("admin_id"), userID)
	if err != nil {
		middleware.SetAdminAccessFailureReason(c, err.Error())
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "user", &userID, user.Phone, user.Nickname)
	middleware.SetAdminAccessAction(c, "user.enable_totp", "开启用户Google验证")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("开启用户 Web 端 Google 验证；user_id=%d", userID))
	setUserAuditChangeSet(c, map[string]any{
		"totp_enabled": map[string]any{
			"before": false,
			"after":  true,
		},
	})
	response.OK(c, nil)
}

// DisableUserTOTP 关闭用户 Web 端 Google 验证。
func DisableUserTOTP(c *gin.Context) {
	userID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	user, err := controller_admin.DisableUserTOTPByAdmin(c.GetUint("admin_id"), userID)
	if err != nil {
		middleware.SetAdminAccessFailureReason(c, err.Error())
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "user", &userID, user.Phone, user.Nickname)
	middleware.SetAdminAccessAction(c, "user.disable_totp", "关闭用户Google验证")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("关闭用户 Web 端 Google 验证，并撤销当前登录态；user_id=%d", userID))
	setUserAuditChangeSet(c, map[string]any{
		"totp_enabled": map[string]any{
			"before": true,
			"after":  false,
		},
		"refresh_sessions": map[string]any{
			"before": "active",
			"after":  "revoked",
		},
	})
	response.OK(c, nil)
}

type transferReq struct {
	TargetAdminID uint `json:"target_admin_id" binding:"required"`
}

type batchUserIDsReq struct {
	UserIDs []uint `json:"userIDs" binding:"required,min=1,max=100"`
}

type batchTransferReq struct {
	UserIDs       []uint `json:"userIDs" binding:"required,min=1,max=100"`
	TargetAdminID uint   `json:"target_admin_id" binding:"required"`
}

// TransferUser 转移用户归属。
func TransferUser(c *gin.Context) {
	userID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	var req transferReq
	if err = c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	assignment, err := controller_admin.TransferUserWithAssignment(c.Request.Context(), adminID, isAdmin, userID, req.TargetAdminID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	setUserAuditTarget(c, adminID, c.GetString("admin_role"), userID)
	middleware.SetAdminAccessAction(c, "user.transfer_owner", "转移用户归属")
	middleware.SetAdminAccessActionDetail(
		c,
		fmt.Sprintf("转移用户归属；user_id=%d；target_admin_id=%d", userID, req.TargetAdminID),
	)
	response.OK(c, gin.H{"assignment": assignment})
}

// BatchBanUsers 批量封禁用户。
func BatchBanUsers(c *gin.Context) {
	var req batchUserIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	result, err := controller_admin.BatchBanUsers(c.Request.Context(), adminID, isAdmin, req.UserIDs)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "user", nil, "", "批量用户")
	middleware.SetAdminAccessAction(c, "user.batch_ban", "批量封禁用户")
	count, summary := summarizeUintIDs(req.UserIDs)
	middleware.SetAdminAccessActionDetail(
		c,
		fmt.Sprintf("批量封禁用户；count=%d；user_ids=%s", count, summary),
	)
	setUserAuditChangeSet(c, map[string]any{
		"affected_count": map[string]any{
			"before": 0,
			"after":  count,
		},
		"status": map[string]any{
			"before": "mixed",
			"after":  "banned",
		},
	})
	response.OK(c, result)
}

// BatchUnbanUsers 批量解封用户。
func BatchUnbanUsers(c *gin.Context) {
	var req batchUserIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	result, err := controller_admin.BatchUnbanUsers(c.Request.Context(), adminID, isAdmin, req.UserIDs)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "user", nil, "", "批量用户")
	middleware.SetAdminAccessAction(c, "user.batch_unban", "批量解封用户")
	count, summary := summarizeUintIDs(req.UserIDs)
	middleware.SetAdminAccessActionDetail(
		c,
		fmt.Sprintf("批量解封用户；count=%d；user_ids=%s", count, summary),
	)
	setUserAuditChangeSet(c, map[string]any{
		"affected_count": map[string]any{
			"before": 0,
			"after":  count,
		},
		"status": map[string]any{
			"before": "mixed",
			"after":  "active",
		},
	})
	response.OK(c, result)
}

// BatchDeleteUsers 批量注销用户。
func BatchDeleteUsers(c *gin.Context) {
	var req batchUserIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	result, err := controller_admin.BatchDeleteUsers(c.Request.Context(), adminID, isAdmin, req.UserIDs)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "user", nil, "", "批量用户")
	middleware.SetAdminAccessAction(c, "user.batch_delete", "批量注销用户")
	count, summary := summarizeUintIDs(req.UserIDs)
	middleware.SetAdminAccessActionDetail(
		c,
		fmt.Sprintf("批量软注销用户；count=%d；user_ids=%s；本地业务数据和腾讯 IM 账号未删除", count, summary),
	)
	setUserAuditChangeSet(c, map[string]any{
		"affected_count": map[string]any{
			"before": 0,
			"after":  count,
		},
		"status": map[string]any{
			"before": "mixed",
			"after":  "deleted",
		},
	})
	response.OK(c, result)
}

// BatchTransferUsers 批量转移用户归属。
func BatchTransferUsers(c *gin.Context) {
	var req batchTransferReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))
	result, err := controller_admin.BatchTransferUsers(c.Request.Context(), adminID, isAdmin, req.UserIDs, req.TargetAdminID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessTarget(c, "user", nil, "", "批量用户")
	middleware.SetAdminAccessAction(c, "user.batch_transfer_owner", "批量转移用户归属")
	count, summary := summarizeUintIDs(req.UserIDs)
	middleware.SetAdminAccessActionDetail(
		c,
		fmt.Sprintf(
			"批量转移用户归属；target_admin_id=%d；count=%d；user_ids=%s",
			req.TargetAdminID,
			count,
			summary,
		),
	)
	setUserAuditChangeSet(c, map[string]any{
		"affected_count": map[string]any{
			"before": 0,
			"after":  count,
		},
		"owner_id": map[string]any{
			"before": "mixed",
			"after":  req.TargetAdminID,
		},
	})
	response.OK(c, result)
}

// parseIDParam 解析路径参数中的 uint ID。
func parseIDParam(c *gin.Context, param string) (uint, error) {
	id, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID 参数")
		c.Abort()
		return 0, err
	}
	return uint(id), nil
}

// parsePagination 解析 page / pageSize 查询参数，带默认值与最大值限制。
func parsePagination(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 20
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(c.Query("pageSize")); err == nil && ps > 0 {
		if ps > 100 {
			ps = 100
		}
		pageSize = ps
	}
	return
}

// SyncUsersPreview 从腾讯 IM 拉取用户数据预览。
func SyncUsersPreview(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	result, err := controller_admin.SyncUsersPreview(adminID, isAdmin)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

type syncUserConfirmReq struct {
	Selections []controller_admin.SyncUserSelection `json:"selections" binding:"required,min=1"`
}

// SyncUsersConfirm 确认同步选中的用户。
func SyncUsersConfirm(c *gin.Context) {
	var req syncUserConfirmReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	result, err := controller_admin.SyncUsersConfirm(adminID, isAdmin, req.Selections)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// GetUser 获取用户详情。
func GetUser(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	role := c.GetString("admin_role")
	userID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	result, err := controller_admin.GetUser(uint(userID), adminID, role)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// ListUserLoginDevices 获取用户历史登录设备（含 Web）。
func ListUserLoginDevices(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}

	result, err := controller_admin.ListUserLoginDevices(adminID, uint(userID))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// UpdateUser 更新用户基本资料。
func UpdateUser(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	role := c.GetString("admin_role")
	userID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var req controller_admin.UpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	before, _ := controller_admin.GetUser(uint(userID), adminID, role)

	err := controller_admin.UpdateUser(uint(userID), adminID, role, req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	setUserAuditTarget(c, adminID, role, uint(userID))
	middleware.SetAdminAccessAction(c, "user.update_profile", "更新用户资料")
	changeSet := map[string]any{}
	if req.Nickname != nil {
		changeSet["nickname"] = map[string]any{"before": before["nickname"], "after": *req.Nickname}
	}
	if req.WananID != nil {
		changeSet["wanan_id"] = map[string]any{"before": before["wanan_id"], "after": *req.WananID}
	}
	if req.UserTag != nil {
		changeSet["user_tag"] = map[string]any{"before": before["user_tag"], "after": *req.UserTag}
	}
	if req.AvatarURL != nil {
		changeSet["avatar_url"] = map[string]any{"before": before["avatar_url"], "after": *req.AvatarURL}
	}
	if req.SelfSignature != nil {
		changeSet["self_signature"] = map[string]any{"before": before["self_signature"], "after": *req.SelfSignature}
	}
	if req.Gender != nil {
		changeSet["gender"] = map[string]any{"before": before["gender"], "after": *req.Gender}
	}
	if req.CanCreateGroup != nil {
		changeSet["can_create_group"] = map[string]any{"before": before["can_create_group"], "after": *req.CanCreateGroup}
	}
	if req.GloballySearchable != nil {
		changeSet["globally_searchable"] = map[string]any{"before": before["globally_searchable"], "after": *req.GloballySearchable}
	}
	setUserAuditChangeSet(c, changeSet)
	response.OK(c, nil)
}

type batchSetUserCreateGroupPermissionReq struct {
	UserIDs []uint `json:"userIDs" binding:"required,min=1"`
	Enabled bool   `json:"enabled"`
}

type batchSetUserGlobalSearchReq struct {
	UserIDs []uint `json:"userIDs" binding:"required,min=1"`
	Enabled bool   `json:"enabled"`
}

type batchSaveUserIPWhitelistReq struct {
	UserIDs []uint  `json:"userIDs" binding:"required,min=1,max=100"`
	Enabled bool    `json:"enabled"`
	IP      *string `json:"ip"`
}

// BatchSetUserCreateGroupPermission 批量设置用户建群权限。
func BatchSetUserCreateGroupPermission(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	isAdmin := controller_admin.HasManagerPrivileges(c.GetString("admin_role"))

	var req batchSetUserCreateGroupPermissionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := controller_admin.BatchSetUserCreateGroupPermission(
		c.Request.Context(),
		adminID,
		isAdmin,
		req.UserIDs,
		req.Enabled,
	)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	middleware.SetAdminAccessAction(c, "user.batch_set_create_group", "批量设置建群权限")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("批量设置建群权限；count=%d；enabled=%v", len(req.UserIDs), req.Enabled))
	setUserAuditChangeSet(c, map[string]any{
		"affected_count": map[string]any{
			"before": 0,
			"after":  len(req.UserIDs),
		},
		"can_create_group": map[string]any{
			"before": "批量变更",
			"after":  req.Enabled,
		},
	})
	response.OK(c, result)
}

// BatchSetUserGlobalSearch 批量设置用户全局搜索。
func BatchSetUserGlobalSearch(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	role := c.GetString("admin_role")
	isAdmin := controller_admin.HasManagerPrivileges(role)

	var req batchSetUserGlobalSearchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := controller_admin.BatchSetUserGlobalSearch(
		c.Request.Context(),
		adminID,
		isAdmin,
		req.UserIDs,
		req.Enabled,
	)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	count, summary := summarizeUintIDs(req.UserIDs)
	middleware.SetAdminAccessTarget(c, "user", nil, "", "批量用户")
	middleware.SetAdminAccessAction(c, "user.batch_set_global_search", "批量设置全局搜索")
	middleware.SetAdminAccessActionDetail(
		c,
		fmt.Sprintf("批量设置用户全局搜索；count=%d；enabled=%v；user_ids=%s", count, req.Enabled, summary),
	)
	setUserAuditChangeSet(c, map[string]any{
		"affected_count": map[string]any{
			"before": 0,
			"after":  count,
		},
		"globally_searchable": map[string]any{
			"before": "批量变更",
			"after":  req.Enabled,
		},
	})
	response.OK(c, result)
}

// BatchSaveUserIPWhitelist 批量保存用户登录白名单。
func BatchSaveUserIPWhitelist(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	role := c.GetString("admin_role")

	var req batchSaveUserIPWhitelistReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := controller_admin.BatchSaveUserIPWhitelist(
		c.Request.Context(),
		adminID,
		role,
		req.UserIDs,
		controller_admin.SaveUserIPWhitelistReq{
			Enabled: req.Enabled,
			IP:      req.IP,
		},
	)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	count, summary := summarizeUintIDs(req.UserIDs)
	middleware.SetAdminAccessTarget(c, "user", nil, "", "批量用户")
	middleware.SetAdminAccessAction(c, "user.batch_save_ip_whitelist", "批量保存用户白名单")
	middleware.SetAdminAccessActionDetail(
		c,
		fmt.Sprintf(
			"批量保存用户登录白名单；count=%d；enabled=%v；forced_logout=%d；user_ids=%s",
			count,
			req.Enabled,
			result.ForcedLogout,
			summary,
		),
	)
	setUserAuditChangeSet(c, map[string]any{
		"affected_count": map[string]any{
			"before": 0,
			"after":  count,
		},
		"ip_whitelist_enabled": map[string]any{
			"before": "批量变更",
			"after":  req.Enabled,
		},
		"ip_whitelist_ip": map[string]any{
			"before": "已隐藏",
			"after":  "已隐藏",
		},
	})
	response.OK(c, result)
}

// ResetUserPassword 重置用户密码。
func ResetUserPassword(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	role := c.GetString("admin_role")
	userID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var req controller_admin.ResetUserPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	err := controller_admin.ResetUserPassword(uint(userID), adminID, role, req)
	if err != nil {
		if authlimit.AbortIfAuthWorkBusy(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	setUserAuditTarget(c, adminID, role, uint(userID))
	middleware.SetAdminAccessAction(c, "user.reset_password", "重置用户密码")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("重置用户密码；user_id=%d；mode=%s；must_change=%v", userID, req.Mode, req.MustChange))
	response.OK(c, nil)
}

// UpdateUserPhone 修改用户手机号。
func UpdateUserPhone(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	role := c.GetString("admin_role")
	userID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var req controller_admin.UpdateUserPhoneReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	err := controller_admin.UpdateUserPhone(uint(userID), adminID, role, req.NewPhone)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	setUserAuditTarget(c, adminID, role, uint(userID))
	middleware.SetAdminAccessAction(c, "user.update_phone", "更新用户手机号")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("更新用户手机号；user_id=%d", userID))
	setUserAuditChangeSet(c, map[string]any{
		"phone": map[string]any{
			"before": "已隐藏",
			"after":  "已隐藏",
		},
	})
	response.OK(c, nil)
}

// SaveUserIPWhitelist 保存用户登录白名单。
func SaveUserIPWhitelist(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	role := c.GetString("admin_role")
	userID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var req controller_admin.SaveUserIPWhitelistReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := controller_admin.SaveUserIPWhitelist(uint(userID), adminID, role, req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	setUserAuditTarget(c, adminID, role, uint(userID))
	middleware.SetAdminAccessAction(c, "user.save_ip_whitelist", "保存用户白名单")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("保存用户登录白名单；user_id=%d；enabled=%v", userID, req.Enabled))
	setUserAuditChangeSet(c, map[string]any{
		"ip_whitelist_enabled": map[string]any{
			"before": "未知",
			"after":  result.Enabled,
		},
		"ip_whitelist_ip": map[string]any{
			"before": "已隐藏",
			"after":  "已隐藏",
		},
	})
	response.OK(c, result)
}

// ResolveUserLoginCooldown 解除用户登录冷却。
func ResolveUserLoginCooldown(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	role := c.GetString("admin_role")
	userID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	if err := controller_admin.ResolveUserLoginCooldown(uint(userID), adminID, role); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	setUserAuditTarget(c, adminID, role, uint(userID))
	middleware.SetAdminAccessAction(c, "user.resolve_login_cooldown", "解除用户登录冷却")
	middleware.SetAdminAccessActionDetail(c, fmt.Sprintf("解除用户登录冷却；user_id=%d", userID))
	setUserAuditChangeSet(c, map[string]any{
		"login_cooldown": map[string]any{
			"before": "cooldown",
			"after":  "resolved",
		},
	})
	response.OK(c, nil)
}

// GetUserStatus 获取用户在线状态。
func GetUserStatus(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	role := c.GetString("admin_role")
	userID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	result, err := controller_admin.GetUserStatus(uint(userID), adminID, role)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}

// BatchGetUserStatus 批量获取用户在线状态。
func BatchGetUserStatus(c *gin.Context) {
	var req controller_admin.BatchGetUserStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := controller_admin.BatchGetUserStatus(req.IMUserIDs)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, result)
}
