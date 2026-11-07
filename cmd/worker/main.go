package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"mars-admin-go/internal/modules/integrations"
	notificationriver "mars-admin-go/internal/modules/notification/adapter/river"
	schedulerpostgres "mars-admin-go/internal/modules/scheduler/adapter/postgres"
	schedulerriver "mars-admin-go/internal/modules/scheduler/adapter/river"
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
	logger.Info("River worker started")
	<-ctx.Done()
	stopCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
	defer cancel()
	if err := notificationBackend.Stop(stopCtx); err != nil {
		logger.Error("stop notification worker", "error", err)
	}
	if err := backend.Stop(stopCtx); err != nil {
		return fmt.Errorf("stop River worker: %w", err)
	}
	logger.Info("River worker stopped")
	return nil
}

func stopBackend(backend *schedulerriver.Backend, cfg config.Config) {
	stopCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
	defer cancel()
	_ = backend.Stop(stopCtx)
}
