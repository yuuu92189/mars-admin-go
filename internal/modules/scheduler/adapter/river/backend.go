package river

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"mars-admin-go/internal/modules/scheduler/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/riverqueue/river/rivertype"
	"github.com/robfig/cron/v3"
)

type ScheduledJobArgs struct {
	RunID          uuid.UUID      `json:"runId"`
	JobID          uuid.UUID      `json:"jobId"`
	Handler        string         `json:"handler"`
	Payload        map[string]any `json:"payload"`
	TimeoutSeconds int32          `json:"timeoutSeconds"`
}

func (ScheduledJobArgs) Kind() string { return "scheduler_job" }

func (ScheduledJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "scheduler", MaxAttempts: 5}
}

type TickArgs struct{}

func (TickArgs) Kind() string { return "scheduler_tick" }

type Enqueuer struct {
	client *river.Client[pgx.Tx]
}

func NewEnqueuer(ctx context.Context, pool *pgxpool.Pool) (*Enqueuer, error) {
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return nil, fmt.Errorf("create River migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return nil, fmt.Errorf("migrate River schema: %w", err)
	}
	workers := river.NewWorkers()
	river.AddWorker(workers, &insertOnlyWorker{})
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger:  slog.Default(),
		Workers: workers,
		Queues:  nil,
	})
	if err != nil {
		return nil, err
	}
	return &Enqueuer{client: client}, nil
}

type insertOnlyWorker struct {
	river.WorkerDefaults[ScheduledJobArgs]
}

func (w *insertOnlyWorker) Work(context.Context, *river.Job[ScheduledJobArgs]) error {
	return fmt.Errorf("insert-only scheduler client must not execute jobs")
}

func (e *Enqueuer) Enqueue(ctx context.Context, command domain.Command, runAt time.Time) (int64, error) {
	result, err := e.client.Insert(ctx, ScheduledJobArgs{
		RunID: command.RunID, JobID: command.JobID, Handler: command.Handler, Payload: command.Payload,
		TimeoutSeconds: int32(command.Timeout.Seconds()),
	}, &river.InsertOpts{
		Queue: "scheduler", MaxAttempts: command.Attempt + 4, ScheduledAt: runAt,
	})
	if err != nil {
		return 0, err
	}
	return result.Job.ID, nil
}

type Backend struct {
	client *river.Client[pgx.Tx]
	pool   *pgxpool.Pool
	logger *slog.Logger
}

type BackendOptions struct {
	Pool             *pgxpool.Pool
	Logger           *slog.Logger
	SchedulerRepo    dueRepository
	MaxSchedulerJobs int
}

type dueRepository interface {
	CreateRun(context.Context, domain.Run) (domain.Run, error)
	UpdateRun(context.Context, domain.Run) error
}

