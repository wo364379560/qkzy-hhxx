package usercache

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/redis/go-redis/v9"
)

func TestOwnershipCacheGenerationRedis(t *testing.T) {
	addr := os.Getenv("RUI_OWNER_REDIS_ADDR")
	if addr == "" {
		t.Skip("requires disposable loopback Redis")
	}
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatal("refusing non-loopback Redis")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()
	ctx := context.Background()
	if client.Ping(ctx).Err() != nil {
		t.Fatal("isolated Redis unavailable")
	}
	oldProvider := redisClient
	redisClient = func() (*redis.Client, bool) { return client, true }
	defer func() { redisClient = oldProvider }()
	im := "owner-test-" + t.Name()
	defer client.Del(ctx, byIMKey(im), byIMKey(im)+":generation")
	old := &model.User{ID: 999, IMUserID: im, OwnerID: 0, OwnerRevision: 0, Status: "active"}
	generation := Generation(ctx, im)
	InvalidateByIMUserID(ctx, im)
	SetByIMUserID(ctx, old, generation)
	if _, ok := GetByIMUserID(ctx, im); ok {
		t.Fatal("late old DB load restored invalidated ownership cache")
	}
	current := &model.User{ID: 999, IMUserID: im, OwnerID: 110, OwnerRevision: 1, Status: "active"}
	SetByIMUserID(ctx, current, Generation(ctx, im))
	saved, ok := GetByIMUserID(ctx, im)
	if !ok || saved.OwnerID != 110 || saved.OwnerRevision != 1 {
		t.Fatal("current ownership summary was not cached")
	}
	SetByIMUserID(ctx, old)
	saved, ok = GetByIMUserID(ctx, im)
	if !ok || saved.OwnerID != 110 {
		t.Fatal("unfenced writer replaced current summary")
	}
}
