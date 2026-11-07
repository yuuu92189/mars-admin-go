package postgres

import (
	"context"
	"encoding/json"
	"net/netip"
	"time"

	sqlc "mars-admin-go/internal/modules/audit/adapter/postgres/sqlc"
	"mars-admin-go/internal/modules/audit/domain"
	"mars-admin-go/internal/platform/requestctx"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Insert(ctx context.Context, event domain.Event) error {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return err
	}
	var actor pgtype.UUID
	if event.ActorUserID != nil {
		actor = pgUUID(*event.ActorUserID)
	}
	return sqlc.New(r.pool).InsertAuditEvent(ctx, sqlc.InsertAuditEventParams{
		ID:           pgUUID(event.ID),
		OccurredAt:   pgTimestamp(event.OccurredAt),
		ActorUserID:  actor,
		ActorName:    event.ActorName,
		Action:       event.Action,
		ResourceType: event.ResourceType,
		ResourceID:   event.ResourceID,
		Outcome:      event.Outcome,
		RequestID:    firstNonEmpty(event.RequestID, requestctx.RequestID(ctx)),
		TraceID:      firstNonEmpty(event.TraceID, requestctx.TraceID(ctx)),
		IpAddress:    parseAddr(event.IPAddress),
		UserAgent:    event.UserAgent,
		Metadata:     metadata,
	})
}

func (r *Repository) List(ctx context.Context, filter domain.Filter) ([]domain.Event, int64, error) {
	q := sqlc.New(r.pool)
	rows, err := q.ListAuditEvents(ctx, sqlc.ListAuditEventsParams{
		Column1: filter.Action,
		Column2: filter.ResourceType,
		Column3: optionalTimestamp(filter.From),
		Column4: optionalTimestamp(filter.To),
		Limit:   int32(filter.PageSize),                     // #nosec G115 -- application layer caps page size at 100.
		Offset:  int32((filter.Page - 1) * filter.PageSize), // #nosec G115 -- page values are bounded by ParsePagination.
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := q.CountAuditEvents(ctx, sqlc.CountAuditEventsParams{
		Column1: filter.Action,
		Column2: filter.ResourceType,
		Column3: optionalTimestamp(filter.From),
		Column4: optionalTimestamp(filter.To),
	})
	if err != nil {
		return nil, 0, err
	}
	events := make([]domain.Event, 0, len(rows))
	for _, row := range rows {
		metadata := make(map[string]any)
		_ = json.Unmarshal(row.Metadata, &metadata)
		event := domain.Event{
			ID:           uuid.UUID(row.ID.Bytes),
			OccurredAt:   row.OccurredAt.Time.UTC(),
			ActorName:    row.ActorName,
			Action:       row.Action,
			ResourceType: row.ResourceType,
			ResourceID:   row.ResourceID,
			Outcome:      row.Outcome,
			RequestID:    row.RequestID,
			TraceID:      row.TraceID,
			IPAddress:    addrString(row.IpAddress),
			UserAgent:    row.UserAgent,
			Metadata:     metadata,
		}
		if row.ActorUserID.Valid {
			id := uuid.UUID(row.ActorUserID.Bytes)
			event.ActorUserID = &id
		}
		events = append(events, event)
	}
	return events, total, nil
}

func (r *Repository) ListLogin(ctx context.Context, page, pageSize int) ([]domain.LoginEvent, int64, error) {
	q := sqlc.New(r.pool)
	rows, err := q.ListLoginEvents(ctx, sqlc.ListLoginEventsParams{
		Limit: int32(pageSize), Offset: int32((page - 1) * pageSize), // #nosec G115 -- application layer caps page size at 100.
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := q.CountLoginEvents(ctx)
	if err != nil {
		return nil, 0, err
	}
	events := make([]domain.LoginEvent, 0, len(rows))
	for _, row := range rows {
		event := domain.LoginEvent{
			ID: uuid.UUID(row.ID.Bytes), Username: row.Username, ClientType: row.ClientType,
			Outcome: row.Outcome, FailureReason: row.FailureReason,
			IPAddress: addrString(row.IpAddress), UserAgent: row.UserAgent, OccurredAt: row.OccurredAt.Time.UTC(),
		}
		if row.UserID.Valid {
			id := uuid.UUID(row.UserID.Bytes)
			event.UserID = &id
		}
		events = append(events, event)
	}
	return events, total, nil
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: id != uuid.Nil}
}

func pgTimestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: !value.IsZero()}
}

func optionalTimestamp(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgTimestamp(*value)
}

func parseAddr(value string) *netip.Addr {
	addr, err := netip.ParseAddr(value)
	if err != nil {
		return nil
	}
	return &addr
}

func addrString(value *netip.Addr) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
