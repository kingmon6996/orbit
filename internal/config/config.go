package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains the runtime settings for Orbit.
type Config struct {
	AppName                   string
	Environment               string
	Host                      string
	Port                      int
	LogLevel                  slog.Level
	ReadTimeout               time.Duration
	WriteTimeout              time.Duration
	IdleTimeout               time.Duration
	ReadHeaderTimeout         time.Duration
	ShutdownTimeout           time.Duration
	DatabaseEnabled           bool
	DatabaseURL               string
	DatabaseMaxConns          int32
	DatabaseMinConns          int32
	DatabaseMaxConnLifetime   time.Duration
	DatabaseMaxConnIdleTime   time.Duration
	DatabaseHealthCheckPeriod time.Duration
}

// Load reads and validates Orbit's environment-based configuration.
func Load() (Config, error) {
	port, err := integer("ORBIT_PORT", "8080")
	if err != nil {
		return Config{}, err
	}

	logLevel, err := level(value("ORBIT_LOG_LEVEL", "info"))
	if err != nil {
		return Config{}, err
	}

	readTimeout, err := duration("ORBIT_READ_TIMEOUT", "15s")
	if err != nil {
		return Config{}, err
	}
	writeTimeout, err := duration("ORBIT_WRITE_TIMEOUT", "15s")
	if err != nil {
		return Config{}, err
	}
	idleTimeout, err := duration("ORBIT_IDLE_TIMEOUT", "60s")
	if err != nil {
		return Config{}, err
	}
	readHeaderTimeout, err := duration("ORBIT_READ_HEADER_TIMEOUT", "5s")
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := duration("ORBIT_SHUTDOWN_TIMEOUT", "10s")
	if err != nil {
		return Config{}, err
	}
	databaseEnabled, err := boolean("DATABASE_ENABLED", "false")
	if err != nil {
		return Config{}, err
	}
	databaseMaxConns, err := positiveInteger("DATABASE_MAX_CONNS", "20")
	if err != nil {
		return Config{}, err
	}
	databaseMinConns, err := positiveInteger("DATABASE_MIN_CONNS", "2")
	if err != nil {
		return Config{}, err
	}
	databaseMaxConnLifetime, err := duration("DATABASE_MAX_CONN_LIFETIME", "30m")
	if err != nil {
		return Config{}, err
	}
	databaseMaxConnIdleTime, err := duration("DATABASE_MAX_CONN_IDLE_TIME", "10m")
	if err != nil {
		return Config{}, err
	}
	databaseHealthCheckPeriod, err := duration("DATABASE_HEALTH_CHECK_PERIOD", "30s")
	if err != nil {
		return Config{}, err
	}

	config := Config{
		AppName:                   value("APP_NAME", "orbit"),
		Environment:               value("APP_ENV", "development"),
		Host:                      value("ORBIT_HOST", "0.0.0.0"),
		Port:                      port,
		LogLevel:                  logLevel,
		ReadTimeout:               readTimeout,
		WriteTimeout:              writeTimeout,
		IdleTimeout:               idleTimeout,
		ReadHeaderTimeout:         readHeaderTimeout,
		ShutdownTimeout:           shutdownTimeout,
		DatabaseEnabled:           databaseEnabled,
		DatabaseURL:               value("DATABASE_URL", ""),
		DatabaseMaxConns:          int32(databaseMaxConns),
		DatabaseMinConns:          int32(databaseMinConns),
		DatabaseMaxConnLifetime:   databaseMaxConnLifetime,
		DatabaseMaxConnIdleTime:   databaseMaxConnIdleTime,
		DatabaseHealthCheckPeriod: databaseHealthCheckPeriod,
	}
	if config.Host == "" {
		return Config{}, fmt.Errorf("invalid ORBIT_HOST: value must not be empty")
	}
	if config.AppName == "" {
		return Config{}, fmt.Errorf("invalid APP_NAME: value must not be empty")
	}
	if config.Environment != "development" && config.Environment != "staging" && config.Environment != "production" {
		return Config{}, fmt.Errorf("invalid APP_ENV: %q", config.Environment)
	}
	if config.DatabaseMinConns > config.DatabaseMaxConns {
		return Config{}, fmt.Errorf("invalid database pool configuration: DATABASE_MIN_CONNS cannot exceed DATABASE_MAX_CONNS")
	}
	if config.DatabaseEnabled && config.DatabaseURL == "" {
		return Config{}, fmt.Errorf("invalid DATABASE_URL: required when DATABASE_ENABLED=true")
	}
	return config, nil
}

func value(key, fallback string) string {
	if configured, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(configured)
	}
	return fallback
}

func integer(key, fallback string) (int, error) {
	parsed, err := strconv.Atoi(value(key, fallback))
	if err != nil || parsed < 1 || parsed > 65535 {
		return 0, fmt.Errorf("invalid %s: must be a TCP port from 1 to 65535", key)
	}
	return parsed, nil
}

func positiveInteger(key, fallback string) (int, error) {
	parsed, err := strconv.Atoi(value(key, fallback))
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid %s: must be a positive integer", key)
	}
	return parsed, nil
}

func boolean(key, fallback string) (bool, error) {
	parsed, err := strconv.ParseBool(value(key, fallback))
	if err != nil {
		return false, fmt.Errorf("invalid %s: must be true or false", key)
	}
	return parsed, nil
}

func duration(key, fallback string) (time.Duration, error) {
	parsed, err := time.ParseDuration(value(key, fallback))
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("invalid %s: must be a non-negative duration", key)
	}
	return parsed, nil
}

func level(raw string) (slog.Level, error) {
	var parsed slog.Level
	if err := parsed.UnmarshalText([]byte(strings.ToLower(raw))); err != nil {
		return 0, fmt.Errorf("invalid ORBIT_LOG_LEVEL: %q (expected debug, info, warn, or error)", raw)
	}
	return parsed, nil
}
