package controller_admin

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/authwork"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
)

// ManagedAdminResult 是管理员账号信息响应。
type ManagedAdminResult struct {
	ID                  uint    `json:"id"`
	Username            string  `json:"username"`
	DisplayName         string  `json:"displayName"`
	PreviousUsername    string  `json:"-"`
	PreviousDisplayName string  `json:"-"`
	Role                string  `json:"role"`
	Status              string  `json:"status"`
	AssistantCount      int64   `json:"assistantCount"`
	CreatedAt           string  `json:"createdAt"`
	TOTPEnabled         bool    `json:"totpEnabled"`
	TOTPBoundAt         *string `json:"totpBoundAt"`
	IPWhitelistEnabled  bool    `json:"ipWhitelistEnabled"`
	LoginCooldown       bool    `json:"loginCooldown"`
	LoginCooldownUntil  string  `json:"loginCooldownUntil"`
	LoginCooldownReason string  `json:"loginCooldownReason"`
}

func toManagedAdminResult(admin *model.AdminUser) ManagedAdminResult {
	var boundAt *string
	if admin.TOTPBoundAt != nil {
		formatted := admin.TOTPBoundAt.Format("2006-01-02 15:04:05")
		boundAt = &formatted
	}

	return ManagedAdminResult{
		ID:                 admin.ID,
		Username:           admin.Username,
		DisplayName:        admin.DisplayName,
		Role:               admin.Role,
		Status:             admin.Status,
		CreatedAt:          admin.CreatedAt.Format("2006-01-02 15:04:05"),
		TOTPEnabled:        admin.TOTPEnabled,
		TOTPBoundAt:        boundAt,
		IPWhitelistEnabled: admin.IPWhitelistEnabled,
	}
}

func applyAdminUserLoginCooldown(result *ManagedAdminResult, cooldown model.LoginRiskControl) {
	result.LoginCooldown = true
	if cooldown.CooldownUntil != nil {
		result.LoginCooldownUntil = cooldown.CooldownUntil.Format("2006-01-02 15:04:05")
	}
	result.LoginCooldownReason = strings.TrimSpace(cooldown.Reason)
}

// ListManagersReq 查询管理员列表参数。
type ListManagersReq struct {
	ActorID  uint
	Page     int
	PageSize int
}

// CreateManagerReq 创建管理员参数。
type CreateManagerReq struct {
	Username    string
	Password    string
	DisplayName string
	CreatedBy   uint
}

// UpdateManagerProfileReq 修改管理员资料参数。
type UpdateManagerProfileReq struct {
	Username    *string
	ActorID     uint
	ManagerID   uint
	DisplayName string
}

// ResetManagerPasswordReq 重置管理员密码参数。
type ResetManagerPasswordReq struct {
	ActorID     uint
	ManagerID   uint
	NewPassword string
}

// AdminStatusChangeResult 是管理员启用/禁用结果。
type AdminStatusChangeResult struct {
	AdminID  uint   `json:"adminId"`
	Username string `json:"username"`
	Status   string `json:"status"`
}

type adminDeletionDependencies struct {
	AssistantCount  int64
	UserCount       int64
	GroupCount      int64
	InviteCodeCount int64
}

// ListManagers 分页列出管理员账号（仅超级管理员）。
func ListManagers(req ListManagersReq) (PageResult[ManagedAdminResult], error) {
	actor, err := BuildActorContext(req.ActorID)
	if err != nil {
		return PageResult[ManagedAdminResult]{}, err
	}
	if !IsSuperAdminRole(actor.Role) {
		return PageResult[ManagedAdminResult]{}, errors.New("仅超级管理员可操作")
	}

	repo := repository.NewAdminUserRepository()
	admins, total, err := repo.ListManagersPaged(req.Page, req.PageSize)
	if err != nil {
		return PageResult[ManagedAdminResult]{}, err
	}

	list := make([]ManagedAdminResult, 0, len(admins))
	managerIDs := make([]uint, 0, len(admins))
	usernames := make([]string, 0, len(admins))
	for _, admin := range admins {
		managerIDs = append(managerIDs, admin.ID)
		usernames = append(usernames, admin.Username)
	}
	assistantCountMap, err := repo.CountAssistantsByParents(managerIDs)
	if err != nil {
		return PageResult[ManagedAdminResult]{}, err
	}
	cooldownMap, err := repository.NewLoginRiskControlRepository().ActiveAdminCooldownsByUsernames(usernames, time.Now())
	if err != nil {
		return PageResult[ManagedAdminResult]{}, err
	}
	for i := range admins {
		item := toManagedAdminResult(&admins[i])
		item.AssistantCount = assistantCountMap[admins[i].ID]
		if cooldown, ok := cooldownMap[strings.TrimSpace(admins[i].Username)]; ok {
			applyAdminUserLoginCooldown(&item, cooldown)
		}
		list = append(list, item)
	}
	return PageResult[ManagedAdminResult]{List: list, Total: total}, nil
}

