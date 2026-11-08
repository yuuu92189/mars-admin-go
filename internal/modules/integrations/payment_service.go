package integrations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"mars-admin-go/internal/platform/apperror"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type PaymentService struct {
	registry *Registry
	store    PaymentStore
	now      func() time.Time
}

func NewPaymentService(registry *Registry, store PaymentStore) *PaymentService {
	return &PaymentService{registry: registry, store: store, now: func() time.Time { return time.Now().UTC() }}
}

func (s *PaymentService) Create(ctx context.Context, createdBy uuid.UUID, providerName string, request PaymentRequest) (PaymentSession, error) {
	providerName = strings.TrimSpace(providerName)
	if s.registry == nil || s.store == nil {
		return PaymentSession{}, apperror.NotConfigured("payment provider")
	}
	provider, err := s.registry.Payment(providerName)
	if err != nil {
		return PaymentSession{}, apperror.NotConfigured("payment provider")
	}
	request.OrderID = strings.TrimSpace(request.OrderID)
	if request.OrderID == "" {
		request.OrderID = newPaymentOrderID()
	}
	if !validMerchantOrderID(request.OrderID) {
		return PaymentSession{}, apperror.Validation("orderId", "orderId must contain only letters, numbers, dot, dash or underscore.")
	}
	if request.AmountCents <= 0 {
		return PaymentSession{}, apperror.Validation("amountCents", "amountCents must be positive.")
	}
	request.Subject = strings.TrimSpace(request.Subject)
	if request.Subject == "" {
		request.Subject = "Payment"
	}
	if len(request.Subject) > 200 {
		return PaymentSession{}, apperror.Validation("subject", "subject must not exceed 200 characters.")
	}
	request.Currency = strings.ToUpper(strings.TrimSpace(request.Currency))
	if request.Currency == "" {
		request.Currency = "CNY"
	}
	if request.Currency != "CNY" {
		return PaymentSession{}, apperror.Validation("currency", "only CNY is supported by the configured providers.")
	}

	orderID, err := uuid.NewV7()
	if err != nil {
		orderID = uuid.New()
	}
	order, err := s.store.CreateOrder(ctx, PaymentOrder{
		ID: orderID, Provider: providerName, MerchantOrderID: request.OrderID,
		AmountCents: request.AmountCents, Currency: request.Currency, Subject: request.Subject,
		Status: "pending", CreatedBy: &createdBy, CreatedAt: s.now(),
	})
	if err != nil {
		return PaymentSession{}, mapPaymentStoreError(err)
	}
	session, err := provider.Create(ctx, request)
	if err != nil {
		_ = s.store.MarkFailed(ctx, order.ID)
		return PaymentSession{}, err
	}
	session.OrderID = order.MerchantOrderID
	if session.Provider == "" {
		session.Provider = providerName
	}
	if session.PaymentURL == "" {
		session.PaymentURL = session.Payload["url"]
	}
	if session.PaymentURL == "" {
		session.PaymentURL = session.Payload["codeUrl"]
	}
	if _, err := s.store.AttachSession(ctx, order.ID, request.OrderID, session.PaymentURL, session.Payload, session.ExpiresAt); err != nil {
		_ = s.store.MarkFailed(ctx, order.ID)
		return PaymentSession{}, mapPaymentStoreError(err)
	}
	return session, nil
}

func (s *PaymentService) HandleCallback(ctx context.Context, providerName string, headers map[string]string, body []byte) (PaymentCallbackResult, error) {
	if s.registry == nil || s.store == nil {
		return PaymentCallbackResult{}, apperror.NotConfigured("payment provider")
	}
	provider, err := s.registry.Payment(providerName)
	if err != nil {
		return PaymentCallbackResult{}, apperror.NotConfigured("payment provider")
	}
	if err := provider.VerifyCallback(ctx, headers, body); err != nil {
		return PaymentCallbackResult{}, apperror.New(apperror.KindBadRequest, "invalid_payment_callback", "Payment callback verification failed.")
	}
	callback, err := provider.ParseCallback(ctx, headers, body)
	if err != nil {
		return PaymentCallbackResult{}, apperror.New(apperror.KindBadRequest, "invalid_payment_callback", err.Error())
	}
	callback.Provider = providerName
	if callback.EventID == "" || callback.MerchantOrderID == "" || callback.ProviderTransactionID == "" {
		return PaymentCallbackResult{}, apperror.New(apperror.KindBadRequest, "invalid_payment_callback", "Payment callback is missing required order fields.")
	}
	if callback.Currency == "" {
		callback.Currency = "CNY"
	}
	callback.PayloadHash = payloadHash(body)
	result, err := s.store.ApplyCallback(ctx, callback)
	if err != nil {
		return PaymentCallbackResult{}, mapPaymentStoreError(err)
	}
	return result, nil
}

func mapPaymentStoreError(err error) error {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return apperror.NotFound("Payment order")
	case errors.Is(err, ErrPaymentOrderNotFound):
		return apperror.NotFound("Payment order")
	case errors.Is(err, ErrPaymentAmountMismatch), errors.Is(err, ErrPaymentCurrencyMismatch):
		return apperror.Conflict("Payment callback amount or currency does not match the order.")
	case errors.Is(err, ErrPaymentEventConflict), errors.Is(err, ErrPaymentStateConflict):
		return apperror.Conflict(err.Error())
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperror.Conflict("A payment order with this merchant order ID already exists.")
		}
		return err
	}
}

func validMerchantOrderID(value string) bool {
	if value == "" || len(value) > 120 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '.' || char == '-' || char == '_' {
			continue
		}
		return false
	}
	return true
}

func newPaymentOrderID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}

func payloadHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
