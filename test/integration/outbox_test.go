//go:build integration

package integration_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	messagingdomain "mars-admin-go/internal/modules/messaging/domain"
	messagingrealtime "mars-admin-go/internal/modules/messaging/realtime"
	notificationriver "mars-admin-go/internal/modules/notification/adapter/river"
	outboxpostgres "mars-admin-go/internal/modules/outbox/adapter/postgres"
	outboxriver "mars-admin-go/internal/modules/outbox/adapter/river"
	outboxapp "mars-admin-go/internal/modules/outbox/application"
	"mars-admin-go/internal/modules/outbox/contracts"
	"mars-admin-go/internal/platform/config"
	"mars-admin-go/internal/platform/database"
	platformoutbox "mars-admin-go/internal/platform/outbox"
	outboxsqlc "mars-admin-go/internal/platform/outbox/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	redisclient "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestOutboxAgainstRealPostgresAndRedis(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	postgresContainer, err := postgres.Run(ctx, "postgres:18.6-bookworm",
		postgres.WithDatabase("mars_admin_outbox"),
		postgres.WithUsername("mars"),
		postgres.WithPassword("mars_test_password"),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, postgresContainer)
	databaseURL, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	redisContainer, err := rediscontainer.Run(ctx, "redis:8.10.1-alpine")
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, redisContainer)
	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	pool, err := database.Open(ctx, config.DatabaseConfig{
		URL: databaseURL, MaxConnections: 30, MinConnections: 2, ConnectionTimeout: 10 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, database.RunMigrations(ctx, pool, "up"))

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := outboxpostgres.New(pool)
	redisClient := redisclient.NewClient(&redisclient.Options{Addr: redisEndpoint})
	t.Cleanup(func() { _ = redisClient.Close() })

	t.Run("business transaction rollback also rolls back outbox", func(t *testing.T) {
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)
		code := "outbox_rollback_" + uuid.NewString()[:8]
		typeID := uuid.New()
		_, err = tx.Exec(ctx, `
			INSERT INTO system_dictionary_types (id, code, name, status)
			VALUES ($1, $2, 'Outbox rollback', 'active')`, typeID, code)
		require.NoError(t, err)
		event := newOutboxEvent(t, "rollback", code, contracts.EventUserStatusChanged, "rollback")
		require.NoError(t, platformoutbox.Insert(ctx, tx, event))
		require.NoError(t, tx.Rollback(ctx))

		var typeCount, eventCount int
		require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM system_dictionary_types WHERE id = $1`, typeID).Scan(&typeCount))
		require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE id = $1`, event.ID).Scan(&eventCount))
		require.Zero(t, typeCount)
		require.Zero(t, eventCount)
	})

	t.Run("duplicate producer event is deduplicated", func(t *testing.T) {
		event := newOutboxEvent(t, "dedupe", "same", contracts.EventUserStatusChanged, "same")
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, platformoutbox.Insert(ctx, tx, event))
		require.NoError(t, platformoutbox.Insert(ctx, tx, event))
		require.NoError(t, tx.Commit(ctx))

		var count int
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT count(*) FROM outbox_events
			WHERE aggregate_type = 'dedupe' AND aggregate_id = 'same'
			  AND event_type = $1 AND deduplication_key = 'same'`,
			contracts.EventUserStatusChanged).Scan(&count))
		require.Equal(t, 1, count)
		_, err = pool.Exec(ctx, `UPDATE outbox_events SET status = 'published', published_at = now() WHERE id = $1`, event.ID)
		require.NoError(t, err)
	})

	t.Run("multiple relay workers claim disjoint batches", func(t *testing.T) {
		const eventCount = 40
		for index := 0; index < eventCount; index++ {
			event := newOutboxEvent(
				t, "multi-worker", uuid.NewString(), contracts.EventDirectMessage,
				uuid.NewString(),
			)
			tx, err := pool.Begin(ctx)
			require.NoError(t, err)
			require.NoError(t, platformoutbox.Insert(ctx, tx, event))
			require.NoError(t, tx.Commit(ctx))
		}
		publisher := &markingPublisher{pool: pool, delay: 2 * time.Millisecond}
		start := make(chan struct{})
		var wait sync.WaitGroup
		errs := make(chan error, 8)
		for range 8 {
			wait.Add(1)
			go func() {
				defer wait.Done()
				<-start
				relay := outboxapp.NewRelay(repo, publisher, logger, outboxapp.RelayOptions{
					BatchSize: 5, LockTimeout: time.Minute,
				})
				errs <- relay.RunOnce(ctx)
			}()
		}
		close(start)
		wait.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}

		var published int
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT count(*) FROM outbox_events
			WHERE aggregate_type = 'multi-worker' AND status = 'published'`).Scan(&published))
		require.Equal(t, eventCount, published)
		require.Equal(t, eventCount, publisher.Count())
	})

	t.Run("failure backoff max attempts dead letter and requeue", func(t *testing.T) {
		event := newOutboxEvent(t, "retry", uuid.NewString(), contracts.EventUserStatusChanged, "retry")
		event.MaxAttempts = 2
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, platformoutbox.Insert(ctx, tx, event))
		require.NoError(t, tx.Commit(ctx))

		failing := publisherFunc(func(context.Context, platformoutbox.Event) error {
			return errors.New("simulated River enqueue failure")
		})
		relay := outboxapp.NewRelay(repo, failing, logger, outboxapp.RelayOptions{
			BatchSize: 1, LockTimeout: time.Minute, BaseBackoff: 50 * time.Millisecond, MaxBackoff: time.Second,
		})
		require.Error(t, relay.RunOnce(ctx))
		var status string
		var attempts int
		var availableAt time.Time
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT status, attempts, available_at FROM outbox_events WHERE id = $1`, event.ID).
			Scan(&status, &attempts, &availableAt))
		require.Equal(t, "failed", status)
		require.Equal(t, 1, attempts)
		require.True(t, availableAt.After(time.Now().UTC()))

		_, err = pool.Exec(ctx, `UPDATE outbox_events SET available_at = now() WHERE id = $1`, event.ID)
		require.NoError(t, err)
		require.Error(t, relay.RunOnce(ctx))
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT status, attempts FROM outbox_events WHERE id = $1`, event.ID).Scan(&status, &attempts))
		require.Equal(t, "dead", status)
		require.Equal(t, 2, attempts)

		require.NoError(t, repo.Requeue(ctx, event.ID))
		success := &markingPublisher{pool: pool}
		require.NoError(t, outboxapp.NewRelay(repo, success, logger, outboxapp.RelayOptions{BatchSize: 1}).RunOnce(ctx))
		require.NoError(t, pool.QueryRow(ctx, `SELECT status FROM outbox_events WHERE id = $1`, event.ID).Scan(&status))
		require.Equal(t, "published", status)
	})

	t.Run("stale processing lock is recovered after relay crash", func(t *testing.T) {
		event := newOutboxEvent(t, "restart", uuid.NewString(), contracts.EventUserStatusChanged, "restart")
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, platformoutbox.Insert(ctx, tx, event))
		require.NoError(t, tx.Commit(ctx))

		claimed, err := repo.Claim(ctx, 1, 20*time.Millisecond)
		require.NoError(t, err)
		require.Len(t, claimed, 1)
		require.Equal(t, int32(1), claimed[0].Attempts)
		time.Sleep(30 * time.Millisecond)
		reclaimed, err := repo.Claim(ctx, 1, 20*time.Millisecond)
		require.NoError(t, err)
		require.Len(t, reclaimed, 1)
		require.Equal(t, event.ID, reclaimed[0].ID)
		require.Equal(t, int32(2), reclaimed[0].Attempts)
		_, err = pool.Exec(ctx, `UPDATE outbox_events SET status = 'published', published_at = now() WHERE id = $1`, event.ID)
		require.NoError(t, err)
	})

	t.Run("River insert and published status are atomic", func(t *testing.T) {
		event := newOutboxEvent(t, "river-atomic", uuid.NewString(), contracts.EventUserStatusChanged, "atomic")
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, platformoutbox.Insert(ctx, tx, event))
		require.NoError(t, tx.Commit(ctx))
		claimed, err := repo.Claim(ctx, 1, time.Minute)
		require.NoError(t, err)
		require.Len(t, claimed, 1)
		require.Equal(t, event.ID, claimed[0].ID)
		_, err = pool.Exec(ctx, `UPDATE outbox_events SET status = 'pending', locked_at = NULL WHERE id = $1`, event.ID)
		require.NoError(t, err)

		backend, err := outboxriver.NewBackend(ctx, pool, logger, outboxriver.Options{})
		require.NoError(t, err)
		require.Error(t, backend.Publish(ctx, claimed[0]))

		var jobs int
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT count(*) FROM river_job
			WHERE kind = 'outbox_event' AND args->>'eventId' = $1`, event.ID.String()).Scan(&jobs))
		require.Zero(t, jobs, "River insert must roll back when outbox status update fails")
		var status string
		require.NoError(t, pool.QueryRow(ctx, `SELECT status FROM outbox_events WHERE id = $1`, event.ID).Scan(&status))
		require.Equal(t, "pending", status)
		_, err = pool.Exec(ctx, `UPDATE outbox_events SET status = 'published', published_at = now() WHERE id = $1`, event.ID)
		require.NoError(t, err)
	})

	t.Run("consumer failure is retried by River", func(t *testing.T) {
		event := newOutboxEvent(t, "consumer-failure", uuid.NewString(), contracts.EventWorkflowDecision, "consumer")
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, platformoutbox.Insert(ctx, tx, event))
		require.NoError(t, tx.Commit(ctx))

		var calls atomic.Int32
		backend, err := outboxriver.NewBackend(ctx, pool, logger, outboxriver.Options{
			MaxJobs: 1,
			Handlers: map[string]outboxriver.Handler{
				contracts.EventWorkflowDecision: func(context.Context, platformoutbox.Event) error {
					calls.Add(1)
					return errors.New("simulated consumer failure")
				},
			},
		})
		require.NoError(t, err)
		require.NoError(t, backend.Start(ctx))
		relayCtx, relayCancel := context.WithCancel(ctx)
		relay := outboxapp.NewRelay(repo, backend, logger, outboxapp.RelayOptions{
			BatchSize: 1, Interval: 50 * time.Millisecond, LockTimeout: time.Minute,
		})
		relayErr := make(chan error, 1)
		go func() { relayErr <- relay.Run(relayCtx) }()
		require.Eventually(t, func() bool { return calls.Load() >= 2 }, 15*time.Second, 100*time.Millisecond)
		relayCancel()
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()
		require.NoError(t, backend.Stop(stopCtx))
		require.ErrorIs(t, <-relayErr, context.Canceled)
	})

	t.Run("Redis realtime target receives event", func(t *testing.T) {
		subscription := redisClient.Subscribe(ctx, "mars:admin:realtime")
		t.Cleanup(func() { _ = subscription.Close() })
		_, err := subscription.Receive(ctx)
		require.NoError(t, err)
		publisher := messagingrealtime.NewRedisPublisher(redisClient)
		userID := uuid.New()
		event := messagingrealtimeEvent(userID)
		require.NoError(t, publisher.Publish(ctx, []uuid.UUID{userID}, event))
		require.NoError(t, publisher.Publish(ctx, []uuid.UUID{userID}, event))
		message, err := subscription.ReceiveMessage(ctx)
		require.NoError(t, err)
		require.Contains(t, message.Payload, userID.String())
		duplicateCtx, duplicateCancel := context.WithTimeout(ctx, 150*time.Millisecond)
		defer duplicateCancel()
		_, err = subscription.ReceiveMessage(duplicateCtx)
		require.Error(t, err, "duplicate realtime event must not be published twice")
		exists, err := redisClient.Exists(ctx, "mars:admin:realtime:event:"+event.ID).Result()
		require.NoError(t, err)
		require.Equal(t, int64(1), exists)
	})

	t.Run("duplicate notice delivery enqueue is idempotent", func(t *testing.T) {
		noticeID := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO notification_notices (
				id, title, content, notice_type, channels, target_type, target_ids, status
			) VALUES ($1, 'Outbox delivery', 'body', 1, '["sms"]'::jsonb, 'all', '[]'::jsonb, 'published')`,
			noticeID)
		require.NoError(t, err)
		backend, err := notificationriver.NewEnqueuer(ctx, pool, logger)
		require.NoError(t, err)
		first, err := backend.Enqueue(ctx, noticeID, "sms")
		require.NoError(t, err)
		second, err := backend.Enqueue(ctx, noticeID, "sms")
		require.NoError(t, err)
		require.Equal(t, first, second)
		var jobs int
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT count(*) FROM river_job
			WHERE kind = 'notification_dispatch'
			  AND args->>'noticeId' = $1
			  AND args->>'channel' = 'sms'`, noticeID.String()).Scan(&jobs))
		require.Equal(t, 1, jobs)
	})

	t.Run("published outbox cleanup respects retention", func(t *testing.T) {
		event := newOutboxEvent(t, "cleanup", uuid.NewString(), contracts.EventUserStatusChanged, "cleanup")
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, platformoutbox.Insert(ctx, tx, event))
		require.NoError(t, tx.Commit(ctx))
		_, err = pool.Exec(ctx, `
			UPDATE outbox_events
			SET status = 'published', published_at = now() - interval '2 days'
			WHERE id = $1`, event.ID)
		require.NoError(t, err)
		deleted, err := repo.DeletePublished(ctx, 24*time.Hour)
		require.NoError(t, err)
		require.GreaterOrEqual(t, deleted, int64(1))
		var count int
		require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE id = $1`, event.ID).Scan(&count))
		require.Zero(t, count)
	})
}

