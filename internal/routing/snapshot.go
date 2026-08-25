package routing

import (
	"fmt"
	"strings"
)

// RouteMatch is the result of matching one request path and method.
type RouteMatch struct {
	RouteID        string
	ServiceID      string
	Route          Route
	Service        Service
	PathParameters map[string]string
	MatchedPath    string
}

type compiledRoute struct {
	route    Route
	service  Service
	segments []string
	kind     int
}

// RoutingSnapshot is immutable after construction and publication.
type RoutingSnapshot struct {
	routesByMethod map[string][]compiledRoute
	paths          map[string]map[string]struct{}
}

// BuildSnapshot validates services/routes and creates an immutable matcher.
func BuildSnapshot(services []Service, routes []Route) (*RoutingSnapshot, error) {
	serviceByID := make(map[string]Service, len(services))
	for _, service := range services {
		if service.ID == "" || service.Name == "" {
			return nil, fmt.Errorf("invalid service: ID and name are required")
		}
		if _, exists := serviceByID[service.ID]; exists {
			return nil, fmt.Errorf("duplicate service ID %q", service.ID)
		}
		serviceByID[service.ID] = service
	}
	byMethod := make(map[string][]compiledRoute)
	paths := make(map[string]map[string]struct{})
	conflicts := make(map[string]struct{})
	for _, route := range routes {
		if !route.Enabled {
			continue
		}
		if err := route.Validate(); err != nil {
			return nil, err
		}
		service, exists := serviceByID[route.ServiceID]
		if !exists || !service.Enabled {
			return nil, fmt.Errorf("route %q references unavailable service %q", route.Name, route.ServiceID)
		}
		segments := strings.Split(strings.TrimPrefix(route.Path, "/"), "/")
		kind := 0
		if segments[len(segments)-1] == "*" {
			kind = 2
		} else {
			for _, segment := range segments {
				if strings.HasPrefix(segment, "{") {
					kind = 1
					break
				}
			}
		}
		conflictKey := route.Method + " " + route.Path
		if _, exists := conflicts[conflictKey]; exists {
			return nil, fmt.Errorf("conflicting routes: %s", conflictKey)
		}
		conflicts[conflictKey] = struct{}{}
		shapeKey := route.Method + " " + routeShape(segments)
		if _, exists := conflicts[shapeKey]; exists {
			return nil, fmt.Errorf("ambiguous routes: %s", conflictKey)
		}
		conflicts[shapeKey] = struct{}{}
		byMethod[route.Method] = append(byMethod[route.Method], compiledRoute{route: route, service: service, segments: segments, kind: kind})
		if paths[route.Path] == nil {
			paths[route.Path] = make(map[string]struct{})
		}
		paths[route.Path][route.Method] = struct{}{}
	}
	for method := range byMethod {
		sortCompiledRoutes(byMethod[method])
	}
	return &RoutingSnapshot{routesByMethod: byMethod, paths: paths}, nil
}

func routeShape(segments []string) string {
	shape := make([]string, len(segments))
	for index, segment := range segments {
		if segment == "*" {
			shape[index] = "*"
		} else if strings.HasPrefix(segment, "{") {
			shape[index] = "{}"
		} else {
			shape[index] = segment
		}
	}
	return strings.Join(shape, "/")
}

func sortCompiledRoutes(routes []compiledRoute) {
	for index := 1; index < len(routes); index++ {
		current := routes[index]
		position := index
		for position > 0 && moreSpecific(current, routes[position-1]) {
			routes[position] = routes[position-1]
			position--
		}
		routes[position] = current
	}
}

func moreSpecific(left, right compiledRoute) bool {
	if left.kind != right.kind {
		return left.kind < right.kind
	}
	return len(left.segments) > len(right.segments)
}

// Match finds a route without accessing external state.
func (snapshot *RoutingSnapshot) Match(method, path string) (RouteMatch, bool, []string) {
	method = strings.ToUpper(method)
	path = normalizePath(path)
	segments := splitPath(path)
	for _, candidate := range snapshot.routesByMethod[method] {
		parameters, ok := matchSegments(candidate.segments, segments)
		if ok {
			return RouteMatch{RouteID: candidate.route.ID, ServiceID: candidate.route.ServiceID, Route: candidate.route, Service: candidate.service, PathParameters: parameters, MatchedPath: path}, true, nil
		}
	}
	allowed := make([]string, 0)
	for candidateMethod, candidates := range snapshot.routesByMethod {
		for _, candidate := range candidates {
			if _, ok := matchSegments(candidate.segments, segments); ok {
				allowed = append(allowed, candidateMethod)
				break
			}
		}
	}
	return RouteMatch{}, false, allowed
}

func splitPath(path string) []string {
	if path == "" || path == "/" {
		return nil
	}
	return strings.Split(strings.TrimPrefix(path, "/"), "/")
}

func matchSegments(pattern, path []string) (map[string]string, bool) {
	parameters := make(map[string]string)
	for index, segment := range pattern {
		if segment == "*" {
			return parameters, index < len(path)
		}
		if index >= len(path) {
			return nil, false
		}
		if strings.HasPrefix(segment, "{") {
			parameters[segment[1:len(segment)-1]] = path[index]
		} else if segment != path[index] {
			return nil, false
		}
	}
	return parameters, len(pattern) == len(path)
}
