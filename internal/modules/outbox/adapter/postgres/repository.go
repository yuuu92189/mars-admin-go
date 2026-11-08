package postgres

import (
	"context"
	"errors"
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

func (r *Repository) Claim(
	ctx context.Context,
	limit int,
	leaseTTL time.Duration,
	maxExecutionTime time.Duration,
) ([]platformoutbox.Event, error) {
	rows, err := outboxsqlc.New(r.pool).ClaimOutboxEvents(ctx, outboxsqlc.ClaimOutboxEventsParams{
		Limit:   int32(limit), // #nosec G115 -- relay batch size is bounded by application configuration.
		Column2: max(leaseTTL.Milliseconds(), 1),
		Column3: max(maxExecutionTime.Milliseconds(), 1),
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

func (r *Repository) MarkFailed(
	ctx context.Context,
	event platformoutbox.Event,
	reason string,
	baseDelay time.Duration,
	maxDelay time.Duration,
) error {
	affected, err := outboxsqlc.New(r.pool).MarkOutboxFailed(ctx, outboxsqlc.MarkOutboxFailedParams{
		ID: pgUUID(event.ID), LeaseToken: pgUUID(event.LeaseToken),
		LeaseGeneration: event.LeaseGeneration, Column4: reason,
		Column5: max(baseDelay.Seconds(), 1),
		Column6: max(maxDelay.Seconds(), baseDelay.Seconds()),
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return platformoutbox.ErrLeaseLost
	}
	return nil
}

func (r *Repository) RenewLease(
	ctx context.Context,
	event platformoutbox.Event,
	leaseTTL time.Duration,
) (platformoutbox.Lease, error) {
	row, err := outboxsqlc.New(r.pool).RenewOutboxLease(ctx, outboxsqlc.RenewOutboxLeaseParams{
		ID: pgUUID(event.ID), LeaseToken: pgUUID(event.LeaseToken),
		LeaseGeneration: event.LeaseGeneration,
		Column4:         max(leaseTTL.Milliseconds(), 1),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return platformoutbox.Lease{}, platformoutbox.ErrLeaseLost
	}
	if err != nil {
		return platformoutbox.Lease{}, err
	}
	return platformoutbox.Lease{
		ExpiresAt:         timeValue(row.LeaseExpiresAt),
		ExecutionDeadline: timeValue(row.ExecutionDeadline),
	}, nil
}

func (r *Repository) MarkPublished(ctx context.Context, event platformoutbox.Event) error {
	affected, err := outboxsqlc.New(r.pool).MarkOutboxPublished(ctx, outboxsqlc.MarkOutboxPublishedParams{
		ID: pgUUID(event.ID), LeaseToken: pgUUID(event.LeaseToken),
		LeaseGeneration: event.LeaseGeneration,
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return platformoutbox.ErrLeaseLost
	}
	return nil
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
		AvailableAt: timeValue(row.AvailableAt), LeaseGeneration: row.LeaseGeneration,
		LastError: row.LastError, CreatedAt: timeValue(row.CreatedAt),
	}
	event.LockedAt = timePtr(row.LockedAt)
	if row.LeaseToken.Valid {
		event.LeaseToken = uuid.UUID(row.LeaseToken.Bytes)
	}
	event.LeaseExpiresAt = timePtr(row.LeaseExpiresAt)
	event.ExecutionDeadline = timePtr(row.ExecutionDeadline)
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
