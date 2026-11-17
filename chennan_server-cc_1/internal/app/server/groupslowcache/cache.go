package groupslowcache

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/logger"
	"github.com/redis/go-redis/v9"
)

const slowModeTTL = 10 * time.Minute

var keyUnsafePattern = regexp.MustCompile(`[^a-zA-Z0-9:_#@.-]+`)

type slowModeSnapshot struct {
	ID              uint   `json:"id"`
	IMGroupID       string `json:"im_group_id"`
	SlowModeEnabled bool   `json:"slow_mode_enabled"`
	SlowModeSeconds int    `json:"slow_mode_seconds"`
	SlowModeVersion int64  `json:"slow_mode_version"`
}

func Get(ctx context.Context, groupID string) (*model.Group, bool) {
	client, ok := redisClient()
	if !ok {
		return nil, false
	}
	raw, err := client.Get(ctx, key(groupID)).Result()
	if err != nil {
		return nil, false
	}
	var snapshot slowModeSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil || snapshot.ID == 0 {
		return nil, false
	}
	return &model.Group{
		ID:              snapshot.ID,
		IMGroupID:       snapshot.IMGroupID,
		SlowModeEnabled: snapshot.SlowModeEnabled,
		SlowModeSeconds: snapshot.SlowModeSeconds,
		SlowModeVersion: snapshot.SlowModeVersion,
		Status:          "active",
	}, true
}

func Set(ctx context.Context, group *model.Group) {
	if group == nil || group.ID == 0 || strings.TrimSpace(group.IMGroupID) == "" {
		return
	}
	client, ok := redisClient()
	if !ok {
		return
	}
	snapshot := slowModeSnapshot{
		ID:              group.ID,
		IMGroupID:       group.IMGroupID,
		SlowModeEnabled: group.SlowModeEnabled,
		SlowModeSeconds: group.SlowModeSeconds,
		SlowModeVersion: group.SlowModeVersion,
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	if err := client.Set(ctx, key(group.IMGroupID), raw, slowModeTTL).Err(); err != nil {
		logger.Warnf("cache group slow mode failed: group=%s err=%v", group.IMGroupID, err)
	}
}

func Invalidate(ctx context.Context, groupID string) {
	client, ok := redisClient()
	if !ok {
		return
	}
	if err := client.Del(ctx, key(groupID)).Err(); err != nil {
		logger.Warnf("invalidate group slow mode cache failed: group=%s err=%v", groupID, err)
	}
}

func redisClient() (*redis.Client, bool) {
	redisDS := datasource.Datasource.Redis
	if redisDS == nil || !redisDS.Enabled() {
		return nil, false
	}
	return redisDS.GetClient(), true
}

func key(groupID string) string {
	return fmt.Sprintf("chat:group:slow_mode:%s", sanitizeKeyPart(groupID))
}

func sanitizeKeyPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "_"
	}
	return keyUnsafePattern.ReplaceAllString(value, "_")
}
