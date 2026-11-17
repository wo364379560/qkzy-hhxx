package controller_admin

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/pkg/authwork"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
)

// AssistantResult 助理信息响应。
type AssistantResult struct {
	ID                  uint    `json:"id"`
	Username            string  `json:"username"`
	DisplayName         string  `json:"displayName"`
	PreviousUsername    string  `json:"-"`
	PreviousDisplayName string  `json:"-"`
	Role                string  `json:"role"`
	Status              string  `json:"status"`
	ParentAdminID       *uint   `json:"parentAdminID"`
	ParentAdminName     string  `json:"parentAdminName"`
	ParentAdminRole     string  `json:"parentAdminRole"`
	CreatedAt           string  `json:"createdAt"`
	TOTPEnabled         bool    `json:"totpEnabled"`
	TOTPBoundAt         *string `json:"totpBoundAt"`
	IPWhitelistEnabled  bool    `json:"ipWhitelistEnabled"`
	LoginCooldown       bool    `json:"loginCooldown"`
	LoginCooldownUntil  string  `json:"loginCooldownUntil"`
	LoginCooldownReason string  `json:"loginCooldownReason"`

	// 邀请码信息（无邀请码时各字段为零值）
	InviteCodeID        *uint   `json:"inviteCodeID"`
	InviteCode          *string `json:"inviteCode"`
	InviteCodeStatus    *string `json:"inviteCodeStatus"`
	InviteCodeUsed      int     `json:"inviteCodeUsed"`
	InviteCodeMax       int     `json:"inviteCodeMax"`
	InviteCodeExpiredAt *string `json:"inviteCodeExpiredAt"`

	// 用户统计
	UserCount int64 `json:"userCount"`
}

func toAssistantResult(a *model.AdminUser) AssistantResult {
	var boundAt *string
	if a.TOTPBoundAt != nil {
		formatted := a.TOTPBoundAt.Format("2006-01-02 15:04:05")
		boundAt = &formatted
	}

	return AssistantResult{
		ID:                 a.ID,
		Username:           a.Username,
		DisplayName:        a.DisplayName,
		Role:               a.Role,
		Status:             a.Status,
		ParentAdminID:      a.ParentAdminID,
		CreatedAt:          a.CreatedAt.Format("2006-01-02 15:04:05"),
		TOTPEnabled:        a.TOTPEnabled,
		TOTPBoundAt:        boundAt,
		IPWhitelistEnabled: a.IPWhitelistEnabled,
	}
}

// ListAssistantsReq 查询参数。
type ListAssistantsReq struct {
	ActorID   uint
	ActorRole string
	Page      int
	PageSize  int
}

// ListAssistantIPWhitelistOverviewReq 查询白名单策略概览参数。
type ListAssistantIPWhitelistOverviewReq struct {
	ActorID   uint
	ActorRole string
	Page      int
	PageSize  int
	Keyword   string
	Status    string
	Enabled   *bool
}

// AssistantIPWhitelistOverviewResult 是助理白名单策略概览响应。
type AssistantIPWhitelistOverviewResult struct {
	ID                 uint     `json:"id"`
	Username           string   `json:"username"`
	Status             string   `json:"status"`
	UserCount          int64    `json:"userCount"`
	IPWhitelistEnabled bool     `json:"ipWhitelistEnabled"`
	RuleCount          int      `json:"ruleCount"`
	RulePreview        []string `json:"rulePreview"`
	LastUpdatedAt      *string  `json:"lastUpdatedAt"`
}

// AssistantBanResult 是禁用/启用助理后的影响结果。
type AssistantBanResult struct {
	AssistantID       uint   `json:"assistantId"`
	Username          string `json:"username"`
	Status            string `json:"status"`
	AffectedUserCount int    `json:"affectedUserCount"`
}

