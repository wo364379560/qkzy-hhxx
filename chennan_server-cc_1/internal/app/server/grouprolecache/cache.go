package grouprolecache

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/logger"
	"github.com/redis/go-redis/v9"
)

const (
	roleCacheTTL     = 30 * time.Minute
	versionCacheTTL  = 7 * 24 * time.Hour
	defaultVersion   = "1"
	cacheValueBypass = "1"
	cacheValueDeny   = "0"
)

var keyUnsafePattern = regexp.MustCompile(`[^a-zA-Z0-9:_#@.-]+`)

func GetBypass(ctx context.Context, groupID, userID string) (bool, bool) {
	client, ok := redisClient()
	if !ok {
		return false, false
	}
	version := getVersion(ctx, groupID)
	value, err := client.Get(ctx, roleKey(groupID, version, userID)).Result()
	if err != nil {
		return false, false
	}
	return value == cacheValueBypass, true
}

func SetBypass(ctx context.Context, groupID, userID string, bypass bool) {
	client, ok := redisClient()
	if !ok {
		return
	}
	version := getVersion(ctx, groupID)
	value := cacheValueDeny
	if bypass {
		value = cacheValueBypass
	}
	if err := client.Set(ctx, roleKey(groupID, version, userID), value, roleCacheTTL).Err(); err != nil {
		logger.Warnf("cache group role bypass failed: group=%s user=%s err=%v", groupID, userID, err)
	}
}

func InvalidateGroup(ctx context.Context, groupID string) {
	client, ok := redisClient()
	if !ok {
		return
	}
	if err := client.Incr(ctx, versionKey(groupID)).Err(); err != nil {
		logger.Warnf("invalidate group role cache failed: group=%s err=%v", groupID, err)
		return
	}
	if err := client.Expire(ctx, versionKey(groupID), versionCacheTTL).Err(); err != nil {
		logger.Warnf("expire group role cache version failed: group=%s err=%v", groupID, err)
	}
}

func getVersion(ctx context.Context, groupID string) string {
	client, ok := redisClient()
	if !ok {
		return defaultVersion
	}
	key := versionKey(groupID)
	version, err := client.Get(ctx, key).Result()
	if err == nil && strings.TrimSpace(version) != "" {
		return strings.TrimSpace(version)
	}
	if err := client.SetNX(ctx, key, defaultVersion, versionCacheTTL).Err(); err != nil {
		logger.Warnf("initialize group role cache version failed: group=%s err=%v", groupID, err)
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

func versionKey(groupID string) string {
	return fmt.Sprintf("chat:group:role_version:%s", sanitizeKeyPart(groupID))
}

func roleKey(groupID, version, userID string) string {
	return fmt.Sprintf("chat:group:role:%s:%s:%s", sanitizeKeyPart(groupID), sanitizeKeyPart(version), sanitizeKeyPart(userID))
}

func sanitizeKeyPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "_"
	}
	return keyUnsafePattern.ReplaceAllString(value, "_")
}