// CreateManager 创建管理员账号（仅超级管理员）。
func CreateManager(req CreateManagerReq) (*ManagedAdminResult, error) {
	actor, err := BuildActorContext(req.CreatedBy)
	if err != nil {
		return nil, err
	}
	if !IsSuperAdminRole(actor.Role) {
		return nil, errors.New("仅超级管理员可操作")
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

	parentID := actor.AdminID
	admin := &model.AdminUser{
		Username:      req.Username,
		DisplayName:   displayName,
		PasswordHash:  string(hash),
		Role:          RoleAdmin,
		ParentAdminID: &parentID,
		CreatedBy:     req.CreatedBy,
		Status:        "active",
		CreatedAt:     time.Now(),
	}
	if err := repo.Create(admin); err != nil {
		return nil, err
	}

	result := toManagedAdminResult(admin)
	return &result, nil
}

// UpdateManagerProfile 修改管理员登录账号和昵称（仅超级管理员）。
func UpdateManagerProfile(req UpdateManagerProfileReq) (*ManagedAdminResult, error) {
	displayName, err := normalizeAdminDisplayName(req.DisplayName)
	if err != nil {
		return nil, err
	}
	admin, err := findManagedManager(req.ActorID, req.ManagerID)
	if err != nil {
		return nil, err
	}
	updated, err := updateManagedAdminProfile(admin, req.Username, displayName)
	if err != nil {
		return nil, err
	}
	result := toManagedAdminResult(updated)
	result.PreviousDisplayName = admin.DisplayName
	result.PreviousUsername = admin.Username
	return &result, nil
}

// BanManager 禁用管理员账号（仅超级管理员）。
func BanManager(actorID, managerID uint) (*AdminStatusChangeResult, error) {
	repo := repository.NewAdminUserRepository()
	admin, err := findManagedManager(actorID, managerID)
	if err != nil {
		return nil, err
	}
	if admin.Status == "banned" {
		return &AdminStatusChangeResult{
			AdminID:  admin.ID,
			Username: admin.Username,
			Status:   admin.Status,
		}, nil
	}
	if err := repo.UpdateStatus(managerID, "banned"); err != nil {
		return nil, err
	}
	return &AdminStatusChangeResult{
		AdminID:  admin.ID,
		Username: admin.Username,
		Status:   "banned",
	}, nil
}

// UnbanManager 启用管理员账号（仅超级管理员）。
func UnbanManager(actorID, managerID uint) (*AdminStatusChangeResult, error) {
	repo := repository.NewAdminUserRepository()
	admin, err := findManagedManager(actorID, managerID)
	if err != nil {
		return nil, err
	}
	if admin.Status == "active" {
		return &AdminStatusChangeResult{
			AdminID:  admin.ID,
			Username: admin.Username,
			Status:   admin.Status,
		}, nil
	}
	if err := repo.UpdateStatus(managerID, "active"); err != nil {
		return nil, err
	}
	return &AdminStatusChangeResult{
		AdminID:  admin.ID,
		Username: admin.Username,
		Status:   "active",
	}, nil
}

// DeleteManager 删除管理员账号（仅超级管理员）。
func DeleteManager(actorID, managerID uint) error {
	admin, err := findManagedManager(actorID, managerID)
	if err != nil {
		return err
	}

	deps, err := loadAdminDeletionDependencies(managerID)
	if err != nil {
		return err
	}
	if deps.AssistantCount > 0 || deps.UserCount > 0 || deps.GroupCount > 0 || deps.InviteCodeCount > 0 {
		return errors.New(buildAdminDeleteBlockedMessage("管理员", deps))
	}

	db := datasource.Datasource.Mysql.GetClient()
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("admin_user_id = ?", managerID).Delete(&model.AdminChatQuietHoursPolicy{}).Error; err != nil {
			return err
		}
		if err := tx.Where("admin_user_id = ?", managerID).Delete(&model.AdminUserIPWhitelist{}).Error; err != nil {
			return err
		}
		if err := tx.Where("admin_id = ?", managerID).Delete(&model.AdminTOTPBackupCode{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.AdminUser{}, admin.ID).Error
	})
}

