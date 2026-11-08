package integrations

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrPaymentOrderNotFound    = errors.New("payment order not found")
	ErrPaymentAmountMismatch   = errors.New("payment amount does not match the order")
	ErrPaymentCurrencyMismatch = errors.New("payment currency does not match the order")
	ErrPaymentStateConflict    = errors.New("payment order state conflict")
	ErrPaymentEventConflict    = errors.New("payment callback event conflicts with a previous event")
)

type PaymentOrder struct {
	ID                    uuid.UUID
	Provider              string
	MerchantOrderID       string
	ProviderOrderID       string
	ProviderTransactionID string
	AmountCents           int64
	Currency              string
	Subject               string
	Status                string
	PaymentURL            string
	ExpiresAt             *time.Time
	PaidAt                *time.Time
	CreatedBy             *uuid.UUID
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type PaymentCallback struct {
	Provider              string
	EventID               string
	MerchantOrderID       string
	ProviderTransactionID string
	AmountCents           int64
	Currency              string
	Status                string
	PaidAt                *time.Time
	PayloadHash           string
}

type PaymentCallbackResult struct {
	Order     PaymentOrder
	Duplicate bool
}

type PaymentStore interface {
	CreateOrder(ctx context.Context, order PaymentOrder) (PaymentOrder, error)
	AttachSession(ctx context.Context, id uuid.UUID, providerOrderID, paymentURL string, payload map[string]string, expiresAt *time.Time) (PaymentOrder, error)
	MarkFailed(ctx context.Context, id uuid.UUID) error
	ApplyCallback(ctx context.Context, callback PaymentCallback) (PaymentCallbackResult, error)
}
