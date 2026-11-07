package cache

import (
	"context"
	"fmt"
	"time"

	"mars-admin-go/internal/platform/config"

	"github.com/redis/go-redis/v9"
)

func Open(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	options, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse Redis URL: %w", err)
	}
	if cfg.Password != "" {
		options.Password = cfg.Password
	}
	if cfg.PoolSize > 0 {
		options.PoolSize = cfg.PoolSize
	}
	client := redis.NewClient(options)
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping Redis: %w", err)
	}
	return client, nil
}

func Health(ctx context.Context, client *redis.Client) error {
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return client.Ping(checkCtx).Err()
}