// ListAssistants 分页列出助理（仅管理员调用）。
func ListAssistants(req ListAssistantsReq) (PageResult[AssistantResult], error) {
	actor, err := BuildActorContext(req.ActorID)
	if err != nil {
		return PageResult[AssistantResult]{}, err
	}
	if !HasManagerPrivileges(actor.Role) {
		return PageResult[AssistantResult]{}, errors.New("仅超级管理员或管理员可操作")
	}

	adminRepo := repository.NewAdminUserRepository()
	var admins []model.AdminUser
	var total int64
	if IsSuperAdminRole(actor.Role) {
		admins, total, err = adminRepo.ListAssistantsPaged(req.Page, req.PageSize)
	} else {
		admins, total, err = adminRepo.ListAssistantsByParentPaged(actor.AdminID, req.Page, req.PageSize)
	}
	if err != nil {
		return PageResult[AssistantResult]{}, err
	}

	list := make([]AssistantResult, 0, len(admins))
	if len(admins) == 0 {
		return PageResult[AssistantResult]{List: list, Total: total}, nil
	}

	// 收集助理 ID
	ids := make([]uint, len(admins))
	usernames := make([]string, 0, len(admins))
	parentIDs := make([]uint, 0, len(admins))
	seenParentIDs := make(map[uint]bool, len(admins))
	for i, a := range admins {
		ids[i] = a.ID
		usernames = append(usernames, a.Username)
		if a.ParentAdminID != nil && !seenParentIDs[*a.ParentAdminID] {
			parentIDs = append(parentIDs, *a.ParentAdminID)
			seenParentIDs[*a.ParentAdminID] = true
		}
	}

	// 批量获取每个助理最新的邀请码
	icRepo := repository.NewInviteCodeRepository()
	codes, err := icRepo.FindLatestByOwners(ids)
	if err != nil {
		return PageResult[AssistantResult]{}, err
	}
	codeMap := make(map[uint]*model.InviteCode, len(codes))
	for i := range codes {
		codeMap[codes[i].OwnerID] = &codes[i]
	}

	// 批量统计每个助理的用户总数
	userRepo := repository.NewUserRepository()
	userCounts, err := userRepo.CountByOwners(ids)
	if err != nil {
		return PageResult[AssistantResult]{}, err
	}

	parentAdmins, err := adminRepo.FindByIDs(parentIDs)
	if err != nil {
		return PageResult[AssistantResult]{}, err
	}
	parentAdminMap := make(map[uint]*model.AdminUser, len(parentAdmins))
	for i := range parentAdmins {
		parentAdminMap[parentAdmins[i].ID] = &parentAdmins[i]
	}

	cooldownMap, err := repository.NewLoginRiskControlRepository().ActiveAdminCooldownsByUsernames(usernames, time.Now())
	if err != nil {
		return PageResult[AssistantResult]{}, err
	}

	for i := range admins {
		r := toAssistantResult(&admins[i])
		if ic, ok := codeMap[admins[i].ID]; ok {
			r.InviteCodeID = &ic.ID
			r.InviteCode = &ic.Code
			r.InviteCodeStatus = &ic.Status
			r.InviteCodeUsed = ic.UsedCount
			r.InviteCodeMax = ic.MaxUses
			if ic.ExpiredAt != nil {
				s := ic.ExpiredAt.Format("2006-01-02")
				r.InviteCodeExpiredAt = &s
			}
		}
		r.UserCount = userCounts[admins[i].ID]
		if admins[i].ParentAdminID != nil {
			if parentAdmin := parentAdminMap[*admins[i].ParentAdminID]; parentAdmin != nil {
				r.ParentAdminName = parentAdmin.Username
				r.ParentAdminRole = parentAdmin.Role
			}
		}
		if cooldown, ok := cooldownMap[strings.TrimSpace(admins[i].Username)]; ok {
			r.LoginCooldown = true
			if cooldown.CooldownUntil != nil {
				r.LoginCooldownUntil = cooldown.CooldownUntil.Format("2006-01-02 15:04:05")
			}
			r.LoginCooldownReason = strings.TrimSpace(cooldown.Reason)
		}
		list = append(list, r)
	}

	return PageResult[AssistantResult]{List: list, Total: total}, nil
}

