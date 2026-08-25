package database

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type migration struct {
	version int64
	name    string
	up      string
	down    string
}

var migrations = []migration{{version: 1, name: "create_gateway_configs", up: "migrations/001_create_gateway_configs.up.sql", down: "migrations/001_create_gateway_configs.down.sql"}}

// MigrateUp applies each unapplied migration in order.
func MigrateUp(ctx context.Context, pool *pgxpool.Pool) error {
	if err := ensureMigrationTable(ctx, pool); err != nil {
		return err
	}
	for _, item := range migrations {
		var applied bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM orbit_schema_migrations WHERE version = $1)`, item.version).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %d: %w", item.version, err)
		}
		if applied {
			continue
		}
		sql, err := migrationSQL(item.up)
		if err != nil {
			return err
		}
		transaction, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", item.version, err)
		}
		if _, err = transaction.Exec(ctx, sql); err == nil {
			_, err = transaction.Exec(ctx, `INSERT INTO orbit_schema_migrations (version, name) VALUES ($1, $2)`, item.version, item.name)
		}
		if err != nil {
			_ = transaction.Rollback(ctx)
			return fmt.Errorf("apply migration %d: %w", item.version, err)
		}
		if err := transaction.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %d: %w", item.version, err)
		}
	}
	return nil
}

// MigrateDown rolls back the latest applied migration.
func MigrateDown(ctx context.Context, pool *pgxpool.Pool) error {
	if err := ensureMigrationTable(ctx, pool); err != nil {
		return err
	}
	var version int64
	if err := pool.QueryRow(ctx, `SELECT version FROM orbit_schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&version); err != nil {
		if err == pgx.ErrNoRows {
			return nil
		}
		return fmt.Errorf("find latest migration: %w", err)
	}
	for _, item := range migrations {
		if item.version != version {
			continue
		}
		sql, err := migrationSQL(item.down)
		if err != nil {
			return err
		}
		transaction, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration rollback: %w", err)
		}
		if _, err = transaction.Exec(ctx, sql); err == nil {
			_, err = transaction.Exec(ctx, `DELETE FROM orbit_schema_migrations WHERE version = $1`, version)
		}
		if err != nil {
			_ = transaction.Rollback(ctx)
			return fmt.Errorf("rollback migration %d: %w", version, err)
		}
		return transaction.Commit(ctx)
	}
	return fmt.Errorf("migration %d is unknown", version)
}

func ensureMigrationTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS orbit_schema_migrations (version BIGINT PRIMARY KEY, name TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`)
	return err
}

func migrationSQL(path string) (string, error) {
	contents, err := migrationFiles.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read migration %s: %w", path, err)
	}
	return strings.TrimSpace(string(contents)), nil
}
