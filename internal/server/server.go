package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kingmon6996/orbit/internal/config"
	"github.com/kingmon6996/orbit/internal/health"
	"github.com/kingmon6996/orbit/internal/version"
)

type requestIDKey struct{}

// RequestIDFromContext returns the request ID assigned by middleware.
func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}

// RequestIDMiddleware assigns and returns a request ID for every request.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestID := request.Header.Get("X-Request-ID")
		if !validRequestID(requestID) {
			requestID = newRequestID()
		}
		response.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(response, request.WithContext(context.WithValue(request.Context(), requestIDKey{}, requestID)))
	})
}

// RequestLoggingMiddleware logs the completed request and its result.
func RequestLoggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: response, status: http.StatusOK}
		next.ServeHTTP(recorder, request)
		logger.Info("http request", "method", request.Method, "path", request.URL.Path, "status", recorder.status, "duration", time.Since(started), "request_id", RequestIDFromContext(request.Context()), "remote_address", request.RemoteAddr, "user_agent", request.UserAgent())
	})
}

// RecoveryMiddleware converts handler panics into an internal server error.
func RecoveryMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("handler panic recovered", "error", recovered, "request_id", RequestIDFromContext(request.Context()))
				writeJSONError(response)
			}
		}()
		next.ServeHTTP(response, request)
	})
}

// NewRouter builds the Module 1 HTTP routes and middleware chain.
func NewRouter(appName string, logger *slog.Logger) http.Handler {
	return NewRouterWithVersion(appName, version.Version, logger)
}

// NewRouterWithVersion builds routes using the supplied application version.
func NewRouterWithVersion(appName, appVersion string, logger *slog.Logger) http.Handler {
	return WithMiddleware(NewBaseRouterWithVersion(appName, appVersion), logger)
}

// NewBaseRouterWithVersion builds the root and health endpoints without middleware.
func NewBaseRouterWithVersion(appName, appVersion string) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/health", health.Handler())
	mux.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			response.Header().Set("Allow", http.MethodGet)
			response.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]string{"name": appName, "version": appVersion, "status": "running"})
	})
	return mux
}

// WithMiddleware applies Orbit's request infrastructure to a handler.
func WithMiddleware(handler http.Handler, logger *slog.Logger) http.Handler {
	return RequestIDMiddleware(RequestLoggingMiddleware(logger, RecoveryMiddleware(logger, handler)))
}

// Server owns Orbit's HTTP server lifecycle.
type Server struct {
	httpServer *http.Server
}

// New creates a configured HTTP server.
func New(configuration config.Config, handler http.Handler) *Server {
	return &Server{httpServer: &http.Server{Addr: fmt.Sprintf("%s:%d", configuration.Host, configuration.Port), Handler: handler, ReadTimeout: configuration.ReadTimeout, WriteTimeout: configuration.WriteTimeout, IdleTimeout: configuration.IdleTimeout, ReadHeaderTimeout: configuration.ReadHeaderTimeout}}
}

// Start begins serving and returns when the listener stops.
func (server *Server) Start() error {
	err := server.httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Shutdown gracefully stops the server within the supplied context deadline.
func (server *Server) Shutdown(ctx context.Context) error {
	return server.httpServer.Shutdown(ctx)
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (recorder *statusRecorder) WriteHeader(status int) {
	if !recorder.wroteHeader {
		recorder.status = status
		recorder.wroteHeader = true
	}
	recorder.ResponseWriter.WriteHeader(status)
}

func (recorder *statusRecorder) Write(body []byte) (int, error) {
	if !recorder.wroteHeader {
		recorder.WriteHeader(http.StatusOK)
	}
	return recorder.ResponseWriter.Write(body)
}

func newRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return hex.EncodeToString(bytes)
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s", hex.EncodeToString(bytes[0:4]), hex.EncodeToString(bytes[4:6]), hex.EncodeToString(bytes[6:8]), hex.EncodeToString(bytes[8:10]), hex.EncodeToString(bytes[10:16]))
}

func validRequestID(requestID string) bool {
	if len(requestID) != 36 || strings.Count(requestID, "-") != 4 {
		return false
	}
	for index, character := range requestID {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if character != '-' {
				return false
			}
		} else if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f') || (character >= 'A' && character <= 'F')) {
			return false
		}
	}
	return true
}

func writeJSONError(response http.ResponseWriter) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(response).Encode(map[string]any{"error": map[string]string{"code": "INTERNAL_SERVER_ERROR", "message": "internal server error"}})
}
