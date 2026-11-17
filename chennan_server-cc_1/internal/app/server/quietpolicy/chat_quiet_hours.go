package quietpolicy

import (
	"context"
	"errors"

	controller_admin "github.com/ixayda/ix-Go/internal/app/server/controller/admin"
	"github.com/ixayda/ix-Go/internal/app/server/quietpolicycache"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"gorm.io/gorm"
)

const (
	// ScopeNone indicates there is no applicable quiet-hours policy for the current owner scope.
	ScopeNone = "none"
	// ScopeGlobal indicates the global super-admin quiet-hours policy is in effect.
	ScopeGlobal = "global"
	// ScopeAdminUser indicates an account-level quiet-hours policy is in effect.
	ScopeAdminUser = "admin_user"
)

// ChatQuietHoursPolicyResolution represents the resolved quiet-hours policy for a scope.
type ChatQuietHoursPolicyResolution struct {
	Policy      config.ChatQuietHoursPolicy
	HasPolicy   bool
	Scope       string
	AdminUserID *uint
}

// ResolveGlobalChatQuietHoursPolicy resolves the global quiet-hours policy.
func ResolveGlobalChatQuietHoursPolicy(ctx context.Context) (ChatQuietHoursPolicyResolution, error) {
	policy, err := config.GetChatQuietHoursPolicy(ctx, datasource.Datasource.Mysql.GetClient())
	if err != nil {
		return ChatQuietHoursPolicyResolution{}, err
	}
	return ChatQuietHoursPolicyResolution{
		Policy:    policy,
		HasPolicy: true,
		Scope:     ScopeGlobal,
	}, nil
}

// ResolveChatQuietHoursPolicyForOwner resolves the quiet-hours policy for a specific owner scope.
func ResolveChatQuietHoursPolicyForOwner(ctx context.Context, ownerID uint) (ChatQuietHoursPolicyResolution, error) {
	if ownerID == 0 {
		return emptyPolicyResolution(), nil
	}
	if cached, ok := quietpolicycache.Get(ctx, ownerID); ok {
		return ChatQuietHoursPolicyResolution{
			Policy:      cached.Policy,
			HasPolicy:   cached.HasPolicy,
			Scope:       cached.Scope,
			AdminUserID: cached.AdminUserID,
		}, nil
	}

	adminRepo := repository.NewAdminUserRepository()
	adminUser, err := adminRepo.FindByID(ownerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			resolution := emptyPolicyResolution()
			quietpolicycache.Set(ctx, ownerID, quietpolicycache.ToCache(resolution.Policy, resolution.HasPolicy, resolution.Scope, resolution.AdminUserID))
			return resolution, nil
		}
		return ChatQuietHoursPolicyResolution{}, err
	}

	if controller_admin.IsSuperAdminRole(adminUser.Role) {
		resolution, err := ResolveGlobalChatQuietHoursPolicy(ctx)
		if err == nil {
			quietpolicycache.Set(ctx, ownerID, quietpolicycache.ToCache(resolution.Policy, resolution.HasPolicy, resolution.Scope, resolution.AdminUserID))
		}
		return resolution, err
	}

	policyRepo := repository.NewAdminChatQuietHoursPolicyRepository()
	policyRecord, err := policyRepo.FindByAdminUserID(ownerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			resolution, globalErr := ResolveGlobalChatQuietHoursPolicy(ctx)
			if globalErr == nil {
				quietpolicycache.Set(ctx, ownerID, quietpolicycache.ToCache(resolution.Policy, resolution.HasPolicy, resolution.Scope, resolution.AdminUserID))
			}
			return resolution, globalErr
		}
		return ChatQuietHoursPolicyResolution{}, err
	}

	policy := config.DefaultChatQuietHoursPolicy()
	policy.Enabled = policyRecord.Enabled
	policy.TimeZone = policyRecord.TimeZone
	policy.Start = policyRecord.Start
	policy.End = policyRecord.End
	policy.BlockRegister = policyRecord.BlockRegister
	policy.BlockSendMessage = policyRecord.BlockSendMessage
	policy.BlockCreateGroup = policyRecord.BlockCreateGroup
	policy.BlockAddFriend = policyRecord.BlockAddFriend
	if err := config.NormalizeChatQuietHoursPolicy(&policy); err != nil {
		return ChatQuietHoursPolicyResolution{}, err
	}

	resolvedOwnerID := ownerID
	resolution := ChatQuietHoursPolicyResolution{
		Policy:      policy,
		HasPolicy:   true,
		Scope:       ScopeAdminUser,
		AdminUserID: &resolvedOwnerID,
	}
	quietpolicycache.Set(ctx, ownerID, quietpolicycache.ToCache(resolution.Policy, resolution.HasPolicy, resolution.Scope, resolution.AdminUserID))
	return resolution, nil
}

func emptyPolicyResolution() ChatQuietHoursPolicyResolution {
	policy := config.DefaultChatQuietHoursPolicy()
	policy.Enabled = false
	return ChatQuietHoursPolicyResolution{
		Policy:    policy,
		HasPolicy: false,
		Scope:     ScopeNone,
	}
}
