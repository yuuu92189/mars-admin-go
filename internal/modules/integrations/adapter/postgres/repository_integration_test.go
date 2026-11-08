//go:build integration

package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"mars-admin-go/internal/modules/integrations"
	"mars-admin-go/internal/platform/config"
	"mars-admin-go/internal/platform/database"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPaymentLedgerBindsAmountAndDeduplicatesCallbacks(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	container, err := postgres.Run(ctx, "postgres:18.6-bookworm",
		postgres.WithDatabase("mars_admin_payments"),
		postgres.WithUsername("mars"),
		postgres.WithPassword("mars_test_password"),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, container)
	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	pool, err := database.Open(ctx, config.DatabaseConfig{
		URL: databaseURL, MaxConnections: 5, MinConnections: 1, ConnectionTimeout: 10 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, database.RunMigrations(ctx, pool, "up"))

	repo := New(pool)
	order, err := repo.CreateOrder(ctx, integrations.PaymentOrder{
		ID: uuid.New(), Provider: "wechat", MerchantOrderID: "ORDER-LEDGER-1",
		AmountCents: 1999, Currency: "CNY", Subject: "Ledger test",
	})
	require.NoError(t, err)
	require.Equal(t, "pending", order.Status)

	paidAt := time.Now().UTC()
	callback := integrations.PaymentCallback{
		Provider: "wechat", EventID: "evt-ledger-1", MerchantOrderID: order.MerchantOrderID,
		ProviderTransactionID: "wx-tx-ledger-1", AmountCents: 1999, Currency: "CNY",
		Status: "paid", PaidAt: &paidAt, PayloadHash: strings.Repeat("a", 64),
	}
	result, err := repo.ApplyCallback(ctx, callback)
	require.NoError(t, err)
	require.False(t, result.Duplicate)
	require.Equal(t, "paid", result.Order.Status)

	result, err = repo.ApplyCallback(ctx, callback)
	require.NoError(t, err)
	require.True(t, result.Duplicate)
	require.Equal(t, "paid", result.Order.Status)

	conflict := callback
	conflict.PayloadHash = strings.Repeat("b", 64)
	_, err = repo.ApplyCallback(ctx, conflict)
	require.ErrorIs(t, err, integrations.ErrPaymentEventConflict)

	mismatch := callback
	mismatch.EventID = "evt-ledger-2"
	mismatch.AmountCents = 100
	_, err = repo.ApplyCallback(ctx, mismatch)
	require.ErrorIs(t, err, integrations.ErrPaymentAmountMismatch)

	var rejected int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT count(*) FROM payment_callbacks
		WHERE provider = 'wechat' AND event_id = 'evt-ledger-2' AND status = 'rejected'
	`).Scan(&rejected))
	require.Equal(t, 1, rejected)
}
