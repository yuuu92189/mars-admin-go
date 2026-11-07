package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mars-admin-go/internal/bootstrap"
	"mars-admin-go/internal/platform/config"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		port := os.Getenv("APP_PORT")
		if port == "" {
			port = "8080"
		}
		client := http.Client{Timeout: 2 * time.Second}
		request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:"+port+"/health/live", nil) // #nosec G704 -- target is the fixed loopback host and a numeric local port.
		if err != nil {
			os.Exit(1)
		}
		response, err := client.Do(request) // #nosec G704 -- request URL is the fixed loopback health endpoint.
		if err != nil {
			os.Exit(1)
		}
		if response.StatusCode != http.StatusOK {
			_ = response.Body.Close()
			os.Exit(1)
		}
		_ = response.Body.Close()
		return
	}
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	app, err := bootstrap.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("bootstrap API: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
		defer cancel()
		if err := app.Close(closeCtx); err != nil {
			app.Logger.Error("close application", "error", err)
		}
	}()

	server := &http.Server{
		Addr:              cfg.Address(),
		Handler:           app.Router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    1 << 20,
	}
	serverErr := make(chan error, 1)
	go func() {
		app.Logger.Info("API server started", "address", cfg.Address(), "env", cfg.App.Env)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		app.Logger.Info("shutdown signal received")
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	return nil
}
