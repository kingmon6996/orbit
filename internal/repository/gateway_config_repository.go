package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kingmon6996/orbit/internal/database"
)

// GatewayConfig is a persisted key-value gateway configuration entry.
type GatewayConfig struct {
	ID        string
	Key       string
	Value     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GatewayConfigRepository stores GatewayConfig records in PostgreSQL.
type GatewayConfigRepository struct {
	pool *pgxpool.Pool
}

// NewGatewayConfigRepository creates a repository backed by pool.
func NewGatewayConfigRepository(pool *pgxpool.Pool) *GatewayConfigRepository {
	return &GatewayConfigRepository{pool: pool}
}

// Create inserts a new configuration entry.
func (repository *GatewayConfigRepository) Create(ctx context.Context, key, value string) (GatewayConfig, error) {
	if key == "" {
		return GatewayConfig{}, fmt.Errorf("key must not be empty")
	}
	identifier, err := newUUID()
	if err != nil {
		return GatewayConfig{}, fmt.Errorf("generate configuration ID: %w", err)
	}
	var configuration GatewayConfig
	err = repository.pool.QueryRow(ctx, `INSERT INTO gateway_configs (id, key, value) VALUES ($1, $2, $3) RETURNING id, key, value, created_at, updated_at`, identifier, key, value).Scan(&configuration.ID, &configuration.Key, &configuration.Value, &configuration.CreatedAt, &configuration.UpdatedAt)
	if err != nil {
		return GatewayConfig{}, mapError(err)
	}
	return configuration, nil
}

// GetByID retrieves a configuration entry by UUID.
func (repository *GatewayConfigRepository) GetByID(ctx context.Context, id string) (GatewayConfig, error) {
	return repository.get(ctx, `SELECT id, key, value, created_at, updated_at FROM gateway_configs WHERE id = $1`, id)
}

// GetByKey retrieves a configuration entry by its unique key.
func (repository *GatewayConfigRepository) GetByKey(ctx context.Context, key string) (GatewayConfig, error) {
	return repository.get(ctx, `SELECT id, key, value, created_at, updated_at FROM gateway_configs WHERE key = $1`, key)
}

// Update changes the value for an existing configuration entry.
func (repository *GatewayConfigRepository) Update(ctx context.Context, id, value string) (GatewayConfig, error) {
	return repository.get(ctx, `UPDATE gateway_configs SET value = $2, updated_at = NOW() WHERE id = $1 RETURNING id, key, value, created_at, updated_at`, id, value)
}

// Delete removes an entry by UUID.
func (repository *GatewayConfigRepository) Delete(ctx context.Context, id string) error {
	result, err := repository.pool.Exec(ctx, `DELETE FROM gateway_configs WHERE id = $1`, id)
	if err != nil {
		return mapError(err)
	}
	if result.RowsAffected() == 0 {
		return database.ErrNotFound
	}
	return nil
}

// List returns all entries ordered by key.
func (repository *GatewayConfigRepository) List(ctx context.Context) ([]GatewayConfig, error) {
	rows, err := repository.pool.Query(ctx, `SELECT id, key, value, created_at, updated_at FROM gateway_configs ORDER BY key`)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	entries := make([]GatewayConfig, 0)
	for rows.Next() {
		var entry GatewayConfig
		if err := rows.Scan(&entry.ID, &entry.Key, &entry.Value, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			return nil, mapError(err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}
	return entries, nil
}

func (repository *GatewayConfigRepository) get(ctx context.Context, query string, args ...any) (GatewayConfig, error) {
	var configuration GatewayConfig
	err := repository.pool.QueryRow(ctx, query, args...).Scan(&configuration.ID, &configuration.Key, &configuration.Value, &configuration.CreatedAt, &configuration.UpdatedAt)
	if err != nil {
		return GatewayConfig{}, mapError(err)
	}
	return configuration, nil
}

func mapError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return database.ErrNotFound
	}
	var constraintError *pgconn.PgError
	if errors.As(err, &constraintError) && constraintError.Code == "23505" {
		return fmt.Errorf("%w: %v", database.ErrConflict, err)
	}
	return fmt.Errorf("%w: %v", database.ErrDatabase, err)
}

func newUUID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(bytes)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32], nil
}
