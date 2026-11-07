package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"mars-admin-go/internal/platform/config"
	"mars-admin-go/internal/platform/database"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	switch command {
	case "up", "down", "status", "version", "reset":
	default:
		return fmt.Errorf("unsupported migration command %q", command)
	}
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pool, err := database.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()
	return database.RunMigrations(ctx, pool, command)
}
