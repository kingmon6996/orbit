package routing

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Service identifies a backend service referenced by routes.
type Service struct {
	ID          string
	Name        string
	Description string
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Route describes a persisted gateway route.
type Route struct {
	ID          string
	Name        string
	Path        string
	Method      string
	ServiceID   string
	Enabled     bool
	StripPrefix bool
	RewritePath string
	Timeout     time.Duration
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate normalizes and validates a route before it enters a snapshot.
func (route *Route) Validate() error {
	route.Method = strings.ToUpper(strings.TrimSpace(route.Method))
	route.Path = normalizePath(route.Path)
	route.Name = strings.TrimSpace(route.Name)
	if route.Name == "" {
		return fmt.Errorf("route name must not be empty")
	}
	if route.ServiceID == "" {
		return fmt.Errorf("route %q: service ID must not be empty", route.Name)
	}
	if !validMethods[route.Method] {
		return fmt.Errorf("route %q: invalid HTTP method %q", route.Name, route.Method)
	}
	if route.Path == "" || !strings.HasPrefix(route.Path, "/") {
		return fmt.Errorf("route %q: path must begin with /", route.Name)
	}
	if route.Timeout < 0 {
		return fmt.Errorf("route %q: timeout must not be negative", route.Name)
	}
	segments := strings.Split(strings.TrimPrefix(route.Path, "/"), "/")
	for index, segment := range segments {
		if segment == "" {
			return fmt.Errorf("route %q: path contains an empty segment", route.Name)
		}
		if segment == "*" && index != len(segments)-1 {
			return fmt.Errorf("route %q: wildcard must be the final segment", route.Name)
		}
		if strings.Contains(segment, "*") && segment != "*" {
			return fmt.Errorf("route %q: wildcard must be a complete final segment", route.Name)
		}
		if strings.ContainsAny(segment, "{}") && (len(segment) < 3 || segment[0] != '{' || segment[len(segment)-1] != '}' || strings.ContainsAny(segment[1:len(segment)-1], "{}") || segment[1:len(segment)-1] == "") {
			return fmt.Errorf("route %q: malformed parameter", route.Name)
		}
	}
	return nil
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return path
	}
	return strings.TrimRight(path, "/")
}

var validMethods = map[string]bool{http.MethodGet: true, http.MethodPost: true, http.MethodPut: true, http.MethodPatch: true, http.MethodDelete: true, http.MethodOptions: true, http.MethodHead: true}
