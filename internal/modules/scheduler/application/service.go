package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"mars-admin-go/internal/modules/scheduler/domain"
	"mars-admin-go/internal/platform/apperror"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/robfig/cron/v3"
)

type Service struct {
	repo     domain.Repository
	enqueuer domain.Enqueuer
	handlers map[string]struct{}
}

func New(repo domain.Repository, enqueuer domain.Enqueuer, handlers []string) *Service {
	allowed := make(map[string]struct{}, len(handlers))
	for _, handler := range handlers {
		allowed[handler] = struct{}{}
	}
	return &Service{repo: repo, enqueuer: enqueuer, handlers: allowed}
}

func (s *Service) Create(ctx context.Context, createdBy uuid.UUID, input domain.Input) (domain.Job, error) {
	nextRun, err := s.validateInput(&input)
	if err != nil {
		return domain.Job{}, err
	}
	job := domain.Job{
		ID: mustID(), Name: input.Name, Handler: input.Handler, CronExpression: input.CronExpression,
		Payload: input.Payload, Status: "paused", MaxAttempts: input.MaxAttempts,
		TimeoutSeconds: input.TimeoutSeconds, ConcurrencyKey: input.ConcurrencyKey,
		NextRunAt: nextRun, CreatedBy: &createdBy, CreatedAt: time.Now().UTC(),
	}
	return s.repo.Create(ctx, job)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (domain.Job, error) {
	job, err := s.repo.Get(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Job{}, apperror.NotFound("Scheduled job")
	}
	return job, err
}

func (s *Service) List(ctx context.Context, status string, page, pageSize int) ([]domain.Job, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(ctx, status, page, pageSize)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, input domain.Input) (domain.Job, error) {
	nextRun, err := s.validateInput(&input)
	if err != nil {
		return domain.Job{}, err
	}
	return s.repo.Update(ctx, id, input, nextRun)
}

func (s *Service) Pause(ctx context.Context, id uuid.UUID) (domain.Job, error) {
	return s.repo.SetStatus(ctx, id, "paused", nil)
}

func (s *Service) Resume(ctx context.Context, id uuid.UUID) (domain.Job, error) {
	job, err := s.Get(ctx, id)
	if err != nil {
		return domain.Job{}, err
	}
	if job.CronExpression == "" {
		return domain.Job{}, apperror.Validation("cronExpression", "A scheduled job must have a cron expression to resume.")
	}
	return s.repo.SetStatus(ctx, id, "active", nextCronTime(job.CronExpression, time.Now().UTC()))
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) RunNow(ctx context.Context, id uuid.UUID) (domain.Run, error) {
	job, err := s.Get(ctx, id)
	if err != nil {
		return domain.Run{}, err
	}
	run := domain.Run{ID: mustID(), JobID: id, Status: "queued", Attempt: 1, CreatedAt: time.Now().UTC()}
	created, err := s.repo.CreateRun(ctx, run)
	if err != nil {
		return domain.Run{}, err
	}
	command := domain.Command{
		RunID: created.ID, JobID: job.ID, Handler: job.Handler, Payload: job.Payload,
		Timeout: time.Duration(job.TimeoutSeconds) * time.Second, ConcurrencyKey: job.ConcurrencyKey,
	}
	riverID, err := s.enqueuer.Enqueue(ctx, command, time.Now().UTC())
	if err != nil {
		created.Status = "failed"
		created.ErrorMessage = err.Error()
		_ = s.repo.UpdateRun(ctx, created)
		return domain.Run{}, err
	}
	created.RiverJobID = &riverID
	_ = s.repo.UpdateRun(ctx, created)
	return created, nil
}

func (s *Service) Runs(ctx context.Context, jobID uuid.UUID, page, pageSize int) ([]domain.Run, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListRuns(ctx, jobID, page, pageSize)
}

func (s *Service) validateInput(input *domain.Input) (*time.Time, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Handler = strings.TrimSpace(input.Handler)
	if input.Name == "" || len(input.Name) > 150 {
		return nil, apperror.Validation("name", "Job name is required and must not exceed 150 characters.")
	}
	if _, ok := s.handlers[input.Handler]; !ok {
		return nil, apperror.Validation("handler", "The job handler is not registered.")
	}
	if input.MaxAttempts == 0 {
		input.MaxAttempts = 5
	}
	if input.MaxAttempts < 1 || input.MaxAttempts > 25 {
		return nil, apperror.Validation("maxAttempts", "maxAttempts must be between 1 and 25.")
	}
	if input.TimeoutSeconds == 0 {
		input.TimeoutSeconds = 300
	}
	if input.TimeoutSeconds < 1 || input.TimeoutSeconds > 86400 {
		return nil, apperror.Validation("timeoutSeconds", "timeoutSeconds must be between 1 and 86400.")
	}
	if input.Payload == nil {
		input.Payload = map[string]any{}
	}
	if input.CronExpression == "" {
		return nil, nil
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(input.CronExpression)
	if err != nil {
		return nil, apperror.Validation("cronExpression", "Invalid five-field cron expression.")
	}
	next := schedule.Next(time.Now().UTC())
	return &next, nil
}

func nextCronTime(expression string, now time.Time) *time.Time {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(expression)
	if err != nil {
		return nil
	}
	next := schedule.Next(now)
	return &next
}

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}
