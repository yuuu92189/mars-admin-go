package database

import (
	"context"
	"fmt"

	"mars-admin-go/db"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// RunMigrations executes Goose against a pgx pool using database/sql only
// inside the migration command. Runtime queries continue to use pgx directly.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, command string) error {
	goose.SetBaseFS(db.Migrations())
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = sqlDB.Close() }()
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping migration database: %w", err)
	}
	if err := goose.RunContext(ctx, command, sqlDB, "."); err != nil {
		return fmt.Errorf("goose %s: %w", command, err)
	}
	return nil
}
