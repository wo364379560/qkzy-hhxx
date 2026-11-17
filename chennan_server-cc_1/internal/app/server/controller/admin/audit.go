package controller_admin

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

// AdminAccessAuditResult 是后台访问审计响应。
type AdminAccessAuditResult struct {
	ID                uint   `json:"id"`
	AdminUserID       *uint  `json:"adminUserId"`
	AdminUsername     string `json:"adminUsername"`
	AdminRole         string `json:"adminRole"`
	AccessType        string `json:"accessType"`
	Module            string `json:"module"`
	ActionLabel       string `json:"actionLabel"`
	ActionDetail      string `json:"actionDetail"`
	ActionKey         string `json:"actionKey"`
	ActionSummary     string `json:"actionSummary"`
	Method            string `json:"method"`
	Route             string `json:"route"`
	TargetID          *uint  `json:"targetId"`
	TargetType        string `json:"targetType"`
	TargetUsername    string `json:"targetUsername"`
	TargetDisplayName string `json:"targetDisplayName"`
	ChangeSet         string `json:"changeSet"`
	Success           bool   `json:"success"`
	StatusCode        int    `json:"statusCode"`
	IPAddress         string `json:"ipAddress"`
	UserAgent         string `json:"userAgent"`
	FailureReason     string `json:"failureReason"`
	CreatedAt         string `json:"createdAt"`
}

// AdminLoginAuditResult 是后台登录审计响应。
type AdminLoginAuditResult struct {
	ID            uint   `json:"id"`
	AdminUserID   *uint  `json:"adminUserId"`
	Username      string `json:"username"`
	AdminRole     string `json:"adminRole"`
	Stage         string `json:"stage"`
	Result        string `json:"result"`
	FailureReason string `json:"failureReason"`
	IPAddress     string `json:"ipAddress"`
	UserAgent     string `json:"userAgent"`
	CreatedAt     string `json:"createdAt"`
}

// ListAdminAccessAuditsReq 是后台访问审计查询参数。
type ListAdminAccessAuditsReq struct {
	Page           int
	PageSize       int
	Keyword        string
	Role           string
	AccessType     string
	Method         string
	TargetType     string
	ActionKey      string
	TargetKeyword  string
	ScopeAdminIDs  []uint
	ScopeUsernames []string
}

// ListAdminLoginAuditsReq 是后台登录审计查询参数。
type ListAdminLoginAuditsReq struct {
	Page           int
	PageSize       int
	Keyword        string
	Role           string
	Stage          string
	Result         string
	ScopeAdminIDs  []uint
	ScopeUsernames []string
}

const (
	// AuditCleanupScopeAccess 表示仅清理访问审计。
	AuditCleanupScopeAccess = "access"
	// AuditCleanupScopeLogin 表示仅清理登录审计。
	AuditCleanupScopeLogin = "login"
	// AuditCleanupScopeAll 表示同时清理访问审计与登录审计。
	AuditCleanupScopeAll = "all"
)

// AuditCleanupReq 是审计日志清理请求参数。
type AuditCleanupReq struct {
	Scope         string
	BeforeTime    string
	Keyword       string
	Role          string
	AccessType    string
	Method        string
	TargetType    string
	ActionKey     string
	TargetKeyword string
	Stage         string
	Result        string
}

// AuditCleanupPreviewResult 是审计日志清理预览结果。
type AuditCleanupPreviewResult struct {
	Scope       string `json:"scope"`
	BeforeTime  string `json:"beforeTime"`
	AccessCount int64  `json:"accessCount"`
	LoginCount  int64  `json:"loginCount"`
	TotalCount  int64  `json:"totalCount"`
}

// AuditCleanupResult 是审计日志清理执行结果。
type AuditCleanupResult struct {
	Scope         string `json:"scope"`
	BeforeTime    string `json:"beforeTime"`
	AccessDeleted int64  `json:"accessDeleted"`
	LoginDeleted  int64  `json:"loginDeleted"`
	TotalDeleted  int64  `json:"totalDeleted"`
}

// RecordAdminAccessAudit 写入后台访问审计。
func RecordAdminAccessAudit(audit *model.AdminAccessAudit) error {
	if strings.TrimSpace(audit.ChangeSetJSON) == "" {
		audit.ChangeSetJSON = "{}"
	}
	repo := repository.NewAdminAccessAuditRepository()
	if err := repo.Create(audit); err != nil {
		if isLegacyAdminAccessAuditSchemaError(err) {
			return repo.CreateLegacy(audit)
		}
		return err
	}
	return nil
}

