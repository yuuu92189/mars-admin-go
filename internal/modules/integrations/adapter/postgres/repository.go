package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"mars-admin-go/internal/modules/integrations"
	sqlc "mars-admin-go/internal/modules/integrations/adapter/postgres/sqlc"

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

func (r *Repository) CreateOrder(ctx context.Context, order integrations.PaymentOrder) (integrations.PaymentOrder, error) {
	row, err := sqlc.New(r.pool).CreatePaymentOrder(ctx, sqlc.CreatePaymentOrderParams{
		ID: pgUUID(order.ID), Provider: order.Provider, MerchantOrderID: order.MerchantOrderID,
		AmountCents: order.AmountCents, Currency: order.Currency, Subject: order.Subject,
		CreatedBy: optionalUUID(order.CreatedBy),
	})
	if err != nil {
		return integrations.PaymentOrder{}, err
	}
	return mapOrder(row), nil
}

func (r *Repository) AttachSession(ctx context.Context, id uuid.UUID, providerOrderID, paymentURL string, payload map[string]string, expiresAt *time.Time) (integrations.PaymentOrder, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return integrations.PaymentOrder{}, err
	}
	row, err := sqlc.New(r.pool).AttachPaymentSession(ctx, sqlc.AttachPaymentSessionParams{
		ID: pgUUID(id), ProviderOrderID: pgText(providerOrderID), PaymentUrl: paymentURL,
		ProviderPayload: encoded, ExpiresAt: optionalTimestamp(expiresAt),
	})
	if err != nil {
		return integrations.PaymentOrder{}, err
	}
	return mapOrder(row), nil
}

func (r *Repository) MarkFailed(ctx context.Context, id uuid.UUID) error {
	return sqlc.New(r.pool).MarkPaymentOrderFailed(ctx, pgUUID(id))
}