// ListAssistantIPWhitelistOverview 分页列出助理白名单策略概览（仅管理员）。
func ListAssistantIPWhitelistOverview(req ListAssistantIPWhitelistOverviewReq) (PageResult[AssistantIPWhitelistOverviewResult], error) {
	actor, err := BuildActorContext(req.ActorID)
	if err != nil {
		return PageResult[AssistantIPWhitelistOverviewResult]{}, err
	}
	if !HasManagerPrivileges(actor.Role) {
		return PageResult[AssistantIPWhitelistOverviewResult]{}, errors.New("仅超级管理员或管理员可操作")
	}

	adminRepo := repository.NewAdminUserRepository()
	var admins []model.AdminUser
	var total int64
	if IsSuperAdminRole(actor.Role) {
		admins, total, err = adminRepo.ListAssistantsFilteredPaged(req.Page, req.PageSize, req.Keyword, req.Status, req.Enabled)
	} else {
		admins, total, err = adminRepo.ListAssistantsByParentFilteredPaged(actor.AdminID, req.Page, req.PageSize, req.Keyword, req.Status, req.Enabled)
	}
	if err != nil {
		return PageResult[AssistantIPWhitelistOverviewResult]{}, err
	}

	list := make([]AssistantIPWhitelistOverviewResult, 0, len(admins))
	if len(admins) == 0 {
		return PageResult[AssistantIPWhitelistOverviewResult]{List: list, Total: total}, nil
	}

	ids := make([]uint, len(admins))
	for i, a := range admins {
		ids[i] = a.ID
	}

	userCounts, err := repository.NewUserRepository().CountByOwners(ids)
	if err != nil {
		return PageResult[AssistantIPWhitelistOverviewResult]{}, err
	}

	rules, err := repository.NewAdminUserIPWhitelistRepository().ListByAdminUserIDs(ids)
	if err != nil {
		return PageResult[AssistantIPWhitelistOverviewResult]{}, err
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
		list = append(list, AssistantIPWhitelistOverviewResult{
			ID:                 admin.ID,
			Username:           admin.Username,
			Status:             admin.Status,
			UserCount:          userCounts[admin.ID],
			IPWhitelistEnabled: admin.IPWhitelistEnabled,
			RuleCount:          ruleCountMap[admin.ID],
			RulePreview:        rulePreviewMap[admin.ID],
			LastUpdatedAt:      lastUpdatedAt,
		})
	}

	return PageResult[AssistantIPWhitelistOverviewResult]{List: list, Total: total}, nil
}

// CreateAssistantReq 创建助理参数。
type CreateAssistantReq struct {
	Username    string
	Password    string
	DisplayName string
	CreatedBy   uint
}

// UpdateAssistantProfileReq 修改助理资料参数。
type UpdateAssistantProfileReq struct {
	Username    *string
	ActorID     uint
	AssistantID uint
	DisplayName string
}

// CreateAssistant 创建助理账号。
func CreateAssistant(req CreateAssistantReq) (*AssistantResult, error) {
	actor, err := BuildActorContext(req.CreatedBy)
	if err != nil {
		return nil, err
	}
	if !HasManagerPrivileges(actor.Role) {
		return nil, errors.New("仅超级管理员或管理员可操作")
	}
	username, err := normalizeAdminLoginName(req.Username)
	if err != nil {
		return nil, err
	}
	req.Username = username
	displayName, err := normalizeAdminDisplayName(req.DisplayName)
	if err != nil {
		return nil, err
	}

	repo := repository.NewAdminUserRepository()

	if _, err := repo.FindByUsername(req.Username); err == nil {
		return nil, errors.New("用户名已存在")
	}

	hash, err := authwork.GeneratePrivateFromPassword([]byte(req.Password), authwork.DefaultCost)
	if err != nil {
		return nil, err
	}

	admin := &model.AdminUser{
		Username:      req.Username,
		DisplayName:   displayName,
		PasswordHash:  string(hash),
		Role:          RoleAssistant,
		ParentAdminID: &actor.AdminID,
		CreatedBy:     req.CreatedBy,
		Status:        "active",
		CreatedAt:     time.Now(),
	}
	if err = repo.Create(admin); err != nil {
		return nil, err
	}
	result := toAssistantResult(admin)
	return &result, nil
}

