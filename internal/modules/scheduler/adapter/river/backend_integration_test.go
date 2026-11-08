//go:build integration

package river

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"mars-admin-go/internal/platform/config"
	"mars-admin-go/internal/platform/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type failingTxInserter struct{}

func (failingTxInserter) InsertTx(context.Context, pgx.Tx, river.JobArgs, *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return nil, errors.New("simulated River insert failure")
}

func TestSchedulerTickRollsBackWhenRiverInsertFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	container, err := postgres.Run(ctx, "postgres:18.6-bookworm",
		postgres.WithDatabase("mars_admin_scheduler"),
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

	first, err := pool.Acquire(ctx)
	require.NoError(t, err)
	defer first.Release()
	second, err := pool.Acquire(ctx)
	require.NoError(t, err)
	defer second.Release()
	acquired, err := tryConcurrencyLock(ctx, first, "same-key")
	require.NoError(t, err)
	require.True(t, acquired)
	acquired, err = tryConcurrencyLock(ctx, second, "same-key")
	require.NoError(t, err)
	require.False(t, acquired)
	require.NoError(t, releaseConcurrencyLock(ctx, first, "same-key"))
	acquired, err = tryConcurrencyLock(ctx, second, "same-key")
	require.NoError(t, err)
	require.True(t, acquired)
	require.NoError(t, releaseConcurrencyLock(ctx, second, "same-key"))

	jobID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO scheduler_jobs (
			id, name, handler, payload, status, max_attempts, timeout_seconds, next_run_at
		) VALUES ($1, 'Scheduler rollback', 'system.health_probe', '{}'::jsonb,
		          'active', 3, 30, now() - interval '1 minute')`, jobID)
	require.NoError(t, err)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	worker := &tickWorker{pool: pool, client: failingTxInserter{}, logger: logger}
	require.Error(t, worker.Work(ctx, nil))

	var runs int
	var due bool
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT count(*) FROM scheduler_job_runs WHERE job_id = $1`, jobID).Scan(&runs))
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT next_run_at <= now() FROM scheduler_jobs WHERE id = $1`, jobID).Scan(&due))
	require.Zero(t, runs)
	require.True(t, due, "next_run_at must roll back when River insert fails")
}