func NewBackend(ctx context.Context, opts BackendOptions) (*Backend, error) {
	driver := riverpgxv5.New(opts.Pool)
	migrator, err := rivermigrate.New(driver, nil)
	if err != nil {
		return nil, fmt.Errorf("create River migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return nil, fmt.Errorf("migrate River schema: %w", err)
	}

	backend := &Backend{pool: opts.Pool, logger: opts.Logger}
	workers := river.NewWorkers()
	tick := &tickWorker{pool: opts.Pool, schedulerRepo: opts.SchedulerRepo, logger: opts.Logger}
	jobWorker := &scheduledWorker{pool: opts.Pool, schedulerRepo: opts.SchedulerRepo, logger: opts.Logger}
	river.AddWorker(workers, tick)
	river.AddWorker(workers, jobWorker)

	periodic := river.NewPeriodicJob(
		river.PeriodicInterval(time.Minute),
		func() (river.JobArgs, *river.InsertOpts) {
			return TickArgs{}, &river.InsertOpts{Queue: "scheduler", MaxAttempts: 3}
		},
		&river.PeriodicJobOpts{ID: "scheduler-tick", RunOnStart: true},
	)
	maxWorkers := opts.MaxSchedulerJobs
	if maxWorkers <= 0 {
		maxWorkers = 8
	}
	client, err := river.NewClient(driver, &river.Config{
		Logger:       opts.Logger,
		Workers:      workers,
		PeriodicJobs: []*river.PeriodicJob{periodic},
		Queues: map[string]river.QueueConfig{
			"scheduler": {MaxWorkers: maxWorkers},
		},
		JobTimeout:                  15 * time.Minute,
		CompletedJobRetentionPeriod: 7 * 24 * time.Hour,
		DiscardedJobRetentionPeriod: 30 * 24 * time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("create River client: %w", err)
	}
	tick.client = client
	backend.client = client
	return backend, nil
}

func (b *Backend) Start(ctx context.Context) error {
	return b.client.Start(ctx)
}

func (b *Backend) Stop(ctx context.Context) error {
	return b.client.Stop(ctx)
}

func (b *Backend) Enqueue(ctx context.Context, command domain.Command, runAt time.Time) (int64, error) {
	result, err := b.client.Insert(ctx, ScheduledJobArgs{
		RunID: command.RunID, JobID: command.JobID, Handler: command.Handler, Payload: command.Payload,
		TimeoutSeconds: int32(command.Timeout.Seconds()),
	}, &river.InsertOpts{Queue: "scheduler", MaxAttempts: 5, ScheduledAt: runAt})
	if err != nil {
		return 0, err
	}
	return result.Job.ID, nil
}

type tickWorker struct {
	river.WorkerDefaults[TickArgs]
	pool          *pgxpool.Pool
	schedulerRepo dueRepository
	client        txJobInserter
	logger        *slog.Logger
}

type txJobInserter interface {
	InsertTx(context.Context, pgx.Tx, river.JobArgs, *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

func (w *tickWorker) Timeout(*river.Job[TickArgs]) time.Duration { return 30 * time.Second }

func (w *tickWorker) Work(ctx context.Context, _ *river.Job[TickArgs]) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		SELECT id, handler, payload, max_attempts, timeout_seconds, cron_expression
		FROM scheduler_jobs
		WHERE status = 'active' AND next_run_at IS NOT NULL AND next_run_at <= now()
		ORDER BY next_run_at
		LIMIT 100
		FOR UPDATE SKIP LOCKED`)
	if err != nil {
		return err
	}
	type dueJob struct {
		id             uuid.UUID
		handler        string
		payload        map[string]any
		maxAttempts    int32
		timeoutSeconds int32
		cronExpression string
	}
	jobs := make([]dueJob, 0)
	for rows.Next() {
		var job dueJob
		var payload []byte
		var cronExpression *string
		if err := rows.Scan(&job.id, &job.handler, &payload, &job.maxAttempts, &job.timeoutSeconds, &cronExpression); err != nil {
			rows.Close()
			return err
		}
		if cronExpression != nil {
			job.cronExpression = *cronExpression
		}
		_ = json.Unmarshal(payload, &job.payload)
		jobs = append(jobs, job)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, job := range jobs {
		next := nextCron(job.cronExpression, time.Now().UTC())
		if _, err := tx.Exec(ctx, `
			UPDATE scheduler_jobs
			SET last_enqueued_at = now(), next_run_at = $2
			WHERE id = $1`, job.id, next); err != nil {
			return err
		}
	}
	for _, job := range jobs {
		runID := mustID()
		if _, err := tx.Exec(ctx, `
			INSERT INTO scheduler_job_runs (id, job_id, status, attempt)
			VALUES ($1, $2, 'queued', 1)`, runID, job.id); err != nil {
			return err
		}
		result, err := w.client.InsertTx(ctx, tx, ScheduledJobArgs{
			RunID: runID, JobID: job.id, Handler: job.handler, Payload: job.payload,
			TimeoutSeconds: job.timeoutSeconds,
		}, &river.InsertOpts{Queue: "scheduler", MaxAttempts: int(job.maxAttempts)})
		if err != nil {
			return fmt.Errorf("enqueue scheduled job %s: %w", job.id, err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE scheduler_job_runs SET river_job_id = $2 WHERE id = $1`,
			runID, result.Job.ID); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

type scheduledWorker struct {
	river.WorkerDefaults[ScheduledJobArgs]
	pool          *pgxpool.Pool
	schedulerRepo dueRepository
	logger        *slog.Logger
}

func (w *scheduledWorker) Timeout(job *river.Job[ScheduledJobArgs]) time.Duration {
	return time.Duration(job.Args.TimeoutSeconds) * time.Second
}

func (w *scheduledWorker) Work(ctx context.Context, job *river.Job[ScheduledJobArgs]) error {
	started := time.Now().UTC()
	run := domain.Run{
		ID: job.Args.RunID, JobID: job.Args.JobID, Status: "running",
		StartedAt: &started, Attempt: int32(job.Attempt), CreatedAt: started, // #nosec G115 -- River attempt counts are bounded by MaxAttempts.
	}
	_ = w.schedulerRepo.UpdateRun(ctx, run)
	var workErr error
	switch job.Args.Handler {
	case "system.health_probe":
		workErr = w.pool.Ping(ctx)
	case "storage.cleanup":
		_, workErr = w.pool.Exec(ctx, `
			DELETE FROM storage_files
			WHERE status = 'deleted' AND deleted_at < now() - interval '30 days'`)
	default:
		workErr = fmt.Errorf("unregistered scheduler handler %q", job.Args.Handler)
	}
	finished := time.Now().UTC()
	duration := finished.Sub(started).Milliseconds()
	run.FinishedAt = &finished
	run.DurationMS = &duration
	if workErr != nil {
		run.Status = "failed"
		run.ErrorMessage = workErr.Error()
	} else {
		run.Status = "succeeded"
	}
	if err := w.schedulerRepo.UpdateRun(ctx, run); err != nil {
		w.logger.ErrorContext(ctx, "persist scheduled run result", "error", err)
	}
	return workErr
}

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}

func nextCron(expression string, now time.Time) *time.Time {
	if expression == "" {
		return nil
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(expression)
	if err != nil {
		return nil
	}
	next := schedule.Next(now)
	return &next
}