// UpdateAssistantProfile 修改助理登录账号和昵称。
func UpdateAssistantProfile(req UpdateAssistantProfileReq) (*AssistantResult, error) {
	displayName, err := normalizeAdminDisplayName(req.DisplayName)
	if err != nil {
		return nil, err
	}
	assistant, err := findManagedAssistant(req.ActorID, req.AssistantID)
	if err != nil {
		return nil, err
	}
	updated, err := updateManagedAdminProfile(assistant, req.Username, displayName)
	if err != nil {
		return nil, err
	}
	result := toAssistantResult(updated)
	result.PreviousDisplayName = assistant.DisplayName
	result.PreviousUsername = assistant.Username
	return &result, nil
}

func normalizeAdminDisplayName(displayName string) (string, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return "", errors.New("昵称不能为空")
	}
	if len([]rune(displayName)) > 64 {
		return "", errors.New("昵称最多 64 个字符")
	}
	return displayName, nil
}

// DeleteAssistant 删除助理账号（硬删除）。
func DeleteAssistant(actorID, assistantID uint) error {
	assistant, err := findManagedAssistant(actorID, assistantID)
	if err != nil {
		return err
	}

	deps, err := loadAdminDeletionDependencies(assistantID)
	if err != nil {
		return err
	}
	if deps.UserCount > 0 || deps.GroupCount > 0 || deps.InviteCodeCount > 0 {
		return errors.New(buildAdminDeleteBlockedMessage("助理", deps))
	}

	db := datasource.Datasource.Mysql.GetClient()
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("admin_user_id = ?", assistantID).Delete(&model.AdminChatQuietHoursPolicy{}).Error; err != nil {
			return err
		}
		if err := tx.Where("admin_user_id = ?", assistantID).Delete(&model.AdminUserIPWhitelist{}).Error; err != nil {
			return err
		}
		if err := tx.Where("admin_id = ?", assistantID).Delete(&model.AdminTOTPBackupCode{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.AdminUser{}, assistant.ID).Error
	})
}

// GenerateAssistantInviteCode 为助理生成邀请码（管理员操作，ownerID = assistantID）。
// 每位助理有且仅有一个有效邀请码，若已存在 active 邀请码则拒绝创建。
func GenerateAssistantInviteCode(actorID, assistantID uint) (*InviteCodeResult, error) {
	if _, err := findManagedAssistant(actorID, assistantID); err != nil {
		return nil, err
	}
	codeRepo := repository.NewInviteCodeRepository()
	if _, err := codeRepo.FindActiveByOwner(assistantID); err == nil {
		return nil, errors.New("该助理已有有效邀请码，请先修改或停用现有邀请码")
	}
	return CreateInviteCode(CreateInviteCodeReq{
		OwnerID: assistantID,
		MaxUses: 0, // 0 表示不限次数
	})
}

// GetAssistantMe 获取当前助理自己的详情（含邀请码信息）。
func GetAssistantMe(assistantID uint) (*AssistantResult, error) {
	adminRepo := repository.NewAdminUserRepository()
	admin, err := adminRepo.FindByID(assistantID)
	if err != nil {
		return nil, errors.New("助理不存在")
	}

	r := toAssistantResult(admin)

	icRepo := repository.NewInviteCodeRepository()
	codes, err := icRepo.FindLatestByOwners([]uint{assistantID})
	if err == nil && len(codes) > 0 {
		ic := codes[0]
		r.InviteCodeID = &ic.ID
		r.InviteCode = &ic.Code
		r.InviteCodeStatus = &ic.Status
		r.InviteCodeUsed = ic.UsedCount
		r.InviteCodeMax = ic.MaxUses
		if ic.ExpiredAt != nil {
			s := ic.ExpiredAt.Format("2006-01-02")
			r.InviteCodeExpiredAt = &s
		}
	}

	userRepo := repository.NewUserRepository()
	counts, _ := userRepo.CountByOwners([]uint{assistantID})
	r.UserCount = counts[assistantID]

	return &r, nil
}

