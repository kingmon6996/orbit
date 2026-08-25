package repository

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kingmon6996/orbit/internal/config"
	"github.com/kingmon6996/orbit/internal/database"
)

func TestGatewayConfigRepositoryPostgreSQL(t *testing.T) {
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
	repository := NewGatewayConfigRepository(databaseConnection.Pool())
	key := "test." + newTestSuffix()
	created, err := repository.Create(ctx, key, "one")
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Delete(ctx, created.ID)
	byID, err := repository.GetByID(ctx, created.ID)
	if err != nil || byID.Key != key {
		t.Fatalf("GetByID() = %+v, error = %v", byID, err)
	}
	byKey, err := repository.GetByKey(ctx, key)
	if err != nil || byKey.ID != created.ID {
		t.Fatalf("GetByKey() = %+v, error = %v", byKey, err)
	}
	updated, err := repository.Update(ctx, created.ID, "two")
	if err != nil || updated.Value != "two" {
		t.Fatalf("Update() = %+v, error = %v", updated, err)
	}
	entries, err := repository.List(ctx)
	if err != nil || len(entries) == 0 {
		t.Fatalf("List() = %d, error = %v", len(entries), err)
	}
	if _, err := repository.Create(ctx, key, "duplicate"); !errors.Is(err, database.ErrConflict) {
		t.Fatalf("duplicate error = %v", err)
	}
	if err := repository.Delete(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.GetByID(ctx, created.ID); !errors.Is(err, database.ErrNotFound) {
		t.Fatalf("missing error = %v", err)
	}
}

func newTestSuffix() string {
	return time.Now().Format("20060102150405.000000000")
}