// ResetManagerPassword 重置管理员密码（仅超级管理员）。
func ResetManagerPassword(req ResetManagerPasswordReq) error {
	repo := repository.NewAdminUserRepository()
	if _, err := findManagedManager(req.ActorID, req.ManagerID); err != nil {
		return err
	}
	hash, err := authwork.GeneratePrivateFromPassword([]byte(req.NewPassword), authwork.DefaultCost)
	if err != nil {
		return err
	}
	return repo.UpdatePassword(req.ManagerID, string(hash))
}

// ResetManagerTOTP 重置管理员 2FA（仅超级管理员）。
func ResetManagerTOTP(actorID, managerID uint) error {
	if _, err := findManagedManager(actorID, managerID); err != nil {
		return err
	}

	db := datasource.Datasource.Mysql.GetClient()
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.AdminUser{}).
			Where("id = ?", managerID).
			Updates(map[string]any{
				"totp_secret":   "",
				"totp_enabled":  false,
				"totp_bound_at": nil,
				"auth_version":  gorm.Expr("GREATEST(COALESCE(auth_version, 1), 1) + 1"),
			}).Error; err != nil {
			return err
		}

		return tx.Where("admin_id = ?", managerID).
			Delete(&model.AdminTOTPBackupCode{}).Error
	})
}

// ResolveManagerLoginCooldown 解除管理员后台登录冷却（仅超级管理员）。
func ResolveManagerLoginCooldown(actorID, managerID uint) error {
	admin, err := findManagedManager(actorID, managerID)
	if err != nil {
		return err
	}
	ok, err := repository.NewLoginRiskControlRepository().ResolveAdminCooldownByUsername(
		admin.Username,
		actorID,
		"管理员列表解除登录冷却",
		time.Now(),
	)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("当前管理员未处于登录冷却")
	}
	return nil
}

func loadAdminDeletionDependencies(ownerID uint) (adminDeletionDependencies, error) {
	adminRepo := repository.NewAdminUserRepository()
	userRepo := repository.NewUserRepository()
	groupRepo := repository.NewGroupRepository()
	inviteCodeRepo := repository.NewInviteCodeRepository()

	assistantCountMap, err := adminRepo.CountAssistantsByParents([]uint{ownerID})
	if err != nil {
		return adminDeletionDependencies{}, err
	}
	userCount, err := userRepo.CountByOwner(ownerID)
	if err != nil {
		return adminDeletionDependencies{}, err
	}
	groupCount, err := groupRepo.CountActiveByOwners([]uint{ownerID})
	if err != nil {
		return adminDeletionDependencies{}, err
	}
	inviteCodeCount, err := inviteCodeRepo.CountByOwner(ownerID)
	if err != nil {
		return adminDeletionDependencies{}, err
	}

	return adminDeletionDependencies{
		AssistantCount:  assistantCountMap[ownerID],
		UserCount:       userCount,
		GroupCount:      groupCount,
		InviteCodeCount: inviteCodeCount,
	}, nil
}

func buildAdminDeleteBlockedMessage(subject string, deps adminDeletionDependencies) string {
	parts := make([]string, 0, 4)
	if deps.AssistantCount > 0 {
		parts = append(parts, "助理")
	}
	if deps.UserCount > 0 {
		parts = append(parts, "用户")
	}
	if deps.GroupCount > 0 {
		parts = append(parts, "群组")
	}
	if deps.InviteCodeCount > 0 {
		parts = append(parts, "邀请码")
	}
	return subject + "名下仍有关联" + strings.Join(parts, "、") + "，请先清理后再删除"
}
