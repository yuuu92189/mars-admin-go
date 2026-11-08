package river

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	platformoutbox "mars-admin-go/internal/platform/outbox"
	outboxsqlc "mars-admin-go/internal/platform/outbox/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

const QueueName = "outbox"

type EventArgs struct {
	EventID          uuid.UUID       `json:"eventId"`
	AggregateType    string          `json:"aggregateType"`
	AggregateID      string          `json:"aggregateId"`
	EventType        string          `json:"eventType"`
	DeduplicationKey string          `json:"deduplicationKey"`
	Payload          json.RawMessage `json:"payload"`
	CreatedAt        time.Time       `json:"createdAt"`
}

func (EventArgs) Kind() string { return "outbox_event" }

func (EventArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: QueueName, MaxAttempts: int(platformoutbox.DefaultMaxAttempts),
		UniqueOpts: river.UniqueOpts{ByArgs: true},
	}
}

type Handler func(context.Context, platformoutbox.Event) error

type Backend struct {
	client   *river.Client[pgx.Tx]
	pool     *pgxpool.Pool
	handlers map[string]Handler
	logger   *slog.Logger
}

type Options struct {
	Handlers map[string]Handler
	MaxJobs  int
}

func NewBackend(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, options Options) (*Backend, error) {
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return nil, fmt.Errorf("create outbox River migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return nil, fmt.Errorf("migrate outbox River schema: %w", err)
	}
	maxJobs := options.MaxJobs
	if maxJobs <= 0 {
		maxJobs = 4
	}
	handlers := options.Handlers
	if handlers == nil {
		handlers = map[string]Handler{}
	}
	workers := river.NewWorkers()
	river.AddWorker(workers, &eventWorker{handlers: handlers, logger: logger})
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger: logger, Workers: workers,
		Queues:     map[string]river.QueueConfig{QueueName: {MaxWorkers: maxJobs}},
		JobTimeout: 2 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("create outbox River client: %w", err)
	}
	return &Backend{client: client, pool: pool, handlers: handlers, logger: logger}, nil
}

func (b *Backend) Start(ctx context.Context) error {
	return b.client.Start(ctx)
}

func (b *Backend) Stop(ctx context.Context) error {
	return b.client.Stop(ctx)
}

func (b *Backend) Publish(ctx context.Context, event platformoutbox.Event) error {
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result, err := b.client.InsertTx(ctx, tx, EventArgs{
		EventID: event.ID, AggregateType: event.AggregateType, AggregateID: event.AggregateID,
		EventType: event.EventType, DeduplicationKey: event.DeduplicationKey,
		Payload: event.Payload, CreatedAt: event.CreatedAt,
	}, &river.InsertOpts{
		Queue: QueueName, MaxAttempts: int(max(event.MaxAttempts, 1)),
		UniqueOpts: river.UniqueOpts{ByArgs: true},
	})
	if err != nil {
		return fmt.Errorf("insert outbox River job: %w", err)
	}
	affected, err := outboxsqlc.New(tx).MarkOutboxPublished(ctx, outboxsqlc.MarkOutboxPublishedParams{
		ID: pgUUID(event.ID), LeaseToken: pgUUID(event.LeaseToken),
		LeaseGeneration: event.LeaseGeneration,
	})
	if err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("outbox event %s: %w", event.ID, platformoutbox.ErrLeaseLost)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit outbox event %s and River job %d: %w", event.ID, result.Job.ID, err)
	}
	return nil
}

type eventWorker struct {
	river.WorkerDefaults[EventArgs]
	handlers map[string]Handler
	logger   *slog.Logger
}

func (w *eventWorker) Work(ctx context.Context, job *river.Job[EventArgs]) error {
	handler := w.handlers[job.Args.EventType]
	if handler == nil {
		return fmt.Errorf("no outbox consumer registered for event type %q", job.Args.EventType)
	}
	event := platformoutbox.Event{
		ID: job.Args.EventID, AggregateType: job.Args.AggregateType,
		AggregateID: job.Args.AggregateID, EventType: job.Args.EventType,
		DeduplicationKey: job.Args.DeduplicationKey, Payload: job.Args.Payload,
		CreatedAt: job.Args.CreatedAt,
	}
	if err := handler(ctx, event); err != nil {
		w.logger.WarnContext(ctx, "outbox consumer failed",
			"event_id", event.ID, "event_type", event.EventType, "attempt", job.Attempt, "error", err)
		return err
	}
	return nil
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: id != uuid.Nil}
}
