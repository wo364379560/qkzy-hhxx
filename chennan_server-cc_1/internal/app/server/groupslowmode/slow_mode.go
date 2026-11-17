package groupslowmode

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/grouprolecache"
	"github.com/ixayda/ix-Go/internal/app/server/groupslowcache"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/logger"
	"gorm.io/gorm"
)

const (
	groupRoleOwner                  = "Owner"
	groupRoleAdmin                  = "Admin"
	groupAdminSourceTencent         = "tencent"
	groupAdminSourceTencentRoleSync = repository.GroupAdminSourceTencentRoleSync
)

var keyUnsafePattern = regexp.MustCompile(`[^a-zA-Z0-9:_#@.-]+`)

type Decision struct {
	Blocked          bool
	Reason           string
	RemainingSeconds int64
}

func CheckBeforeSend(ctx context.Context, groupID, userID string) (Decision, error) {
	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	if groupID == "" || userID == "" {
		return Decision{}, nil
	}

	group, ok := groupslowcache.Get(ctx, groupID)
	if !ok {
		var err error
		group, err = repository.NewGroupRepository().FindActiveByIMGroupID(groupID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return Decision{}, nil
			}
			return Decision{}, err
		}
		groupslowcache.Set(ctx, group)
	}
	if !group.SlowModeEnabled || group.SlowModeSeconds <= 0 || group.SlowModeVersion <= 0 {
		return Decision{}, nil
	}

	bypass, err := canBypassSlowMode(groupID, userID)
	if err != nil {
		return Decision{}, err
	}
	if bypass {
		return Decision{}, nil
	}

	return applyRedisSlowMode(ctx, group, userID), nil
}

func canBypassSlowMode(groupID, userID string) (bool, error) {
	if bypass, ok := grouprolecache.GetBypass(context.Background(), groupID, userID); ok {
		return bypass, nil
	}

	repo := repository.NewGroupBusinessAdminRepository()
	hasSyncedOwner, err := repo.HasSyncedTencentOwner(groupID)
	if err != nil {
		return false, err
	}
	if hasSyncedOwner {
		bypass, err := repo.IsActiveRole(
			groupID,
			userID,
			repository.GroupRoleOwner,
			repository.GroupRoleTencentAdmin,
			repository.GroupRoleBusinessAdmin,
		)
		if err != nil {
			return false, err
		}
		grouprolecache.SetBypass(context.Background(), groupID, userID, bypass)
		return bypass, nil
	}

	members, err := tencentim.NewClient().GroupGetAllMembers(groupID)
	if err != nil {
		return false, err
	}
	if err := syncTencentGroupRoles(groupID, members, userID); err != nil {
		return false, err
	}
	bypass := false
	for _, member := range members {
		if strings.TrimSpace(member.Account) != userID {
			continue
		}
		if member.Role == groupRoleOwner || member.Role == groupRoleAdmin {
			bypass = true
		}
		break
	}
	if !bypass {
		isBusinessAdmin, err := repo.IsActive(groupID, userID)
		if err != nil {
			return false, err
		}
		bypass = isBusinessAdmin
	}
	grouprolecache.SetBypass(context.Background(), groupID, userID, bypass)
	return bypass, nil
}

func syncTencentGroupRoles(groupID string, members []tencentim.GroupMember, updatedBy string) error {
	ownerID := ""
	adminIDs := make([]string, 0)
	for _, member := range members {
		account := strings.TrimSpace(member.Account)
		if account == "" {
			continue
		}
		switch strings.TrimSpace(member.Role) {
		case groupRoleOwner:
			ownerID = account
		case groupRoleAdmin:
			adminIDs = append(adminIDs, account)
		}
	}

	repo := repository.NewGroupBusinessAdminRepository()
	if ownerID != "" {
		if err := repo.MarkActiveRoleDeletedExcept(groupID, repository.GroupRoleOwner, ownerID); err != nil {
			return err
		}
		if _, err := repo.UpsertActiveRole(
			groupID,
			ownerID,
			updatedBy,
			groupAdminSourceTencentRoleSync,
			repository.GroupRoleOwner,
		); err != nil {
			return err
		}
	}
	if err := repo.MarkActiveRoleDeletedExceptMany(groupID, repository.GroupRoleTencentAdmin, adminIDs); err != nil {
		return err
	}
	for _, adminID := range adminIDs {
		if _, err := repo.UpsertActiveRole(
			groupID,
			adminID,
			updatedBy,
			groupAdminSourceTencent,
			repository.GroupRoleTencentAdmin,
		); err != nil {
			return err
		}
	}
	grouprolecache.InvalidateGroup(context.Background(), groupID)
	return nil
}

func applyRedisSlowMode(ctx context.Context, group *model.Group, userID string) Decision {
	redisDS := datasource.Datasource.Redis
	if redisDS == nil || !redisDS.Enabled() {
		return Decision{}
	}

	client := redisDS.GetClient()
	interval := time.Duration(group.SlowModeSeconds) * time.Second
	key := fmt.Sprintf(
		"group_slow_mode:%s:%d:%s",
		sanitizeKeyPart(group.IMGroupID),
		group.SlowModeVersion,
		sanitizeKeyPart(userID),
	)
	allowed, err := client.SetNX(ctx, key, time.Now().Unix(), interval).Result()
	if err != nil {
		logger.Warnf("group slow mode redis setnx failed: group=%s user=%s err=%v", group.IMGroupID, userID, err)
		return Decision{}
	}
	if allowed {
		return Decision{}
	}

	ttl, err := client.TTL(ctx, key).Result()
	if err != nil {
		logger.Warnf("group slow mode redis ttl failed: group=%s user=%s err=%v", group.IMGroupID, userID, err)
		ttl = time.Second
	}
	if ttl <= 0 {
		return Decision{}
	}
	remaining := int64((ttl + time.Second - 1) / time.Second)
	return Decision{
		Blocked:          true,
		Reason:           fmt.Sprintf("发言过于频繁，请 %d 秒后再试", remaining),
		RemainingSeconds: remaining,
	}
}

func sanitizeKeyPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "_"
	}
	return keyUnsafePattern.ReplaceAllString(value, "_")
}
