package logging

import (
	"log/slog"
	"os"

	"github.com/kingmon6996/orbit/internal/config"
)

// New creates an environment-appropriate structured logger.
func New(configuration config.Config) *slog.Logger {
	options := &slog.HandlerOptions{Level: configuration.LogLevel}
	if configuration.Environment == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, options))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, options))
}
