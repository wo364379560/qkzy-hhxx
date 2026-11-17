package quietpolicycache

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/logger"
	"github.com/redis/go-redis/v9"
)

const (
	policyTTL      = 5 * time.Minute
	versionTTL     = 7 * 24 * time.Hour
	defaultVersion = "1"
	scopeNone      = "none"
	scopeGlobal    = "global"
	scopeAdminUser = "admin_user"
)

var keyUnsafePattern = regexp.MustCompile(`[^a-zA-Z0-9:_#@.-]+`)

type Resolution struct {
	Policy      config.ChatQuietHoursPolicy `json:"policy"`
	HasPolicy   bool                        `json:"has_policy"`
	Scope       string                      `json:"scope"`
	AdminUserID *uint                       `json:"admin_user_id,omitempty"`
}

func Get(ctx context.Context, ownerID uint) (Resolution, bool) {
	client, ok := redisClient()
	if !ok {
		return Resolution{}, false
	}
	version := getVersion(ctx)
	raw, err := client.Get(ctx, policyKey(version, ownerID)).Result()
	if err != nil {
		return Resolution{}, false
	}
	var resolution Resolution
	if err := json.Unmarshal([]byte(raw), &resolution); err != nil {
		return Resolution{}, false
	}
	return resolution, true
}

func Set(ctx context.Context, ownerID uint, resolution Resolution) {
	client, ok := redisClient()
	if !ok {
		return
	}
	version := getVersion(ctx)
	raw, err := json.Marshal(resolution)
	if err != nil {
		return
	}
	if err := client.Set(ctx, policyKey(version, ownerID), raw, policyTTL).Err(); err != nil {
		logger.Warnf("cache quiet policy failed: owner=%d err=%v", ownerID, err)
	}
}

func InvalidateAll(ctx context.Context) {
	client, ok := redisClient()
	if !ok {
		return
	}
	if err := client.Incr(ctx, versionKey()).Err(); err != nil {
		logger.Warnf("invalidate quiet policy cache failed: err=%v", err)
		return
	}
	if err := client.Expire(ctx, versionKey(), versionTTL).Err(); err != nil {
		logger.Warnf("expire quiet policy cache version failed: err=%v", err)
	}
}

func getVersion(ctx context.Context) string {
	client, ok := redisClient()
	if !ok {
		return defaultVersion
	}
	version, err := client.Get(ctx, versionKey()).Result()
	if err == nil && strings.TrimSpace(version) != "" {
		return strings.TrimSpace(version)
	}
	if err := client.SetNX(ctx, versionKey(), defaultVersion, versionTTL).Err(); err != nil {
		logger.Warnf("initialize quiet policy cache version failed: err=%v", err)
	}
	return defaultVersion
}

func redisClient() (*redis.Client, bool) {
	redisDS := datasource.Datasource.Redis
	if redisDS == nil || !redisDS.Enabled() {
		return nil, false
	}
	return redisDS.GetClient(), true
}

func versionKey() string {
	return "chat:quiet_policy_version"
}

func policyKey(version string, ownerID uint) string {
	return fmt.Sprintf("chat:quiet_policy:owner:%s:%d", sanitizeKeyPart(version), ownerID)
}

func sanitizeKeyPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "_"
	}
	return keyUnsafePattern.ReplaceAllString(value, "_")
}

func ToCache(policy config.ChatQuietHoursPolicy, hasPolicy bool, scope string, adminUserID *uint) Resolution {
	return Resolution{
		Policy:      policy,
		HasPolicy:   hasPolicy,
		Scope:       scope,
		AdminUserID: adminUserID,
	}
}
