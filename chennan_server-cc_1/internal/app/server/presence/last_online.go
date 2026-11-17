package presence

import (
	"context"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/logger"
	"github.com/redis/go-redis/v9"
	"strings"
	"sync"
	"time"
)

const lastOnlinePendingKey = "presence:last_online:events:v2"

type activityEvent struct {
	id string
	at time.Time
}

var startOnce sync.Once
var fallbackQueue = make(chan activityEvent, 4096)

func StartLastOnlineWorker(ctx context.Context) {
	startOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					FlushLastOnline(flushCtx)
					cancel()
					return
				case <-ticker.C:
					FlushLastOnline(ctx)
				}
			}
		}()
	})
}

func EnqueueLastOnline(imUserID string) {
	id := strings.TrimSpace(imUserID)
	if id == "" {
		return
	}
	event := activityEvent{id, time.Now().Truncate(time.Millisecond)}
	ds := datasource.Datasource.Redis
	if ds != nil && ds.Enabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		err := ds.GetClient().ZAddGT(ctx, lastOnlinePendingKey, redis.Z{Score: float64(event.at.UnixMilli()), Member: id}).Err()
		cancel()
		if err == nil {
			return
		}
	}
	requeue(event)
}

func requeue(event activityEvent) {
	select {
	case fallbackQueue <- event:
	default:
		logger.Warnf("last online activity queue is full")
	}
}

func FlushLastOnline(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	pending := map[string]activityEvent{}
	remember := func(event activityEvent) {
		if old, ok := pending[event.id]; !ok || event.at.After(old.at) {
			pending[event.id] = event
		}
	}
	draining := true
	for draining {
		select {
		case event := <-fallbackQueue:
			remember(event)
		default:
			draining = false
		}
	}
	ds := datasource.Datasource.Redis
	if ds != nil && ds.Enabled() {
		rows, err := ds.GetClient().ZRangeWithScores(ctx, lastOnlinePendingKey, 0, 499).Result()
		if err == nil {
			for _, row := range rows {
				if id, ok := row.Member.(string); ok {
					remember(activityEvent{id, time.UnixMilli(int64(row.Score))})
				}
			}
		}
	}
	groups := map[int64][]string{}
	for _, event := range pending {
		groups[event.at.UnixMilli()] = append(groups[event.at.UnixMilli()], event.id)
	}
	for millis, ids := range groups {
		if ctx.Err() != nil {
			for _, id := range ids {
				requeue(pending[id])
			}
			continue
		}
		err := repository.NewUserRepository().BatchTouchLastOnlineAtByIMUserIDs(ids, time.UnixMilli(millis))
		if err != nil {
			for _, id := range ids {
				requeue(pending[id])
			}
			logger.Warnf("last online activity write deferred")
			continue
		}
		if ds != nil && ds.Enabled() {
			for _, id := range ids {
				// Never acknowledge a newer event that arrived during this DB write.
				_ = ds.GetClient().Eval(ctx, `local s=redis.call('ZSCORE',KEYS[1],ARGV[1]);if s and tonumber(s)<=tonumber(ARGV[2]) then return redis.call('ZREM',KEYS[1],ARGV[1]) end;return 0`, []string{lastOnlinePendingKey}, id, millis).Err()
			}
		}
	}
}
