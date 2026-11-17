package controller_chat

import (
	"context"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/quietpolicy"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/config"
)

// ensureRegistrationAllowedNow validates whether self-registration is currently allowed.
func ensureRegistrationAllowedNow(ctx context.Context, ownerID uint, now time.Time) error {
	var (
		policy config.ChatQuietHoursPolicy
		err    error
	)

	if ownerID > 0 {
		resolution, resolveErr := quietpolicy.ResolveChatQuietHoursPolicyForOwner(ctx, ownerID)
		if resolveErr != nil {
			return resolveErr
		}
		if !resolution.HasPolicy {
			return nil
		}
		policy = resolution.Policy
	} else {
		resolution, resolveErr := quietpolicy.ResolveGlobalChatQuietHoursPolicy(ctx)
		if resolveErr != nil {
			return resolveErr
		}
		policy = resolution.Policy
	}

	blocked, err := config.IsChatQuietHoursActionBlockedNow(policy, config.ChatQuietHoursActionRegister, now)
	if err != nil {
		return err
	}
	if blocked {
		return AuthErr(config.ChatQuietHoursBlockedMessage(config.ChatQuietHoursActionRegister, policy))
	}
	return nil
}

// ensureCreateGroupAllowedNow validates whether user-side group creation is currently allowed.
func ensureCreateGroupAllowedNow(ctx context.Context, ownerID uint, now time.Time) error {
	resolution, err := quietpolicy.ResolveChatQuietHoursPolicyForOwner(ctx, ownerID)
	if err != nil {
		return err
	}
	if !resolution.HasPolicy {
		return nil
	}
	policy := resolution.Policy

	blocked, err := config.IsChatQuietHoursActionBlockedNow(policy, config.ChatQuietHoursActionCreateGroup, now)
	if err != nil {
		return err
	}
	if blocked {
		return AuthErr(config.ChatQuietHoursBlockedMessage(config.ChatQuietHoursActionCreateGroup, policy))
	}
	return nil
}

func ensureUserCanCreateGroup(ctx context.Context, imUserID string) error {
	user, err := repository.NewUserRepository().FindByIMUserID(imUserID)
	if err != nil {
		return err
	}
	if !user.CanCreateGroup {
		return AuthErr("当前账号未开启建群权限，请联系管理员")
	}
	return nil
}
