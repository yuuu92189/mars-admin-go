package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"mars-admin-go/internal/modules/messaging/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisPublisher struct {
	client *redis.Client
}

const publishOnceScript = `
local inserted = redis.call("SET", KEYS[1], "1", "NX", "EX", ARGV[2])
if inserted then
    redis.call("PUBLISH", KEYS[2], ARGV[1])
    return 1
end
return 0
`

func NewRedisPublisher(client *redis.Client) *RedisPublisher {
	return &RedisPublisher{client: client}
}

func (p *RedisPublisher) Publish(ctx context.Context, recipients []uuid.UUID, event domain.Event) error {
	if p == nil || p.client == nil {
		return errors.New("realtime Redis publisher is not configured")
	}
	if event.ID == "" {
		return errors.New("realtime event id is required")
	}
	payload, err := json.Marshal(envelope{Recipients: recipients, Event: event})
	if err != nil {
		return err
	}
	return p.client.Eval(ctx, publishOnceScript,
		[]string{"mars:admin:realtime:event:" + event.ID, redisChannel},
		payload, int64((7 * 24 * time.Hour).Seconds()),
	).Err()
}
