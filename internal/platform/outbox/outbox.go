package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	outboxsqlc "mars-admin-go/internal/platform/outbox/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const DefaultMaxAttempts int32 = 10

var ErrLeaseLost = errors.New("outbox lease is no longer owned")

type Event struct {
	ID                uuid.UUID       `json:"id"`
	AggregateType     string          `json:"aggregateType"`
	AggregateID       string          `json:"aggregateId"`
	EventType         string          `json:"eventType"`
	DeduplicationKey  string          `json:"deduplicationKey"`
	Payload           json.RawMessage `json:"payload"`
	Status            string          `json:"status"`
	Attempts          int32           `json:"attempts"`
	MaxAttempts       int32           `json:"maxAttempts"`
	AvailableAt       time.Time       `json:"availableAt"`
	LockedAt          *time.Time      `json:"lockedAt,omitempty"`
	LeaseToken        uuid.UUID       `json:"leaseToken"`
	LeaseGeneration   int64           `json:"leaseGeneration"`
	LeaseExpiresAt    *time.Time      `json:"leaseExpiresAt,omitempty"`
	ExecutionDeadline *time.Time      `json:"executionDeadline,omitempty"`
	PublishedAt       *time.Time      `json:"publishedAt,omitempty"`
	LastError         string          `json:"lastError,omitempty"`
	CreatedAt         time.Time       `json:"createdAt"`
}

type Lease struct {
	ExpiresAt         time.Time
	ExecutionDeadline time.Time
}

func NewEvent(aggregateType, aggregateID, eventType, deduplicationKey string, payload any) (Event, error) {
	if aggregateType == "" || aggregateID == "" || eventType == "" {
		return Event{}, errors.New("outbox aggregate and event type are required")
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		id = uuid.New()
	}
	if deduplicationKey == "" {
		deduplicationKey = id.String()
	}
	now := time.Now().UTC()
	return Event{
		ID: id, AggregateType: aggregateType, AggregateID: aggregateID,
		EventType: eventType, DeduplicationKey: deduplicationKey, Payload: encoded,
		Status: "pending", MaxAttempts: DefaultMaxAttempts, CreatedAt: now,
	}, nil
}

func Insert(ctx context.Context, executor outboxsqlc.DBTX, event Event) error {
	if event.ID == uuid.Nil {
		return errors.New("outbox event id is required")
	}
	if event.MaxAttempts <= 0 {
		event.MaxAttempts = DefaultMaxAttempts
	}
	return outboxsqlc.New(executor).InsertOutboxEvent(ctx, outboxsqlc.InsertOutboxEventParams{
		ID:               pgUUID(event.ID),
		AggregateType:    event.AggregateType,
		AggregateID:      event.AggregateID,
		EventType:        event.EventType,
		DeduplicationKey: event.DeduplicationKey,
		Payload:          event.Payload,
		MaxAttempts:      event.MaxAttempts,
		AvailableAt:      pgTimestamp(event.AvailableAt),
	})
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: id != uuid.Nil}
}

func pgTimestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: !value.IsZero()}
}
