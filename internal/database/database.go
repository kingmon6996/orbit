package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kingmon6996/orbit/internal/config"
)

var (
	ErrNotFound = errors.New("record not found")
	ErrConflict = errors.New("record conflict")
	ErrDatabase = errors.New("database error")
)

// Database owns the application's PostgreSQL connection pool.
type Database struct {
	pool *pgxpool.Pool
}

// New creates and validates a PostgreSQL pool. It returns nil when disabled.
func New(ctx context.Context, configuration config.Config, logger *slog.Logger) (*Database, error) {
	if !configuration.DatabaseEnabled {
		return nil, nil
	}
	poolConfig, err := pgxpool.ParseConfig(configuration.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	poolConfig.MaxConns = configuration.DatabaseMaxConns
	poolConfig.MinConns = configuration.DatabaseMinConns
	poolConfig.MaxConnLifetime = configuration.DatabaseMaxConnLifetime
	poolConfig.MaxConnIdleTime = configuration.DatabaseMaxConnIdleTime
	poolConfig.HealthCheckPeriod = configuration.DatabaseHealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	database := &Database{pool: pool}
	if err := database.Ping(ctx); err != nil {
		database.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	logger.Info("database connected", "max_conns", poolConfig.MaxConns, "min_conns", poolConfig.MinConns)
	return database, nil
}

// Pool returns the managed pool for repository construction.
func (database *Database) Pool() *pgxpool.Pool { return database.pool }

// Ping checks database connectivity using the supplied context.
func (database *Database) Ping(ctx context.Context) error {
	if database == nil || database.pool == nil {
		return nil
	}
	return database.pool.Ping(ctx)
}

// Close releases all connections in the pool.
func (database *Database) Close() {
	if database != nil && database.pool != nil {
		database.pool.Close()
	}
}

// WithTransaction executes work in a transaction and rolls back on failure.
func (database *Database) WithTransaction(ctx context.Context, work func(pgx.Tx) error) error {
	transaction, err := database.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if err := work(transaction); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// HealthCheck runs an explicit connectivity check for future readiness endpoints.
func (database *Database) HealthCheck(ctx context.Context) error {
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return database.Ping(checkCtx)
}
