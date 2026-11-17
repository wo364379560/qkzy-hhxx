package controller_admin

import (
	"context"
	"errors"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/quietpolicycache"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/authwork"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"gorm.io/gorm"
)

// AccountChatQuietHoursPolicyResult 表示后台账号级静默时段策略。
type AccountChatQuietHoursPolicyResult struct {
	AdminUserID uint                        `json:"adminUserId"`
	Username    string                      `json:"username"`
	Role        string                      `json:"role"`
	HasPolicy   bool                        `json:"hasPolicy"`
	Policy      config.ChatQuietHoursPolicy `json:"policy"`
}

// SaveAccountChatQuietHoursPolicyReq 表示保存后台账号级静默时段策略的参数。
type SaveAccountChatQuietHoursPolicyReq struct {
	ActorID           uint
	TargetAdminUserID uint
	CurrentPassword   string
	Policy            config.ChatQuietHoursPolicy
}

func toAccountChatQuietHoursPolicyResult(admin *model.AdminUser, policy config.ChatQuietHoursPolicy, hasPolicy bool) *AccountChatQuietHoursPolicyResult {
	return &AccountChatQuietHoursPolicyResult{
		AdminUserID: admin.ID,
		Username:    admin.Username,
		Role:        admin.Role,
		HasPolicy:   hasPolicy,
		Policy:      policy,
	}
}

// GetManagerChatQuietHoursPolicy 获取管理员账号级静默时段策略。
func GetManagerChatQuietHoursPolicy(actorID, managerID uint) (*AccountChatQuietHoursPolicyResult, error) {
	manager, err := findViewableManagerChatQuietHoursTarget(actorID, managerID)
	if err != nil {
		return nil, err
	}
	return loadAccountChatQuietHoursPolicyResult(manager)
}

// SaveManagerChatQuietHoursPolicy 保存管理员账号级静默时段策略。
func SaveManagerChatQuietHoursPolicy(req SaveAccountChatQuietHoursPolicyReq) error {
	manager, err := findEditableManagerChatQuietHoursTarget(req.ActorID, req.TargetAdminUserID, req.CurrentPassword)
	if err != nil {
		return err
	}
	return upsertAccountChatQuietHoursPolicy(manager.ID, req.Policy, req.ActorID)
}

// ClearManagerChatQuietHoursPolicy 清除管理员账号级静默时段策略。
func ClearManagerChatQuietHoursPolicy(actorID, managerID uint, currentPassword string) error {
	manager, err := findEditableManagerChatQuietHoursTarget(actorID, managerID, currentPassword)
	if err != nil {
		return err
	}
	if err := repository.NewAdminChatQuietHoursPolicyRepository().DeleteByAdminUserID(manager.ID); err != nil {
		return err
	}
	quietpolicycache.InvalidateAll(context.Background())
	return nil
}

// GetAssistantChatQuietHoursPolicy 获取助理账号级静默时段策略。
func GetAssistantChatQuietHoursPolicy(actorID, assistantID uint) (*AccountChatQuietHoursPolicyResult, error) {
	assistant, err := findViewableAssistantChatQuietHoursTarget(actorID, assistantID)
	if err != nil {
		return nil, err
	}
	return loadAccountChatQuietHoursPolicyResult(assistant)
}

// SaveAssistantChatQuietHoursPolicy 保存助理账号级静默时段策略。
func SaveAssistantChatQuietHoursPolicy(req SaveAccountChatQuietHoursPolicyReq) error {
	assistant, err := findEditableAssistantChatQuietHoursTarget(req.ActorID, req.TargetAdminUserID, req.CurrentPassword)
	if err != nil {
		return err
	}
	return upsertAccountChatQuietHoursPolicy(assistant.ID, req.Policy, req.ActorID)
}

// ClearAssistantChatQuietHoursPolicy 清除助理账号级静默时段策略。
func ClearAssistantChatQuietHoursPolicy(actorID, assistantID uint, currentPassword string) error {
	assistant, err := findEditableAssistantChatQuietHoursTarget(actorID, assistantID, currentPassword)
	if err != nil {
		return err
	}
	if err := repository.NewAdminChatQuietHoursPolicyRepository().DeleteByAdminUserID(assistant.ID); err != nil {
		return err
	}
	quietpolicycache.InvalidateAll(context.Background())
	return nil
}

func loadAccountChatQuietHoursPolicyResult(admin *model.AdminUser) (*AccountChatQuietHoursPolicyResult, error) {
	policy := config.DefaultChatQuietHoursPolicy()
	policy.Enabled = false

	repo := repository.NewAdminChatQuietHoursPolicyRepository()
	record, err := repo.FindByAdminUserID(admin.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return toAccountChatQuietHoursPolicyResult(admin, policy, false), nil
		}
		return nil, err
	}

	policy.Enabled = record.Enabled
	policy.TimeZone = record.TimeZone
	policy.Start = record.Start
	policy.End = record.End
	policy.BlockRegister = record.BlockRegister
	policy.BlockSendMessage = record.BlockSendMessage
	policy.BlockCreateGroup = record.BlockCreateGroup
	policy.BlockAddFriend = record.BlockAddFriend
	if err := config.NormalizeChatQuietHoursPolicy(&policy); err != nil {
		return nil, err
	}
	return toAccountChatQuietHoursPolicyResult(admin, policy, true), nil
}

