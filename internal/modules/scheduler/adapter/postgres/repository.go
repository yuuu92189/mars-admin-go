package postgres

import (
	"context"
	"encoding/json"
	"time"

	sqlc "mars-admin-go/internal/modules/scheduler/adapter/postgres/sqlc"
	"mars-admin-go/internal/modules/scheduler/domain"
	"mars-admin-go/internal/platform/apperror"

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

func (r *Repository) Create(ctx context.Context, job domain.Job) (domain.Job, error) {
	payload, _ := json.Marshal(job.Payload)
	row, err := sqlc.New(r.pool).CreateSchedulerJob(ctx, sqlc.CreateSchedulerJobParams{
		ID: pgUUID(job.ID), Name: job.Name, Handler: job.Handler, CronExpression: pgText(job.CronExpression),
		Payload: payload, Status: job.Status, MaxAttempts: job.MaxAttempts, TimeoutSeconds: job.TimeoutSeconds,
		ConcurrencyKey: pgText(job.ConcurrencyKey), NextRunAt: optionalTimestamp(job.NextRunAt),
		CreatedBy: optionalUUID(job.CreatedBy),
	})
	if err != nil {
		return domain.Job{}, err
	}
	return mapJob(row), nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (domain.Job, error) {
	row, err := sqlc.New(r.pool).GetSchedulerJob(ctx, pgUUID(id))
	if err != nil {
		return domain.Job{}, err
	}
	return mapJob(row), nil
}

func (r *Repository) List(ctx context.Context, status string, page, pageSize int) ([]domain.Job, int64, error) {
	q := sqlc.New(r.pool)
	rows, err := q.ListSchedulerJobs(ctx, sqlc.ListSchedulerJobsParams{
		Column1: status, Limit: int32(pageSize), Offset: int32((page - 1) * pageSize), // #nosec G115 -- application layer caps page size at 100.
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := q.CountSchedulerJobs(ctx, status)
	if err != nil {
		return nil, 0, err
	}
	jobs := make([]domain.Job, 0, len(rows))
	for _, row := range rows {
		jobs = append(jobs, mapJob(row))
	}
	return jobs, total, nil
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, input domain.Input, nextRunAt *time.Time) (domain.Job, error) {
	payload, _ := json.Marshal(input.Payload)
	row, err := sqlc.New(r.pool).UpdateSchedulerJob(ctx, sqlc.UpdateSchedulerJobParams{
		ID: pgUUID(id), Name: input.Name, Handler: input.Handler, CronExpression: pgText(input.CronExpression),
		Payload: payload, MaxAttempts: input.MaxAttempts, TimeoutSeconds: input.TimeoutSeconds,
		ConcurrencyKey: pgText(input.ConcurrencyKey), NextRunAt: optionalTimestamp(nextRunAt),
	})
	if err != nil {
		return domain.Job{}, err
	}
	return mapJob(row), nil
}

func (r *Repository) SetStatus(ctx context.Context, id uuid.UUID, status string, nextRunAt *time.Time) (domain.Job, error) {
	row, err := sqlc.New(r.pool).SetSchedulerJobStatus(ctx, sqlc.SetSchedulerJobStatusParams{
		ID: pgUUID(id), Status: status, NextRunAt: optionalTimestamp(nextRunAt),
	})
	if err != nil {
		return domain.Job{}, err
	}
	return mapJob(row), nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	affected, err := sqlc.New(r.pool).DeleteSchedulerJob(ctx, pgUUID(id))
	if err != nil {
		return err
	}
	if affected == 0 {
		return apperror.NotFound("Scheduled job")
	}
	return nil
}

func (r *Repository) CreateRun(ctx context.Context, run domain.Run) (domain.Run, error) {
	row, err := sqlc.New(r.pool).CreateSchedulerRun(ctx, sqlc.CreateSchedulerRunParams{
		ID: pgUUID(run.ID), JobID: pgUUID(run.JobID), RiverJobID: optionalInt64(run.RiverJobID), Status: run.Status,
	})
	if err != nil {
		return domain.Run{}, err
	}
	return mapRun(row), nil
}

func (r *Repository) UpdateRun(ctx context.Context, run domain.Run) error {
	return sqlc.New(r.pool).UpdateSchedulerRun(ctx, sqlc.UpdateSchedulerRunParams{
		ID: pgUUID(run.ID), Status: run.Status, StartedAt: optionalTimestamp(run.StartedAt),
		FinishedAt: optionalTimestamp(run.FinishedAt), DurationMs: optionalInt64(run.DurationMS),
		Attempt: run.Attempt, ErrorMessage: run.ErrorMessage,
	})
}

func (r *Repository) ListRuns(ctx context.Context, jobID uuid.UUID, page, pageSize int) ([]domain.Run, error) {
	rows, err := sqlc.New(r.pool).ListSchedulerRuns(ctx, sqlc.ListSchedulerRunsParams{
		JobID: pgUUID(jobID), Limit: int32(pageSize), Offset: int32((page - 1) * pageSize), // #nosec G115 -- application layer caps page size at 100.
	})
	if err != nil {
		return nil, err
	}
	runs := make([]domain.Run, 0, len(rows))
	for _, row := range rows {
		runs = append(runs, mapRun(row))
	}
	return runs, nil
}

func mapJob(row sqlc.SchedulerJob) domain.Job {
	job := domain.Job{
		ID: uuid.UUID(row.ID.Bytes), Name: row.Name, Handler: row.Handler,
		CronExpression: text(row.CronExpression), Status: row.Status, MaxAttempts: row.MaxAttempts,
		TimeoutSeconds: row.TimeoutSeconds, ConcurrencyKey: text(row.ConcurrencyKey),
		CreatedAt: row.CreatedAt.Time.UTC(), UpdatedAt: row.UpdatedAt.Time.UTC(),
	}
	_ = json.Unmarshal(row.Payload, &job.Payload)
	job.NextRunAt = timestampPtr(row.NextRunAt)
	job.LastEnqueuedAt = timestampPtr(row.LastEnqueuedAt)
	if row.CreatedBy.Valid {
		id := uuid.UUID(row.CreatedBy.Bytes)
		job.CreatedBy = &id
	}
	return job
}

func mapRun(row sqlc.SchedulerJobRun) domain.Run {
	run := domain.Run{
		ID: uuid.UUID(row.ID.Bytes), JobID: uuid.UUID(row.JobID.Bytes), Status: row.Status,
		Attempt: row.Attempt, ErrorMessage: row.ErrorMessage, CreatedAt: row.CreatedAt.Time.UTC(),
	}
	if row.RiverJobID.Valid {
		value := row.RiverJobID.Int64
		run.RiverJobID = &value
	}
	if row.DurationMs.Valid {
		value := row.DurationMs.Int64
		run.DurationMS = &value
	}
	run.StartedAt = timestampPtr(row.StartedAt)
	run.FinishedAt = timestampPtr(row.FinishedAt)
	return run
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

func text(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func optionalTimestamp(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func timestampPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time.UTC()
	return &result
}

func optionalInt64(value *int64) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *value, Valid: true}
}
