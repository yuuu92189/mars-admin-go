package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Job struct {
	ID             uuid.UUID      `json:"id"`
	Name           string         `json:"name"`
	Handler        string         `json:"handler"`
	CronExpression string         `json:"cronExpression,omitempty"`
	Payload        map[string]any `json:"payload"`
	Status         string         `json:"status"`
	MaxAttempts    int32          `json:"maxAttempts"`
	TimeoutSeconds int32          `json:"timeoutSeconds"`
	ConcurrencyKey string         `json:"concurrencyKey,omitempty"`
	NextRunAt      *time.Time     `json:"nextRunAt,omitempty"`
	LastEnqueuedAt *time.Time     `json:"lastEnqueuedAt,omitempty"`
	CreatedBy      *uuid.UUID     `json:"createdBy,omitempty"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type Run struct {
	ID           uuid.UUID  `json:"id"`
	JobID        uuid.UUID  `json:"jobId"`
	RiverJobID   *int64     `json:"riverJobId,omitempty"`
	Status       string     `json:"status"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	FinishedAt   *time.Time `json:"finishedAt,omitempty"`
	DurationMS   *int64     `json:"durationMs,omitempty"`
	Attempt      int32      `json:"attempt"`
	ErrorMessage string     `json:"errorMessage,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type Input struct {
	Name           string         `json:"name"`
	Handler        string         `json:"handler"`
	CronExpression string         `json:"cronExpression"`
	Payload        map[string]any `json:"payload"`
	MaxAttempts    int32          `json:"maxAttempts"`
	TimeoutSeconds int32          `json:"timeoutSeconds"`
	ConcurrencyKey string         `json:"concurrencyKey"`
}

type Command struct {
	RunID   uuid.UUID
	JobID   uuid.UUID
	Handler string
	Payload map[string]any
	Timeout time.Duration
	Attempt int
}

type Enqueuer interface {
	Enqueue(ctx context.Context, command Command, runAt time.Time) (int64, error)
}

type Repository interface {
	Create(ctx context.Context, job Job) (Job, error)
	Get(ctx context.Context, id uuid.UUID) (Job, error)
	List(ctx context.Context, status string, page, pageSize int) ([]Job, int64, error)
	Update(ctx context.Context, id uuid.UUID, input Input, nextRunAt *time.Time) (Job, error)
	SetStatus(ctx context.Context, id uuid.UUID, status string, nextRunAt *time.Time) (Job, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CreateRun(ctx context.Context, run Run) (Run, error)
	UpdateRun(ctx context.Context, run Run) error
	ListRuns(ctx context.Context, jobID uuid.UUID, page, pageSize int) ([]Run, error)
}