// RecordAdminLoginAudit 写入后台登录审计。
func RecordAdminLoginAudit(audit *model.AdminLoginAudit) error {
	return repository.NewAdminLoginAuditRepository().Create(audit)
}

// RecordAdminAccessAuditAction 写入结构化后台访问审计。
func RecordAdminAccessAuditAction(
	audit *model.AdminAccessAudit,
	targetType, targetUsername, targetDisplayName, actionKey, actionSummary string,
	changeSet any,
) error {
	audit.TargetType = targetType
	audit.TargetUsername = targetUsername
	audit.TargetDisplayName = targetDisplayName
	audit.ActionKey = actionKey
	audit.ActionSummary = actionSummary
	if changeSet != nil {
		if b, err := json.Marshal(changeSet); err == nil {
			audit.ChangeSetJSON = string(b)
		}
	}
	return RecordAdminAccessAudit(audit)
}

func isLegacyAdminAccessAuditSchemaError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unknown column") || strings.Contains(msg, "doesn't have a default value")
}

// UpdateAdminLoginAuditOutcome 更新后台登录审计结果。
func UpdateAdminLoginAuditOutcome(id uint, stage, result, failureReason string) error {
	return repository.NewAdminLoginAuditRepository().UpdateOutcome(id, stage, result, failureReason)
}

// ListAdminAccessAudits 分页查询后台访问审计。
func ListAdminAccessAudits(req ListAdminAccessAuditsReq) (PageResult[AdminAccessAuditResult], error) {
	rows, total, err := repository.NewAdminAccessAuditRepository().ListPaged(
		req.Page,
		req.PageSize,
		req.Keyword,
		req.Role,
		req.AccessType,
		req.Method,
		req.TargetType,
		req.ActionKey,
		req.TargetKeyword,
		req.ScopeAdminIDs,
		req.ScopeUsernames,
	)
	if err != nil {
		return PageResult[AdminAccessAuditResult]{}, err
	}

	list := make([]AdminAccessAuditResult, 0, len(rows))
	for _, row := range rows {
		module, actionLabel := mapAdminAuditAction(row.Method, row.Route, row.AccessType)
		list = append(list, AdminAccessAuditResult{
			ID:                row.ID,
			AdminUserID:       row.AdminUserID,
			AdminUsername:     row.AdminUsername,
			AdminRole:         row.AdminRole,
			AccessType:        row.AccessType,
			Module:            module,
			ActionLabel:       actionLabel,
			ActionDetail:      row.ActionDetail,
			ActionKey:         row.ActionKey,
			ActionSummary:     row.ActionSummary,
			Method:            row.Method,
			Route:             row.Route,
			TargetID:          row.TargetID,
			TargetType:        row.TargetType,
			TargetUsername:    row.TargetUsername,
			TargetDisplayName: row.TargetDisplayName,
			ChangeSet:         row.ChangeSetJSON,
			Success:           row.Success,
			StatusCode:        row.StatusCode,
			IPAddress:         row.IPAddress,
			UserAgent:         row.UserAgent,
			FailureReason:     row.FailureReason,
			CreatedAt:         row.CreatedAt.Format(time.DateTime),
		})
	}

	return PageResult[AdminAccessAuditResult]{List: list, Total: total}, nil
}

