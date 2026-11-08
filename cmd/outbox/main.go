package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	outboxpostgres "mars-admin-go/internal/modules/outbox/adapter/postgres"
	"mars-admin-go/internal/platform/config"
	"mars-admin-go/internal/platform/database"

	"github.com/google/uuid"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return errors.New("usage: outbox list [status] [limit] | requeue <id> | cleanup <retention>")
	}
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()
	repo := outboxpostgres.New(pool)

	switch os.Args[1] {
	case "list":
		status := ""
		if len(os.Args) > 2 {
			status = os.Args[2]
		}
		limit := 50
		if len(os.Args) > 3 {
			parsed, err := strconv.Atoi(os.Args[3])
			if err != nil || parsed < 1 || parsed > 500 {
				return errors.New("limit must be between 1 and 500")
			}
			limit = parsed
		}
		events, err := repo.List(ctx, status, limit, 0)
		if err != nil {
			return err
		}
		return writeJSON(events)
	case "requeue":
		if len(os.Args) != 3 {
			return errors.New("requeue requires an event id")
		}
		id, err := uuid.Parse(os.Args[2])
		if err != nil {
			return fmt.Errorf("parse event id: %w", err)
		}
		if err := repo.Requeue(ctx, id); err != nil {
			return err
		}
		return writeJSON(map[string]any{"id": id, "status": "pending"})
	case "cleanup":
		retention := 7 * 24 * time.Hour
		if len(os.Args) > 2 {
			parsed, err := time.ParseDuration(os.Args[2])
			if err != nil || parsed <= 0 {
				return errors.New("cleanup retention must be a positive Go duration")
			}
			retention = parsed
		}
		deleted, err := repo.DeletePublished(ctx, retention)
		if err != nil {
			return err
		}
		return writeJSON(map[string]any{"deleted": deleted, "retention": retention.String()})
	default:
		return fmt.Errorf("unsupported command %q", os.Args[1])
	}
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
