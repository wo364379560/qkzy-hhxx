package datasource

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/pkg/logger"
)

type redisClient struct {
	client  *redis.Client
	enabled bool
}

func (r *redisClient) GetClient() *redis.Client {
	return r.client
}

func (r *redisClient) Enabled() bool {
	return r.enabled && r.client != nil
}

func (r *redisClient) openRedis() {
	options := config.Cfg.Get().Redis
	if !options.Enabled {
		logger.Infof("Redis disabled")
		return
	}

	client := redis.NewClient(&redis.Options{
		Addr:         options.Addr,
		Password:     options.Password,
		DB:           options.DB,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warnf("Redis unavailable, distributed rate limits disabled; local fallback remains active: addr=%s err=%v", options.Addr, err)
		_ = client.Close()
		return
	}

	r.client = client
	r.enabled = true
	logger.Infof("Redis Use %s [OK]", options.Addr)
}