func mapAdminAuditAction(method, route, accessType string) (string, string) {
	if accessType == "illegal_access" {
		return mapAdminAuditModule(route), "非法访问"
	}

	key := method + " " + route
	switch key {
	case "POST /api/v1/admin/change-password":
		return "安全", "修改后台密码"
	case "POST /api/v1/admin/audits/cleanup/preview":
		return "审计", "预览审计日志清理"
	case "POST /api/v1/admin/audits/cleanup":
		return "审计", "清理审计日志"
	case "POST /api/v1/admin/admins":
		return "管理员", "创建管理员"
	case "POST /api/v1/admin/admins/:id/delete":
		return "管理员", "删除管理员"
	case "POST /api/v1/admin/admins/:id/ban":
		return "管理员", "禁用管理员"
	case "POST /api/v1/admin/admins/:id/unban":
		return "管理员", "启用管理员"
	case "POST /api/v1/admin/admins/:id/reset-password":
		return "管理员", "重置管理员密码"
	case "POST /api/v1/admin/admins/:id/reset-totp":
		return "管理员", "重置管理员2FA"
	case "POST /api/v1/admin/admins/:id/login-cooldown/resolve":
		return "管理员", "解除管理员登录冷却"
	case "POST /api/v1/admin/admins/:id/chat-quiet-hours":
		return "静默时段", "保存管理员定时策略"
	case "POST /api/v1/admin/admins/:id/chat-quiet-hours/delete":
		return "静默时段", "清除管理员定时策略"
	case "POST /api/v1/admin/settings/2fa":
		return "安全", "修改全局2FA"
	case "POST /api/v1/admin/settings/chat-quiet-hours":
		return "安全", "修改静默时段管控"
	case "POST /api/v1/admin/settings/sms-bypass":
		return "安全", "修改万能验证码"
	case "POST /api/v1/admin/settings/sms-bypass/reveal":
		return "安全", "查看万能验证码"
	case "POST /api/v1/admin/settings/backend-domain-pool":
		return "安全", "修改后台域名池"
	case "POST /api/v1/admin/totp/regenerate-backup-codes":
		return "安全", "重置备份码"
	case "POST /api/v1/admin/users":
		return "用户", "创建用户"
	case "POST /api/v1/admin/users/batch":
		return "用户", "批量创建用户"
	case "POST /api/v1/admin/users/export":
		return "用户", "导出用户资料"
	case "POST /api/v1/admin/users/generate":
		return "用户", "批量模板生成用户"
	case "POST /api/v1/admin/users/sync/confirm":
		return "用户", "确认同步用户"
	case "POST /api/v1/admin/users/:id/ban":
		return "用户", "封禁用户"
	case "POST /api/v1/admin/users/:id/unban":
		return "用户", "解封用户"
	case "POST /api/v1/admin/users/:id/kick":
		return "用户", "踢出用户"
	case "POST /api/v1/admin/users/:id/delete":
		return "用户", "注销用户"
	case "POST /api/v1/admin/users/:id/update":
		return "用户", "更新用户资料"
	case "POST /api/v1/admin/users/:id/reset-password":
		return "用户", "重置用户密码"
	case "POST /api/v1/admin/users/:id/update-phone":
		return "用户", "更新用户手机号"
	case "POST /api/v1/admin/users/:id/ip-whitelist":
		return "用户", "保存登录白名单"
	case "POST /api/v1/admin/users/:id/login-cooldown/resolve":
		return "用户", "解除用户登录冷却"
	case "POST /api/v1/admin/users/:id/transfer":
		return "用户", "转移用户归属"
	case "POST /api/v1/admin/users/batch-ban":
		return "用户", "批量封禁用户"
	case "POST /api/v1/admin/users/batch-unban":
		return "用户", "批量解封用户"
	case "POST /api/v1/admin/users/batch-transfer":
		return "用户", "批量转移用户归属"
	case "POST /api/v1/admin/users/batch-global-search":
		return "用户", "批量设置全局搜索"
	case "POST /api/v1/admin/users/batch-ip-whitelist":
		return "用户", "批量保存登录白名单"
	case "POST /api/v1/admin/invite-codes":
		return "邀请码", "创建邀请码"
	case "POST /api/v1/admin/invite-codes/batch":
		return "邀请码", "批量创建邀请码"
	case "POST /api/v1/admin/invite-codes/:id/update":
		return "邀请码", "更新邀请码"
	case "POST /api/v1/admin/invite-codes/:id/regenerate":
		return "邀请码", "重置邀请码"
	case "POST /api/v1/admin/invite-codes/:id/delete":
		return "邀请码", "删除邀请码"
	case "POST /api/v1/admin/groups":
		return "群组", "创建群组"
	case "POST /api/v1/admin/groups/sync/confirm":
		return "群组", "确认同步群组"
	case "POST /api/v1/admin/groups/:id/members":
		return "群组", "添加群成员"
	case "POST /api/v1/admin/groups/:id/members/remove":
		return "群组", "移除群成员"
	case "POST /api/v1/admin/groups/:id/mute":
		return "群组", "设置群静默"
	case "POST /api/v1/admin/groups/batch-mute":
		return "群组", "批量静默群组"
	case "POST /api/v1/admin/groups/batch-unmute":
		return "群组", "批量取消静默"
	case "POST /api/v1/admin/groups/:id/repair":
		return "群组", "修复群组"
	case "POST /api/v1/admin/groups/:id/delete":
		return "群组", "解散群组"
	case "POST /api/v1/admin/reports/:id/review":
		return "举报", "处理举报"
	case "POST /api/v1/admin/reports/:id/dismiss":
		return "举报", "忽略举报"
	case "POST /api/v1/admin/community/posts/:id/update":
		return "省钱社区", "修改内容"
	case "POST /api/v1/admin/community/posts/:id/delete":
		return "省钱社区", "删除内容"
	case "POST /api/v1/admin/community/comments/:id/delete":
		return "省钱社区", "删除评论"
	case "POST /api/v1/admin/community/users/:id/restrictions":
		return "省钱社区", "修改用户省钱社区权限"
	case "POST /api/v1/admin/offline-activities":
		return "线下活动", "创建活动"
	case "POST /api/v1/admin/offline-activities/:id/update":
		return "线下活动", "更新活动"
	case "POST /api/v1/admin/offline-activities/:id/close":
		return "线下活动", "关闭活动"
	case "POST /api/v1/admin/im/c2c/send":
		return "IM工具", "发送C2C消息"
	case "POST /api/v1/admin/im/c2c/withdraw":
		return "IM工具", "撤回C2C消息"
	case "POST /api/v1/admin/im/conversations/delete":
		return "IM工具", "删除会话"
	case "POST /api/v1/admin/assistants":
		return "助理", "创建助理"
	case "POST /api/v1/admin/assistants/:id/delete":
		return "助理", "删除助理"
	case "POST /api/v1/admin/assistants/:id/invite-code":
		return "助理", "生成助理邀请码"
	case "POST /api/v1/admin/assistants/:id/ban":
		return "助理", "禁用助理"
	case "POST /api/v1/admin/assistants/:id/unban":
		return "助理", "启用助理"
	case "POST /api/v1/admin/assistants/:id/reset-password":
		return "助理", "重置助理密码"
	case "POST /api/v1/admin/assistants/:id/reset-totp":
		return "助理", "重置助理2FA"
	case "POST /api/v1/admin/assistants/:id/login-cooldown/resolve":
		return "助理", "解除助理登录冷却"
	case "POST /api/v1/admin/assistants/:id/chat-quiet-hours":
		return "静默时段", "保存助理定时策略"
	case "POST /api/v1/admin/assistants/:id/chat-quiet-hours/delete":
		return "静默时段", "清除助理定时策略"
	case "POST /api/v1/admin/assistants/:id/ip-whitelist":
		return "白名单策略", "保存白名单规则"
	case "POST /api/v1/admin/admins/:id/ip-whitelist":
		return "白名单策略", "保存管理员白名单规则"
	case "POST /api/v1/admin/ip-whitelists/:id":
		return "白名单策略", "保存白名单规则"
	default:
		return mapAdminAuditModule(route), "后台操作"
	}
}

