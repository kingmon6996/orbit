package routing

import (
	"encoding/json"
	"net/http"
	"sort"
)

// Handler matches requests against the registry and returns Module 3 results.
type Handler struct {
	registry *Registry
	fallback http.Handler
}

// NewHandler creates a route matcher with a fallback for non-gateway endpoints.
func NewHandler(registry *Registry, fallback http.Handler) http.Handler {
	return &Handler{registry: registry, fallback: fallback}
}

// ServeHTTP preserves health/root fallbacks and handles configured routes.
func (handler *Handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.URL.Path == "/" || request.URL.Path == "/health" {
		handler.fallback.ServeHTTP(response, request)
		return
	}
	match, ok, allowed := handler.registry.Get().Match(request.Method, request.URL.Path)
	if !ok {
		if len(allowed) > 0 {
			response.Header().Set("Allow", joinMethods(allowed))
			writeError(response, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		writeError(response, http.StatusNotFound, "ROUTE_NOT_FOUND", "route not found")
		return
	}
	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(map[string]any{"matched": true, "route_id": match.RouteID, "service_id": match.ServiceID, "path": request.URL.Path, "parameters": match.PathParameters})
}

func joinMethods(methods []string) string {
	sort.Strings(methods)
	result := ""
	for index, method := range methods {
		if index > 0 {
			result += ", "
		}
		result += method
	}
	return result
}

func writeError(response http.ResponseWriter, status int, code, message string) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
}
