package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	platformoutbox "mars-admin-go/internal/platform/outbox"

	"github.com/google/uuid"
)

type Repository interface {
	Claim(ctx context.Context, limit int, lockTimeout time.Duration) ([]platformoutbox.Event, error)
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, baseDelay, maxDelay time.Duration) error
	Requeue(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (platformoutbox.Event, error)
	List(ctx context.Context, status string, limit, offset int) ([]platformoutbox.Event, error)
	DeletePublished(ctx context.Context, retention time.Duration) (int64, error)
}

type Publisher interface {
	Publish(ctx context.Context, event platformoutbox.Event) error
}

type Relay struct {
	repo               Repository
	publisher          Publisher
	logger             *slog.Logger
	batchSize          int
	interval           time.Duration
	lockTTL            time.Duration
	baseDelay          time.Duration
	maxDelay           time.Duration
	cleanupInterval    time.Duration
	publishedRetention time.Duration
}

type RelayOptions struct {
	BatchSize          int
	Interval           time.Duration
	LockTimeout        time.Duration
	BaseBackoff        time.Duration
	MaxBackoff         time.Duration
	CleanupInterval    time.Duration
	PublishedRetention time.Duration
}

func NewRelay(repo Repository, publisher Publisher, logger *slog.Logger, options RelayOptions) *Relay {
	if options.BatchSize <= 0 {
		options.BatchSize = 50
	}
	if options.Interval <= 0 {
		options.Interval = time.Second
	}
	if options.LockTimeout <= 0 {
		options.LockTimeout = 2 * time.Minute
	}
	if options.BaseBackoff <= 0 {
		options.BaseBackoff = 2 * time.Second
	}
	if options.MaxBackoff < options.BaseBackoff {
		options.MaxBackoff = 10 * time.Minute
	}
	if options.CleanupInterval <= 0 {
		options.CleanupInterval = time.Hour
	}
	if options.PublishedRetention <= 0 {
		options.PublishedRetention = 7 * 24 * time.Hour
	}
	return &Relay{
		repo: repo, publisher: publisher, logger: logger,
		batchSize: options.BatchSize, interval: options.Interval, lockTTL: options.LockTimeout,
		baseDelay: options.BaseBackoff, maxDelay: options.MaxBackoff,
		cleanupInterval: options.CleanupInterval, publishedRetention: options.PublishedRetention,
	}
}

func (r *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	cleanupTicker := time.NewTicker(r.cleanupInterval)
	defer cleanupTicker.Stop()
	for {
		if err := r.RunOnce(ctx); err != nil && ctx.Err() == nil {
			r.logger.ErrorContext(ctx, "outbox relay batch failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		case <-cleanupTicker.C:
			deleted, err := r.Cleanup(ctx, r.publishedRetention)
			if err != nil && ctx.Err() == nil {
				r.logger.ErrorContext(ctx, "outbox cleanup failed", "error", err)
			} else if deleted > 0 {
				r.logger.InfoContext(ctx, "outbox cleanup completed", "deleted", deleted)
			}
		}
	}
}

func (r *Relay) RunOnce(ctx context.Context) error {
	events, err := r.repo.Claim(ctx, r.batchSize, r.lockTTL)
	if err != nil {
		return fmt.Errorf("claim outbox events: %w", err)
	}
	var failures []error
	for _, event := range events {
		if err := r.publisher.Publish(ctx, event); err != nil {
			markErr := r.repo.MarkFailed(ctx, event.ID, err.Error(), r.baseDelay, r.maxDelay)
			if markErr != nil {
				failures = append(failures, fmt.Errorf("publish %s: %w", event.ID, errors.Join(err, markErr)))
			} else {
				failures = append(failures, fmt.Errorf("publish %s: %w", event.ID, err))
			}
			continue
		}
		r.logger.InfoContext(ctx, "outbox event published to River",
			"event_id", event.ID, "event_type", event.EventType,
			"aggregate_type", event.AggregateType, "aggregate_id", event.AggregateID)
	}
	if len(failures) > 0 {
		return fmt.Errorf("%d outbox events failed: %w", len(failures), failures[0])
	}
	return nil
}

func (r *Relay) Requeue(ctx context.Context, id uuid.UUID) error {
	return r.repo.Requeue(ctx, id)
}

func (r *Relay) Cleanup(ctx context.Context, retention time.Duration) (int64, error) {
	return r.repo.DeletePublished(ctx, retention)
}
