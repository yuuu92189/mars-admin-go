package postgres

import (
	"context"
	"time"

	platformoutbox "mars-admin-go/internal/platform/outbox"
	outboxsqlc "mars-admin-go/internal/platform/outbox/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Claim(ctx context.Context, limit int, lockTimeout time.Duration) ([]platformoutbox.Event, error) {
	rows, err := outboxsqlc.New(r.pool).ClaimOutboxEvents(ctx, outboxsqlc.ClaimOutboxEventsParams{
		Limit: int32(limit), Column2: max(lockTimeout.Milliseconds(), 0), // #nosec G115 -- relay batch size is bounded by application configuration.
	})
	if err != nil {
		return nil, err
	}
	events := make([]platformoutbox.Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, mapEvent(row))
	}
	return events, nil
}

func (r *Repository) MarkFailed(ctx context.Context, id uuid.UUID, reason string, baseDelay, maxDelay time.Duration) error {
	_, err := outboxsqlc.New(r.pool).MarkOutboxFailed(ctx, outboxsqlc.MarkOutboxFailedParams{
		ID: pgUUID(id), Column2: reason,
		Column3: max(baseDelay.Seconds(), 1),
		Column4: max(maxDelay.Seconds(), baseDelay.Seconds()),
	})
	return err
}

func (r *Repository) Requeue(ctx context.Context, id uuid.UUID) error {
	affected, err := outboxsqlc.New(r.pool).RequeueOutboxEvent(ctx, pgUUID(id))
	if err != nil {
		return err
	}
	if affected == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (platformoutbox.Event, error) {
	row, err := outboxsqlc.New(r.pool).GetOutboxEvent(ctx, pgUUID(id))
	if err != nil {
		return platformoutbox.Event{}, err
	}
	return mapEvent(row), nil
}

func (r *Repository) List(ctx context.Context, status string, limit, offset int) ([]platformoutbox.Event, error) {
	rows, err := outboxsqlc.New(r.pool).ListOutboxEvents(ctx, outboxsqlc.ListOutboxEventsParams{
		Column1: status, Limit: int32(limit), Offset: int32(offset), // #nosec G115 -- CLI validates bounded pagination.
	})
	if err != nil {
		return nil, err
	}
	events := make([]platformoutbox.Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, mapEvent(row))
	}
	return events, nil
}

func (r *Repository) DeletePublished(ctx context.Context, retention time.Duration) (int64, error) {
	return outboxsqlc.New(r.pool).DeletePublishedOutboxEvents(ctx, max(retention.Milliseconds(), 0))
}

func mapEvent(row outboxsqlc.OutboxEvent) platformoutbox.Event {
	event := platformoutbox.Event{
		ID: uuid.UUID(row.ID.Bytes), AggregateType: row.AggregateType,
		AggregateID: row.AggregateID, EventType: row.EventType,
		DeduplicationKey: row.DeduplicationKey, Payload: row.Payload,
		Status: row.Status, Attempts: row.Attempts, MaxAttempts: row.MaxAttempts,
		AvailableAt: timeValue(row.AvailableAt), LastError: row.LastError,
		CreatedAt: timeValue(row.CreatedAt),
	}
	event.LockedAt = timePtr(row.LockedAt)
	event.PublishedAt = timePtr(row.PublishedAt)
	return event
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: id != uuid.Nil}
}

func timeValue(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time.UTC()
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time.UTC()
	return &result
}