func upsertAccountChatQuietHoursPolicy(adminUserID uint, policy config.ChatQuietHoursPolicy, updatedBy uint) error {
	if err := config.NormalizeChatQuietHoursPolicy(&policy); err != nil {
		return err
	}
	record := &model.AdminChatQuietHoursPolicy{
		AdminUserID:      adminUserID,
		Enabled:          policy.Enabled,
		TimeZone:         policy.TimeZone,
		Start:            policy.Start,
		End:              policy.End,
		BlockRegister:    policy.BlockRegister,
		BlockSendMessage: policy.BlockSendMessage,
		BlockCreateGroup: policy.BlockCreateGroup,
		BlockAddFriend:   policy.BlockAddFriend,
		UpdatedBy:        updatedBy,
	}
	if err := repository.NewAdminChatQuietHoursPolicyRepository().Upsert(record); err != nil {
		return err
	}
	quietpolicycache.InvalidateAll(context.Background())
	return nil
}

func findViewableManagerChatQuietHoursTarget(actorID, managerID uint) (*model.AdminUser, error) {
	repo := repository.NewAdminUserRepository()
	manager, err := repo.FindByID(managerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("管理员不存在")
		}
		return nil, err
	}
	if manager.Role != RoleAdmin {
		return nil, errors.New("该账号不是管理员")
	}

	actor, err := BuildActorContext(actorID)
	if err != nil {
		return nil, err
	}
	if IsSuperAdminRole(actor.Role) || (actor.Role == RoleAdmin && actor.AdminID == manager.ID) {
		return manager, nil
	}
	return nil, errors.New("无权查看此管理员策略")
}

func findEditableManagerChatQuietHoursTarget(actorID, managerID uint, currentPassword string) (*model.AdminUser, error) {
	repo := repository.NewAdminUserRepository()
	actor, err := repo.FindByID(actorID)
	if err != nil {
		return nil, err
	}
	if err := authwork.ComparePrivateHashAndPassword([]byte(actor.PasswordHash), []byte(currentPassword)); err != nil {
		if errors.Is(err, authwork.ErrBusy) {
			return nil, err
		}
		return nil, errors.New("当前密码错误")
	}
	if IsSuperAdminRole(actor.Role) {
		return findManagedManager(actorID, managerID)
	}
	if actor.Role == RoleAdmin && actor.ID == managerID {
		return findViewableManagerChatQuietHoursTarget(actorID, managerID)
	}
	return nil, errors.New("仅超级管理员或管理员本人可操作")
}

func findViewableAssistantChatQuietHoursTarget(actorID, assistantID uint) (*model.AdminUser, error) {
	repo := repository.NewAdminUserRepository()
	assistant, err := repo.FindByID(assistantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("助理不存在")
		}
		return nil, err
	}
	if assistant.Role != RoleAssistant {
		return nil, errors.New("该账号不是助理")
	}

	actor, err := BuildActorContext(actorID)
	if err != nil {
		return nil, err
	}
	if IsSuperAdminRole(actor.Role) {
		return assistant, nil
	}
	if actor.Role == RoleAdmin && assistant.ParentAdminID != nil && *assistant.ParentAdminID == actor.AdminID {
		return assistant, nil
	}
	if actor.Role == RoleAssistant && actor.AdminID == assistant.ID {
		return assistant, nil
	}
	return nil, errors.New("无权查看此助理策略")
}

func findEditableAssistantChatQuietHoursTarget(actorID, assistantID uint, currentPassword string) (*model.AdminUser, error) {
	repo := repository.NewAdminUserRepository()
	actor, err := repo.FindByID(actorID)
	if err != nil {
		return nil, err
	}
	if err := authwork.ComparePrivateHashAndPassword([]byte(actor.PasswordHash), []byte(currentPassword)); err != nil {
		if errors.Is(err, authwork.ErrBusy) {
			return nil, err
		}
		return nil, errors.New("当前密码错误")
	}
	if IsSuperAdminRole(actor.Role) {
		return findManagedAssistant(actorID, assistantID)
	}
	if actor.Role == RoleAdmin {
		return findManagedAssistant(actorID, assistantID)
	}
	if actor.Role == RoleAssistant && actor.ID == assistantID {
		return findViewableAssistantChatQuietHoursTarget(actorID, assistantID)
	}
	return nil, errors.New("仅超级管理员、所属管理员或助理本人可操作")
}