func mapAdminAuditModule(route string) string {
	switch {
	case route == "":
		return "其他"
	case strings.Contains(route, "/audits"):
		return "审计"
	case route == "/api/v1/admin/change-password" || strings.Contains(route, "/api/v1/admin/settings/") || strings.Contains(route, "/totp/"):
		return "安全"
	case strings.Contains(route, "/community/"):
		return "省钱社区"
	case strings.Contains(route, "/users"):
		return "用户"
	case strings.Contains(route, "/invite-codes"):
		return "邀请码"
	case strings.Contains(route, "/groups"):
		return "群组"
	case strings.Contains(route, "/reports"):
		return "举报"
	case strings.Contains(route, "/offline-activities"):
		return "线下活动"
	case strings.Contains(route, "/im/"):
		return "IM工具"
	case strings.Contains(route, "/assistants"):
		return "助理"
	case strings.Contains(route, "/ip-whitelists"):
		return "白名单策略"
	default:
		return "其他"
	}
}

// ListAdminLoginAudits 分页查询后台登录审计。
func ListAdminLoginAudits(req ListAdminLoginAuditsReq) (PageResult[AdminLoginAuditResult], error) {
	rows, total, err := repository.NewAdminLoginAuditRepository().ListPaged(
		req.Page,
		req.PageSize,
		req.Keyword,
		req.Role,
		req.Stage,
		req.Result,
		req.ScopeAdminIDs,
		req.ScopeUsernames,
	)
	if err != nil {
		return PageResult[AdminLoginAuditResult]{}, err
	}

	list := make([]AdminLoginAuditResult, 0, len(rows))
	for _, row := range rows {
		list = append(list, AdminLoginAuditResult{
			ID:            row.ID,
			AdminUserID:   row.AdminUserID,
			Username:      row.Username,
			AdminRole:     row.AdminRole,
			Stage:         row.Stage,
			Result:        row.Result,
			FailureReason: row.FailureReason,
			IPAddress:     row.IPAddress,
			UserAgent:     row.UserAgent,
			CreatedAt:     row.CreatedAt.Format(time.DateTime),
		})
	}

	return PageResult[AdminLoginAuditResult]{List: list, Total: total}, nil
}