// BanAssistant 禁用助理账号。
func BanAssistant(actorID, assistantID uint) (*AssistantBanResult, error) {
	repo := repository.NewAdminUserRepository()
	admin, err := findManagedAssistant(actorID, assistantID)
	if err != nil {
		return nil, err
	}
	userRepo := repository.NewUserRepository()
	users, err := userRepo.ListByOwner(assistantID)
	if err != nil {
		return nil, err
	}
	if admin.Status == "banned" {
		return &AssistantBanResult{
			AssistantID:       admin.ID,
			Username:          admin.Username,
			Status:            "banned",
			AffectedUserCount: len(users),
		}, nil
	}
	if err = repo.UpdateStatus(assistantID, "banned"); err != nil {
		return nil, err
	}

	// 尽力将名下用户踢下线，避免旧会话继续使用。
	imClient := tencentim.NewClient()
	if imClient.IsConfigured() {
		for _, user := range users {
			if user.IMUserID == "" {
				continue
			}
			_ = imClient.AccountKick(user.IMUserID)
			_ = imClient.AccountBan(user.IMUserID)
		}
	}

	return &AssistantBanResult{
		AssistantID:       admin.ID,
		Username:          admin.Username,
		Status:            "banned",
		AffectedUserCount: len(users),
	}, nil
}

// UnbanAssistant 启用助理账号。
func UnbanAssistant(actorID, assistantID uint) (*AssistantBanResult, error) {
	repo := repository.NewAdminUserRepository()
	admin, err := findManagedAssistant(actorID, assistantID)
	if err != nil {
		return nil, err
	}
	userRepo := repository.NewUserRepository()
	users, err := userRepo.ListByOwner(assistantID)
	if err != nil {
		return nil, err
	}
	if admin.Status == "active" {
		return &AssistantBanResult{
			AssistantID:       admin.ID,
			Username:          admin.Username,
			Status:            "active",
			AffectedUserCount: len(users),
		}, nil
	}
	if err = repo.UpdateStatus(assistantID, "active"); err != nil {
		return nil, err
	}
	imClient := tencentim.NewClient()
	if imClient.IsConfigured() {
		for _, user := range users {
			if user.IMUserID == "" || user.Status != "active" {
				continue
			}
			_ = imClient.AccountUnban(user.IMUserID)
		}
	}

	return &AssistantBanResult{
		AssistantID:       admin.ID,
		Username:          admin.Username,
		Status:            "active",
		AffectedUserCount: len(users),
	}, nil
}

// ResetAssistantPasswordReq 重置密码参数。
type ResetAssistantPasswordReq struct {
	ActorID     uint
	AssistantID uint
	NewPassword string
}

// ResetAssistantPassword 重置助理密码。
func ResetAssistantPassword(req ResetAssistantPasswordReq) error {
	repo := repository.NewAdminUserRepository()
	if _, err := findManagedAssistant(req.ActorID, req.AssistantID); err != nil {
		return err
	}
	hash, err := authwork.GeneratePrivateFromPassword([]byte(req.NewPassword), authwork.DefaultCost)
	if err != nil {
		return err
	}
	return repo.UpdatePassword(req.AssistantID, string(hash))
}

// ResolveAssistantLoginCooldown 解除助理后台登录冷却。
func ResolveAssistantLoginCooldown(actorID, assistantID uint) error {
	assistant, err := findManagedAssistant(actorID, assistantID)
	if err != nil {
		return err
	}
	ok, err := repository.NewLoginRiskControlRepository().ResolveAdminCooldownByUsername(
		assistant.Username,
		actorID,
		"助理列表解除登录冷却",
		time.Now(),
	)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("当前助理未处于登录冷却")
	}
	return nil
}
