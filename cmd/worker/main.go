package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	identitypostgres "mars-admin-go/internal/modules/identity/adapter/postgres"
	"mars-admin-go/internal/modules/integrations"
	messagingrealtime "mars-admin-go/internal/modules/messaging/realtime"
	notificationriver "mars-admin-go/internal/modules/notification/adapter/river"
	outboxpostgres "mars-admin-go/internal/modules/outbox/adapter/postgres"
	outboxriver "mars-admin-go/internal/modules/outbox/adapter/river"
	outboxapp "mars-admin-go/internal/modules/outbox/application"
	"mars-admin-go/internal/modules/outbox/contracts"
	schedulerpostgres "mars-admin-go/internal/modules/scheduler/adapter/postgres"
	schedulerriver "mars-admin-go/internal/modules/scheduler/adapter/river"
	storagelocal "mars-admin-go/internal/modules/storage/adapter/local"
	storages3 "mars-admin-go/internal/modules/storage/adapter/s3"
	storagedomain "mars-admin-go/internal/modules/storage/domain"
	"mars-admin-go/internal/platform/cache"
	"mars-admin-go/internal/platform/config"
	"mars-admin-go/internal/platform/database"
	"mars-admin-go/internal/platform/logging"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	logger := logging.New(cfg.App.LogLevel, cfg.App.Name+" Worker", cfg.App.Version)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := database.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := database.RunMigrations(ctx, pool, "up"); err != nil {
		return err
	}
	repo := schedulerpostgres.New(pool)
	backend, err := schedulerriver.NewBackend(ctx, schedulerriver.BackendOptions{
		Pool: pool, Logger: logger, SchedulerRepo: repo, MaxSchedulerJobs: 8,
	})
	if err != nil {
		return err
	}
	if err := backend.Start(ctx); err != nil {
		return fmt.Errorf("start River worker: %w", err)
	}
	providerRegistry, smsProvider, err := integrations.Configure(cfg.Integrations, logger, cfg.IsProduction())
	if err != nil {
		stopBackend(backend, cfg)
		return err
	}
	var pushProvider integrations.PushProvider
	if provider, providerErr := providerRegistry.Push(cfg.Integrations.PushWebhook.Provider); providerErr == nil {
		pushProvider = provider
	}
	var mailProvider integrations.MailProvider
	if provider, providerErr := providerRegistry.Mail("smtp"); providerErr == nil {
		mailProvider = provider
	}
	notificationBackend, err := notificationriver.NewBackend(ctx, pool, logger, notificationriver.Options{
		SMS: smsProvider, Push: pushProvider, Mail: mailProvider, MaxJobs: 4,
	})
	if err != nil {
		stopBackend(backend, cfg)
		return err
	}
	redisClient, err := cache.Open(ctx, cfg.Redis)
	if err != nil {
		stopBackend(notificationBackend, cfg)
		stopBackend(backend, cfg)
		return fmt.Errorf("open worker Redis: %w", err)
	}
	defer func() { _ = redisClient.Close() }()

	realtimePublisher := messagingrealtime.NewRedisPublisher(redisClient)
	storageBackend, err := buildWorkerStorageBackend(cfg, logger)
	if err != nil {
		stopBackend(notificationBackend, cfg)
		stopBackend(backend, cfg)
		return err
	}
	consumers := outboxapp.NewConsumers(realtimePublisher, notificationBackend, storageBackend)
	outboxBackend, err := outboxriver.NewBackend(ctx, pool, logger, outboxriver.Options{
		MaxJobs: 8,
		Handlers: map[string]outboxriver.Handler{
			contracts.EventUserStatusChanged: consumers.HandleUserStatusChanged,
			contracts.EventNoticePublished:   consumers.HandleNoticePublished,
			contracts.EventDirectMessage:     consumers.HandleRealtime,
			contracts.EventGroupMessage:      consumers.HandleRealtime,
			contracts.EventWorkflowDecision:  consumers.HandleWorkflowDecision,
			contracts.EventFileDeleted:       consumers.HandleFileDeleted,
		},
	})
	if err != nil {
		stopBackend(notificationBackend, cfg)
		stopBackend(backend, cfg)
		return err
	}
	if err := outboxBackend.Start(ctx); err != nil {
		stopBackend(notificationBackend, cfg)
		stopBackend(backend, cfg)
		return fmt.Errorf("start outbox River worker: %w", err)
	}
	outboxRelay := outboxapp.NewRelay(
		outboxpostgres.New(pool), outboxBackend, logger, outboxapp.RelayOptions{},
	)
	relayCtx, stopRelay := context.WithCancel(ctx)
	relayErr := make(chan error, 1)
	go func() {
		relayErr <- outboxRelay.Run(relayCtx)
	}()
	cleanupCtx, stopCleanup := context.WithCancel(ctx)
	var cleanupWait sync.WaitGroup
	cleanupWait.Add(1)
	go func() {
		defer cleanupWait.Done()
		runRefreshHistoryCleanup(cleanupCtx, identitypostgres.New(pool), cfg.Auth.RefreshHistoryRetention, logger)
	}()
	logger.Info("River worker started")
	var runErr error
	select {
	case <-ctx.Done():
	case err := <-relayErr:
		if err != nil && !errors.Is(err, context.Canceled) {
			runErr = fmt.Errorf("outbox relay stopped: %w", err)
		}
	}
	stopRelay()
	stopCleanup()
	cleanupWait.Wait()
	stopCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
	defer cancel()
	if err := outboxBackend.Stop(stopCtx); err != nil {
		logger.Error("stop outbox worker", "error", err)
	}
	if err := notificationBackend.Stop(stopCtx); err != nil {
		logger.Error("stop notification worker", "error", err)
	}
	if err := backend.Stop(stopCtx); err != nil {
		return fmt.Errorf("stop River worker: %w", err)
	}
	if runErr != nil {
		return runErr
	}
	logger.Info("River worker stopped")
	return nil
}

func buildWorkerStorageBackend(cfg config.Config, logger *slog.Logger) (storagedomain.Backend, error) {
	switch cfg.Storage.Provider {
	case "local":
		return storagelocal.New(cfg.Storage.LocalDir, cfg.Storage.PublicBaseURL)
	case "s3", "minio":
		return storages3.New(storages3.Config{
			Endpoint: cfg.Storage.S3.Endpoint, Region: cfg.Storage.S3.Region,
			Bucket: cfg.Storage.S3.Bucket, AccessKey: cfg.Storage.S3.AccessKey,
			SecretKey: cfg.Storage.S3.SecretKey, PublicBase: cfg.Storage.PublicBaseURL,
		})
	default:
		logger.Error("unsupported worker storage provider", "provider", cfg.Storage.Provider)
		return nil, fmt.Errorf("unsupported storage provider %q", cfg.Storage.Provider)
	}
}

type refreshHistoryCleaner interface {
	CleanupRefreshHistory(context.Context, time.Duration) (int64, error)
}

func runRefreshHistoryCleanup(ctx context.Context, cleaner refreshHistoryCleaner, retention time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		deleted, err := cleaner.CleanupRefreshHistory(ctx, retention)
		if err != nil && ctx.Err() == nil {
			logger.ErrorContext(ctx, "refresh history cleanup failed", "error", err)
		} else if deleted > 0 {
			logger.InfoContext(ctx, "refresh history cleanup completed", "deleted", deleted)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

type backendStopper interface {
	Stop(context.Context) error
}

func stopBackend(backend backendStopper, cfg config.Config) {
	stopCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
	defer cancel()
	_ = backend.Stop(stopCtx)
}
