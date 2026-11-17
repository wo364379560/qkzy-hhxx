package usercache

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

const byIMTTL = 10 * time.Minute

var keyUnsafePattern = regexp.MustCompile(`[^a-zA-Z0-9:_#@.-]+`)

type Summary struct {
	ID            uint   `json:"id"`
	IMUserID      string `json:"im_user_id"`
	OwnerID       uint   `json:"owner_id"`
	Status        string `json:"status"`
	OwnerRevision uint64 `json:"owner_revision"`
}

func GetByIMUserID(ctx context.Context, imUserID string) (*model.User, bool) {
	client, ok := redisClient()
	if !ok {
		return nil, false
	}
	raw, err := client.Get(ctx, byIMKey(imUserID)).Result()
	if err != nil {
		return nil, false
	}
	var summary Summary
	if err := json.Unmarshal([]byte(raw), &summary); err != nil || summary.ID == 0 {
		return nil, false
	}
	return &model.User{
		ID:            summary.ID,
		IMUserID:      summary.IMUserID,
		OwnerID:       summary.OwnerID,
		Status:        summary.Status,
		OwnerRevision: summary.OwnerRevision,
	}, true
}

func SetByIMUserID(ctx context.Context, user *model.User, generations ...int64) {
	if user == nil || user.ID == 0 || strings.TrimSpace(user.IMUserID) == "" {
		return
	}
	client, ok := redisClient()
	if !ok {
		return
	}
	if len(generations) != 1 || generations[0] < 0 {
		return
	}
	summary := Summary{
		ID:            user.ID,
		IMUserID:      strings.TrimSpace(user.IMUserID),
		OwnerID:       user.OwnerID,
		Status:        user.Status,
		OwnerRevision: user.OwnerRevision,
	}
	raw, err := json.Marshal(summary)
	if err != nil {
		return
	}
	if err := client.Eval(ctx, `if tonumber(redis.call('GET',KEYS[2]) or '0') ~= tonumber(ARGV[1]) then return 0 end; redis.call('SET',KEYS[1],ARGV[2],'EX',ARGV[3]); return 1`, []string{byIMKey(user.IMUserID), byIMKey(user.IMUserID) + ":generation"}, generations[0], string(raw), int(byIMTTL/time.Second)).Err(); err != nil {
		logger.Warnf("cache user summary failed: im_user_id=%s err=%v", user.IMUserID, err)
	}
}

func InvalidateByIMUserID(ctx context.Context, imUserID string) {
	client, ok := redisClient()
	if !ok {
		return
	}
	if err := client.Eval(ctx, `redis.call('INCR',KEYS[2]); redis.call('EXPIRE',KEYS[2],86400); return redis.call('DEL',KEYS[1])`, []string{byIMKey(imUserID), byIMKey(imUserID) + ":generation"}).Err(); err != nil {
		logger.Warnf("invalidate user summary cache failed: im_user_id=%s err=%v", imUserID, err)
	}
}

func Generation(ctx context.Context, imID string) int64 {
	client, ok := redisClient()
	if !ok {
		return -1
	}
	v, err := client.Get(ctx, byIMKey(imID)+":generation").Int64()
	if err == redis.Nil {
		return 0
	}
	if err != nil {
		return -1
	}
	return v
}

var redisClient = configuredRedisClient

func configuredRedisClient() (*redis.Client, bool) {
	redisDS := datasource.Datasource.Redis
	if redisDS == nil || !redisDS.Enabled() {
		return nil, false
	}
	return redisDS.GetClient(), true
}

func byIMKey(imUserID string) string {
	return fmt.Sprintf("chat:user:by_im:v2:%s", sanitizeKeyPart(imUserID))
}

func sanitizeKeyPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "_"
	}
	return keyUnsafePattern.ReplaceAllString(value, "_")
}