func (r *Repository) ApplyCallback(ctx context.Context, callback integrations.PaymentCallback) (integrations.PaymentCallbackResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return integrations.PaymentCallbackResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := sqlc.New(tx)
	order, err := q.GetPaymentOrderByMerchantID(ctx, sqlc.GetPaymentOrderByMerchantIDParams{
		Provider: callback.Provider, MerchantOrderID: callback.MerchantOrderID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return integrations.PaymentCallbackResult{}, integrations.ErrPaymentOrderNotFound
	}
	if err != nil {
		return integrations.PaymentCallbackResult{}, err
	}
	if order.AmountCents != callback.AmountCents {
		return r.rejectCallback(ctx, tx, order, callback, "amount_mismatch", integrations.ErrPaymentAmountMismatch)
	}
	if order.Currency != callback.Currency {
		return r.rejectCallback(ctx, tx, order, callback, "currency_mismatch", integrations.ErrPaymentCurrencyMismatch)
	}

	callbackRow, err := q.InsertPaymentCallback(ctx, callbackParams(order, callback, "accepted", ""))
	if errors.Is(err, pgx.ErrNoRows) {
		existing, getErr := q.GetPaymentCallback(ctx, sqlc.GetPaymentCallbackParams{
			Provider: callback.Provider, EventID: callback.EventID,
		})
		if getErr != nil {
			return integrations.PaymentCallbackResult{}, getErr
		}
		if existing.Status != "accepted" || existing.PayloadHash != callback.PayloadHash {
			return integrations.PaymentCallbackResult{}, integrations.ErrPaymentEventConflict
		}
		current, getErr := q.GetPaymentOrder(ctx, order.ID)
		if getErr != nil {
			return integrations.PaymentCallbackResult{}, getErr
		}
		if err := tx.Commit(ctx); err != nil {
			return integrations.PaymentCallbackResult{}, err
		}
		return integrations.PaymentCallbackResult{Order: mapOrder(current), Duplicate: true}, nil
	}
	if err != nil {
		return integrations.PaymentCallbackResult{}, err
	}
	_ = callbackRow

	switch callback.Status {
	case "paid":
		paidAt := callback.PaidAt
		if paidAt == nil {
			now := time.Now().UTC()
			paidAt = &now
		}
		paid, err := q.MarkPaymentOrderPaid(ctx, sqlc.MarkPaymentOrderPaidParams{
			ID: order.ID, ProviderTransactionID: pgText(callback.ProviderTransactionID),
			PaidAt: optionalTimestamp(paidAt),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			current, getErr := q.GetPaymentOrder(ctx, order.ID)
			if getErr != nil {
				return integrations.PaymentCallbackResult{}, getErr
			}
			if current.Status == "paid" && current.ProviderTransactionID.String == callback.ProviderTransactionID {
				if err := tx.Commit(ctx); err != nil {
					return integrations.PaymentCallbackResult{}, err
				}
				return integrations.PaymentCallbackResult{Order: mapOrder(current), Duplicate: true}, nil
			}
			return integrations.PaymentCallbackResult{}, integrations.ErrPaymentStateConflict
		}
		if err != nil {
			return integrations.PaymentCallbackResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return integrations.PaymentCallbackResult{}, err
		}
		return integrations.PaymentCallbackResult{Order: mapOrder(paid)}, nil
	case "failed":
		closed, err := q.MarkPaymentOrderClosed(ctx, order.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			current, getErr := q.GetPaymentOrder(ctx, order.ID)
			if getErr != nil {
				return integrations.PaymentCallbackResult{}, getErr
			}
			if current.Status == "closed" || current.Status == "failed" {
				if err := tx.Commit(ctx); err != nil {
					return integrations.PaymentCallbackResult{}, err
				}
				return integrations.PaymentCallbackResult{Order: mapOrder(current), Duplicate: true}, nil
			}
			return integrations.PaymentCallbackResult{}, integrations.ErrPaymentStateConflict
		}
		if err != nil {
			return integrations.PaymentCallbackResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return integrations.PaymentCallbackResult{}, err
		}
		return integrations.PaymentCallbackResult{Order: mapOrder(closed)}, nil
	default:
		return integrations.PaymentCallbackResult{}, integrations.ErrPaymentStateConflict
	}
}

func (r *Repository) rejectCallback(ctx context.Context, tx pgx.Tx, order sqlc.PaymentOrder, callback integrations.PaymentCallback, errorCode string, cause error) (integrations.PaymentCallbackResult, error) {
	q := sqlc.New(tx)
	if _, err := q.InsertPaymentCallback(ctx, callbackParams(order, callback, "rejected", errorCode)); errors.Is(err, pgx.ErrNoRows) {
		return integrations.PaymentCallbackResult{}, integrations.ErrPaymentEventConflict
	} else if err != nil {
		return integrations.PaymentCallbackResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return integrations.PaymentCallbackResult{}, err
	}
	return integrations.PaymentCallbackResult{}, cause
}

func callbackParams(order sqlc.PaymentOrder, callback integrations.PaymentCallback, status, errorCode string) sqlc.InsertPaymentCallbackParams {
	return sqlc.InsertPaymentCallbackParams{
		ID: pgUUID(uuid.New()), Provider: callback.Provider, EventID: callback.EventID,
		OrderID: order.ID, ProviderTransactionID: callback.ProviderTransactionID,
		AmountCents: callback.AmountCents, Currency: callback.Currency, Status: status,
		PayloadHash: callback.PayloadHash, ErrorCode: errorCode,
	}
}

func mapOrder(row sqlc.PaymentOrder) integrations.PaymentOrder {
	order := integrations.PaymentOrder{
		ID: uuid.UUID(row.ID.Bytes), Provider: row.Provider, MerchantOrderID: row.MerchantOrderID,
		AmountCents: row.AmountCents, Currency: row.Currency, Subject: row.Subject,
		Status: row.Status, PaymentURL: row.PaymentUrl, CreatedAt: row.CreatedAt.Time.UTC(),
		UpdatedAt: row.UpdatedAt.Time.UTC(),
	}
	if row.ProviderOrderID.Valid {
		order.ProviderOrderID = row.ProviderOrderID.String
	}
	if row.ProviderTransactionID.Valid {
		order.ProviderTransactionID = row.ProviderTransactionID.String
	}
	if row.ExpiresAt.Valid {
		value := row.ExpiresAt.Time.UTC()
		order.ExpiresAt = &value
	}
	if row.PaidAt.Valid {
		value := row.PaidAt.Time.UTC()
		order.PaidAt = &value
	}
	if row.CreatedBy.Valid {
		value := uuid.UUID(row.CreatedBy.Bytes)
		order.CreatedBy = &value
	}
	return order
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: id != uuid.Nil}
}

func optionalUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgUUID(*id)
}

func pgText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func optionalTimestamp(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}