// PreviewAdminAuditCleanup 预览清理指定时间点之前的审计日志数量。
func PreviewAdminAuditCleanup(req AuditCleanupReq) (*AuditCleanupPreviewResult, error) {
	scope, before, err := normalizeAuditCleanupReq(req)
	if err != nil {
		return nil, err
	}

	accessRepo := repository.NewAdminAccessAuditRepository()
	loginRepo := repository.NewAdminLoginAuditRepository()

	result := &AuditCleanupPreviewResult{
		Scope:      scope,
		BeforeTime: before.Format(time.DateTime),
	}

	switch scope {
	case AuditCleanupScopeAccess:
		result.AccessCount, err = accessRepo.CountBefore(
			before,
			req.Keyword,
			req.Role,
			req.AccessType,
			req.Method,
			req.TargetType,
			req.ActionKey,
			req.TargetKeyword,
		)
	case AuditCleanupScopeLogin:
		result.LoginCount, err = loginRepo.CountBefore(
			before,
			req.Keyword,
			req.Role,
			req.Stage,
			req.Result,
		)
	case AuditCleanupScopeAll:
		if result.AccessCount, err = accessRepo.CountBefore(
			before,
			req.Keyword,
			req.Role,
			req.AccessType,
			req.Method,
			req.TargetType,
			req.ActionKey,
			req.TargetKeyword,
		); err != nil {
			return nil, err
		}
		result.LoginCount, err = loginRepo.CountBefore(
			before,
			req.Keyword,
			req.Role,
			req.Stage,
			req.Result,
		)
	default:
		return nil, errors.New("无效的清理范围")
	}
	if err != nil {
		return nil, err
	}
	result.TotalCount = result.AccessCount + result.LoginCount
	return result, nil
}

// CleanupAdminAudits 清理指定时间点之前的审计日志。
func CleanupAdminAudits(req AuditCleanupReq) (*AuditCleanupResult, error) {
	scope, before, err := normalizeAuditCleanupReq(req)
	if err != nil {
		return nil, err
	}

	result := &AuditCleanupResult{
		Scope:      scope,
		BeforeTime: before.Format(time.DateTime),
	}

	db := datasource.Datasource.Mysql.GetClient()
	err = db.Transaction(func(tx *gorm.DB) error {
		accessRepo := repository.NewAdminAccessAuditRepositoryWithDB(tx)
		loginRepo := repository.NewAdminLoginAuditRepositoryWithDB(tx)

		switch scope {
		case AuditCleanupScopeAccess:
			result.AccessDeleted, err = accessRepo.DeleteBefore(
				before,
				req.Keyword,
				req.Role,
				req.AccessType,
				req.Method,
				req.TargetType,
				req.ActionKey,
				req.TargetKeyword,
			)
		case AuditCleanupScopeLogin:
			result.LoginDeleted, err = loginRepo.DeleteBefore(
				before,
				req.Keyword,
				req.Role,
				req.Stage,
				req.Result,
			)
		case AuditCleanupScopeAll:
			if result.AccessDeleted, err = accessRepo.DeleteBefore(
				before,
				req.Keyword,
				req.Role,
				req.AccessType,
				req.Method,
				req.TargetType,
				req.ActionKey,
				req.TargetKeyword,
			); err != nil {
				return err
			}
			result.LoginDeleted, err = loginRepo.DeleteBefore(
				before,
				req.Keyword,
				req.Role,
				req.Stage,
				req.Result,
			)
		default:
			return errors.New("无效的清理范围")
		}

		return err
	})
	if err != nil {
		return nil, err
	}

	result.TotalDeleted = result.AccessDeleted + result.LoginDeleted
	return result, nil
}

func normalizeAuditCleanupReq(req AuditCleanupReq) (string, time.Time, error) {
	scope := strings.TrimSpace(req.Scope)
	if scope == "" {
		scope = AuditCleanupScopeAccess
	}
	if scope != AuditCleanupScopeAccess && scope != AuditCleanupScopeLogin && scope != AuditCleanupScopeAll {
		return "", time.Time{}, errors.New("无效的清理范围")
	}

	before, err := parseAuditCleanupBeforeTime(req.BeforeTime)
	if err != nil {
		return "", time.Time{}, err
	}
	return scope, before, nil
}

func parseAuditCleanupBeforeTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, errors.New("清理时间不能为空")
	}

	layouts := []string{
		time.DateTime,
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02",
	}

	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			if parsed.After(time.Now()) {
				return time.Time{}, errors.New("清理时间不能晚于当前时间")
			}
			return parsed, nil
		}
	}

	return time.Time{}, errors.New("清理时间格式无效")
}
