package repository

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kingmon6996/orbit/internal/config"
	"github.com/kingmon6996/orbit/internal/database"
	"github.com/kingmon6996/orbit/internal/routing"
)

func TestServiceRouteSnapshotPostgreSQL(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	databaseConnection, err := database.New(ctx, config.Config{DatabaseEnabled: true, DatabaseURL: url, DatabaseMaxConns: 4, DatabaseMinConns: 1, DatabaseMaxConnLifetime: time.Hour, DatabaseMaxConnIdleTime: time.Minute, DatabaseHealthCheckPeriod: time.Minute}, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer databaseConnection.Close()
	if err := database.MigrateUp(ctx, databaseConnection.Pool()); err != nil {
		t.Fatal(err)
	}
	serviceRepository := NewServiceRepository(databaseConnection.Pool())
	routeRepository := NewRouteRepository(databaseConnection.Pool())
	service, err := serviceRepository.Create(ctx, "test-service-"+newTestSuffix(), "integration test", true)
	if err != nil {
		t.Fatal(err)
	}
	defer serviceRepository.Delete(ctx, service.ID)
	route, err := routeRepository.Create(ctx, routing.Route{Name: "test-route-" + newTestSuffix(), Path: "/integration/{id}", Method: "get", ServiceID: service.ID, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	defer routeRepository.Delete(ctx, route.ID)
	services, err := serviceRepository.ListEnabled(ctx)
	if err != nil {
		t.Fatal(err)
	}
	routes, err := routeRepository.ListEnabled(ctx)
	if err != nil {
		t.Fatal(err)
	}
	registry := routing.NewRegistry()
	if err := registry.Reload(services, routes); err != nil {
		t.Fatal(err)
	}
	match, ok, _ := registry.Get().Match("GET", "/integration/123")
	if !ok || match.ServiceID != service.ID || match.PathParameters["id"] != "123" {
		t.Fatalf("match = %+v, ok = %v", match, ok)
	}
}