type markingPublisher struct {
	pool  *pgxpool.Pool
	delay time.Duration
	mu    sync.Mutex
	seen  map[uuid.UUID]struct{}
}

func (p *markingPublisher) Publish(ctx context.Context, event platformoutbox.Event) error {
	if p.delay > 0 {
		time.Sleep(p.delay)
	}
	affected, err := outboxsqlc.New(p.pool).MarkOutboxPublished(ctx, pgUUID(event.ID))
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("event is not processing")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.seen == nil {
		p.seen = make(map[uuid.UUID]struct{})
	}
	if _, exists := p.seen[event.ID]; exists {
		return errors.New("duplicate outbox publish")
	}
	p.seen[event.ID] = struct{}{}
	return nil
}

func (p *markingPublisher) Count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.seen)
}

type publisherFunc func(context.Context, platformoutbox.Event) error

func (f publisherFunc) Publish(ctx context.Context, event platformoutbox.Event) error {
	return f(ctx, event)
}

func newOutboxEvent(t *testing.T, aggregateType, aggregateID, eventType, dedup string) platformoutbox.Event {
	t.Helper()
	event, err := platformoutbox.NewEvent(
		aggregateType, aggregateID, eventType, dedup,
		contracts.UserStatusChanged{UserID: uuid.New(), Status: "active", ChangedAt: time.Now().UTC()},
	)
	require.NoError(t, err)
	return event
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: id != uuid.Nil}
}

func messagingrealtimeEvent(userID uuid.UUID) messagingdomain.Event {
	return messagingdomain.Event{
		ID: uuid.NewString(), Type: "userStatus", UserID: &userID,
		Content: "active", Timestamp: time.Now().UTC().UnixMilli(),
	}
}
