package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kingmon6996/orbit/internal/config"
)

func TestRequestIDMiddleware(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if RequestIDFromContext(request.Context()) == "" {
			t.Error("request ID missing from context")
		}
	}))
	for _, requestID := range []string{"", "123e4567-e89b-12d3-a456-426614174000"} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("X-Request-ID", requestID)
		record := httptest.NewRecorder()
		handler.ServeHTTP(record, request)
		if !validRequestID(record.Header().Get("X-Request-ID")) {
			t.Fatalf("invalid response request ID: %q", record.Header().Get("X-Request-ID"))
		}
		if requestID != "" && record.Header().Get("X-Request-ID") != requestID {
			t.Errorf("request ID was not preserved")
		}
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := RecoveryMiddleware(logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("secret panic") }))
	record := httptest.NewRecorder()
	handler.ServeHTTP(record, httptest.NewRequest(http.MethodGet, "/", nil))
	if record.Code != http.StatusInternalServerError || strings.Contains(record.Body.String(), "secret panic") {
		t.Fatalf("unexpected recovery response: %d %s", record.Code, record.Body.String())
	}
}

func TestRequestLoggingMiddlewareRecordsCompletedResponse(t *testing.T) {
	var logs strings.Builder
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	handler := RequestLoggingMiddleware(logger, http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusCreated)
	}))
	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/items", nil)
	request = request.WithContext(context.WithValue(request.Context(), requestIDKey{}, "123e4567-e89b-12d3-a456-426614174000"))
	handler.ServeHTTP(record, request)
	if record.Code != http.StatusCreated || !strings.Contains(logs.String(), "status=201") || !strings.Contains(logs.String(), "path=/items") || !strings.Contains(logs.String(), "duration=") {
		t.Fatalf("response or log fields missing: status=%d logs=%s", record.Code, logs.String())
	}
}

func TestNewRouterEndpoints(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for path, expected := range map[string]string{"/": "\"name\":\"orbit\"", "/health": "\"status\":\"ok\""} {
		record := httptest.NewRecorder()
		NewRouter("orbit", logger).ServeHTTP(record, httptest.NewRequest(http.MethodGet, path, nil))
		if record.Code != http.StatusOK || !strings.Contains(record.Body.String(), expected) {
			t.Errorf("%s: got %d %s", path, record.Code, record.Body.String())
		}
	}
}

func TestServerShutdown(t *testing.T) {
	server := New(config.Config{Host: "127.0.0.1", Port: 0}, http.NewServeMux())
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}
