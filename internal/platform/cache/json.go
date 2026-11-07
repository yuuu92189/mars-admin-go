package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type JSON struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

func NewJSON(client *redis.Client, prefix string, ttl time.Duration) *JSON {
	return &JSON{client: client, prefix: prefix, ttl: ttl}
}

func (c *JSON) Get(ctx context.Context, key string, destination any) (bool, error) {
	if c == nil || c.client == nil {
		return false, nil
	}
	value, err := c.client.Get(ctx, c.key(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read cache: %w", err)
	}
	if err := json.Unmarshal(value, destination); err != nil {
		_ = c.client.Del(ctx, c.key(key)).Err()
		return false, fmt.Errorf("decode cache: %w", err)
	}
	return true, nil
}

func (c *JSON) Set(ctx context.Context, key string, value any) error {
	if c == nil || c.client == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode cache: %w", err)
	}
	if err := c.client.Set(ctx, c.key(key), encoded, c.ttl).Err(); err != nil {
		return fmt.Errorf("write cache: %w", err)
	}
	return nil
}

func (c *JSON) Delete(ctx context.Context, keys ...string) error {
	if c == nil || c.client == nil || len(keys) == 0 {
		return nil
	}
	prefixed := make([]string, 0, len(keys))
	for _, key := range keys {
		prefixed = append(prefixed, c.key(key))
	}
	return c.client.Del(ctx, prefixed...).Err()
}

func (c *JSON) DeletePrefix(ctx context.Context, suffix string) error {
	if c == nil || c.client == nil {
		return nil
	}
	match := c.key(suffix) + "*"
	iterator := c.client.Scan(ctx, 0, match, 200).Iterator()
	keys := make([]string, 0, 200)
	for iterator.Next(ctx) {
		keys = append(keys, iterator.Val())
		if len(keys) >= 200 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
			keys = keys[:0]
		}
	}
	if err := iterator.Err(); err != nil {
		return err
	}
	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}
	return nil
}

func (c *JSON) key(value string) string {
	return c.prefix + ":" + value
}
