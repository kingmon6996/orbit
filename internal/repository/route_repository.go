package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kingmon6996/orbit/internal/database"
	"github.com/kingmon6996/orbit/internal/routing"
)

// RouteRepository stores route definitions in PostgreSQL.
type RouteRepository struct{ pool *pgxpool.Pool }

// NewRouteRepository creates a route repository backed by pool.
func NewRouteRepository(pool *pgxpool.Pool) *RouteRepository { return &RouteRepository{pool: pool} }

// Create inserts a validated route definition.
func (repository *RouteRepository) Create(ctx context.Context, route routing.Route) (routing.Route, error) {
	if err := route.Validate(); err != nil {
		return routing.Route{}, err
	}
	if route.ID == "" {
		var err error
		route.ID, err = newIdentifier()
		if err != nil {
			return routing.Route{}, err
		}
	}
	return repository.scanOne(ctx, `INSERT INTO routes (id, name, path, method, service_id, enabled, strip_prefix, rewrite_path, timeout) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id, name, path, method, service_id, enabled, strip_prefix, rewrite_path, timeout, created_at, updated_at`, route.ID, route.Name, route.Path, route.Method, route.ServiceID, route.Enabled, route.StripPrefix, route.RewritePath, route.Timeout.Nanoseconds())
}

// GetByID retrieves a route by ID.
func (repository *RouteRepository) GetByID(ctx context.Context, id string) (routing.Route, error) {
	return repository.scanOne(ctx, `SELECT id, name, path, method, service_id, enabled, strip_prefix, rewrite_path, timeout, created_at, updated_at FROM routes WHERE id = $1`, id)
}

// Update changes a route definition.
func (repository *RouteRepository) Update(ctx context.Context, route routing.Route) (routing.Route, error) {
	if err := route.Validate(); err != nil {
		return routing.Route{}, err
	}
	return repository.scanOne(ctx, `UPDATE routes SET name = $2, path = $3, method = $4, service_id = $5, enabled = $6, strip_prefix = $7, rewrite_path = $8, timeout = $9, updated_at = NOW() WHERE id = $1 RETURNING id, name, path, method, service_id, enabled, strip_prefix, rewrite_path, timeout, created_at, updated_at`, route.ID, route.Name, route.Path, route.Method, route.ServiceID, route.Enabled, route.StripPrefix, route.RewritePath, route.Timeout.Nanoseconds())
}

// Delete removes a route.
func (repository *RouteRepository) Delete(ctx context.Context, id string) error {
	result, err := repository.pool.Exec(ctx, `DELETE FROM routes WHERE id = $1`, id)
	if err != nil {
		return mapServiceError(err)
	}
	if result.RowsAffected() == 0 {
		return database.ErrNotFound
	}
	return nil
}

// List returns all routes.
func (repository *RouteRepository) List(ctx context.Context) ([]routing.Route, error) {
	return repository.list(ctx, `SELECT id, name, path, method, service_id, enabled, strip_prefix, rewrite_path, timeout, created_at, updated_at FROM routes ORDER BY method, path`)
}

// ListEnabled returns all enabled routes for snapshot loading.
func (repository *RouteRepository) ListEnabled(ctx context.Context) ([]routing.Route, error) {
	return repository.list(ctx, `SELECT id, name, path, method, service_id, enabled, strip_prefix, rewrite_path, timeout, created_at, updated_at FROM routes WHERE enabled = TRUE ORDER BY method, path`)
}

func (repository *RouteRepository) list(ctx context.Context, query string) ([]routing.Route, error) {
	rows, err := repository.pool.Query(ctx, query)
	if err != nil {
		return nil, mapServiceError(err)
	}
	defer rows.Close()
	routes := make([]routing.Route, 0)
	for rows.Next() {
		route, err := scanRoute(rows)
		if err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}
	if err := rows.Err(); err != nil {
		return nil, mapServiceError(err)
	}
	return routes, nil
}

func (repository *RouteRepository) scanOne(ctx context.Context, query string, args ...any) (routing.Route, error) {
	return scanRoute(repository.pool.QueryRow(ctx, query, args...))
}

type rowScanner interface{ Scan(...any) error }

func scanRoute(row rowScanner) (routing.Route, error) {
	var route routing.Route
	var timeout int64
	if err := row.Scan(&route.ID, &route.Name, &route.Path, &route.Method, &route.ServiceID, &route.Enabled, &route.StripPrefix, &route.RewritePath, &timeout, &route.CreatedAt, &route.UpdatedAt); err != nil {
		return routing.Route{}, fmt.Errorf("%w: %v", database.ErrDatabase, err)
	}
	route.Timeout = time.Duration(timeout)
	return route, nil
}
