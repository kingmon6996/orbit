package config

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	clearConfigEnvironment(t)
	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.AppName != "orbit" || got.Environment != "development" || got.Host != "0.0.0.0" || got.Port != 8080 || got.LogLevel != slog.LevelInfo || got.ReadTimeout != 15*time.Second || got.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("unexpected defaults: %+v", got)
	}
}

func TestLoadConfiguredValues(t *testing.T) {
	clearConfigEnvironment(t)
	for key, value := range map[string]string{"APP_NAME": "test-orbit", "APP_ENV": "staging", "ORBIT_HOST": "127.0.0.1", "ORBIT_PORT": "9090", "ORBIT_LOG_LEVEL": "debug", "ORBIT_READ_TIMEOUT": "1s", "ORBIT_WRITE_TIMEOUT": "2s", "ORBIT_IDLE_TIMEOUT": "3s", "ORBIT_READ_HEADER_TIMEOUT": "4s", "ORBIT_SHUTDOWN_TIMEOUT": "5s"} {
		t.Setenv(key, value)
	}
	got, err := Load()
	if err != nil || got.Port != 9090 || got.Environment != "staging" || got.LogLevel != slog.LevelDebug || got.ShutdownTimeout != 5*time.Second {
		t.Fatalf("Load() = %+v, error = %v", got, err)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	for key, value := range map[string]string{"ORBIT_PORT": "70000", "ORBIT_READ_TIMEOUT": "-1s", "ORBIT_LOG_LEVEL": "verbose", "APP_ENV": "test"} {
		clearConfigEnvironment(t)
		t.Setenv(key, value)
		if _, err := Load(); err == nil {
			t.Errorf("Load() accepted invalid %s", key)
		}
	}
}

func clearConfigEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{"APP_NAME", "APP_ENV", "ORBIT_HOST", "ORBIT_PORT", "ORBIT_LOG_LEVEL", "ORBIT_READ_TIMEOUT", "ORBIT_WRITE_TIMEOUT", "ORBIT_IDLE_TIMEOUT", "ORBIT_READ_HEADER_TIMEOUT", "ORBIT_SHUTDOWN_TIMEOUT"} {
		if value, ok := os.LookupEnv(key); ok {
			t.Setenv(key, "")
			t.Cleanup(func() { os.Setenv(key, value) })
		}
		os.Unsetenv(key)
	}
}
