package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kingmon6996/orbit/internal/config"
	"github.com/kingmon6996/orbit/internal/database"
	"github.com/kingmon6996/orbit/internal/logging"
	"github.com/kingmon6996/orbit/internal/server"
	"github.com/kingmon6996/orbit/internal/version"
)

func main() {
	configuration, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		return
	}
	logger := logging.New(configuration)
	logger.Info("starting Orbit", "application", configuration.AppName, "version", version.Version, "environment", configuration.Environment, "host", configuration.Host, "port", configuration.Port)
	applicationContext := context.Background()
	databaseConnection, err := database.New(applicationContext, configuration, logger)
	if err != nil {
		logger.Error("failed to initialize database", "error", err)
		return
	}
	httpServer := server.New(configuration, server.NewRouterWithVersion(configuration.AppName, version.Version, logger))
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- httpServer.Start() }()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serverErrors:
		if err != nil {
			logger.Error("failed to start HTTP server", "error", err)
		}
	case signal := <-shutdownSignal:
		logger.Info("shutdown signal received", "signal", signal.String())
		ctx, cancel := context.WithTimeout(context.Background(), configuration.ShutdownTimeout)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("graceful shutdown failed", "error", err)
		} else {
			logger.Info("server stopped")
		}
	}
	if databaseConnection != nil {
		logger.Info("closing database pool")
		databaseConnection.Close()
	}
}
