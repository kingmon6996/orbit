package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kingmon6996/orbit/internal/database"
	"github.com/kingmon6996/orbit/internal/routing"
)

// ServiceRepository stores backend service definitions in PostgreSQL.
type ServiceRepository struct{ pool *pgxpool.Pool }

// NewServiceRepository creates a service repository backed by pool.
func NewServiceRepository(pool *pgxpool.Pool) *ServiceRepository {
	return &ServiceRepository{pool: pool}
}

// Create inserts a service.
func (repository *ServiceRepository) Create(ctx context.Context, name, description string, enabled bool) (routing.Service, error) {
	if name == "" {
		return routing.Service{}, fmt.Errorf("service name must not be empty")
	}
	id, err := newIdentifier()
	if err != nil {
		return routing.Service{}, fmt.Errorf("generate service ID: %w", err)
	}
	return repository.scanOne(ctx, `INSERT INTO services (id, name, description, enabled) VALUES ($1, $2, $3, $4) RETURNING id, name, description, enabled, created_at, updated_at`, id, name, description, enabled)
}

// GetByID retrieves a service by ID.
func (repository *ServiceRepository) GetByID(ctx context.Context, id string) (routing.Service, error) {
	return repository.scanOne(ctx, `SELECT id, name, description, enabled, created_at, updated_at FROM services WHERE id = $1`, id)
}

// GetByName retrieves a service by unique name.
func (repository *ServiceRepository) GetByName(ctx context.Context, name string) (routing.Service, error) {
	return repository.scanOne(ctx, `SELECT id, name, description, enabled, created_at, updated_at FROM services WHERE name = $1`, name)
}

// Update changes a service's metadata.
func (repository *ServiceRepository) Update(ctx context.Context, id, name, description string, enabled bool) (routing.Service, error) {
	return repository.scanOne(ctx, `UPDATE services SET name = $2, description = $3, enabled = $4, updated_at = NOW() WHERE id = $1 RETURNING id, name, description, enabled, created_at, updated_at`, id, name, description, enabled)
}

// Delete removes a service when no routes reference it.
func (repository *ServiceRepository) Delete(ctx context.Context, id string) error {
	result, err := repository.pool.Exec(ctx, `DELETE FROM services WHERE id = $1`, id)
	if err != nil {
		return mapServiceError(err)
	}
	if result.RowsAffected() == 0 {
		return database.ErrNotFound
	}
	return nil
}

// List returns all services.
func (repository *ServiceRepository) List(ctx context.Context) ([]routing.Service, error) {
	return repository.list(ctx, `SELECT id, name, description, enabled, created_at, updated_at FROM services ORDER BY name`)
}

// ListEnabled returns enabled services.
func (repository *ServiceRepository) ListEnabled(ctx context.Context) ([]routing.Service, error) {
	return repository.list(ctx, `SELECT id, name, description, enabled, created_at, updated_at FROM services WHERE enabled = TRUE ORDER BY name`)
}

func (repository *ServiceRepository) list(ctx context.Context, query string) ([]routing.Service, error) {
	rows, err := repository.pool.Query(ctx, query)
	if err != nil {
		return nil, mapServiceError(err)
	}
	defer rows.Close()
	services := make([]routing.Service, 0)
	for rows.Next() {
		var service routing.Service
		if err := rows.Scan(&service.ID, &service.Name, &service.Description, &service.Enabled, &service.CreatedAt, &service.UpdatedAt); err != nil {
			return nil, mapServiceError(err)
		}
		services = append(services, service)
	}
	if err := rows.Err(); err != nil {
		return nil, mapServiceError(err)
	}
	return services, nil
}

func (repository *ServiceRepository) scanOne(ctx context.Context, query string, args ...any) (routing.Service, error) {
	var service routing.Service
	err := repository.pool.QueryRow(ctx, query, args...).Scan(&service.ID, &service.Name, &service.Description, &service.Enabled, &service.CreatedAt, &service.UpdatedAt)
	if err != nil {
		return routing.Service{}, mapServiceError(err)
	}
	return service, nil
}

func mapServiceError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return database.ErrNotFound
	}
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" {
		return fmt.Errorf("%w: %v", database.ErrConflict, err)
	}
	return fmt.Errorf("%w: %v", database.ErrDatabase, err)
}

func newIdentifier() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(bytes)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32], nil
}
