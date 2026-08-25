package routing

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func testService() Service { return Service{ID: "service-1", Name: "users", Enabled: true} }
func testRoute(id, method, path string) Route {
	return Route{ID: id, Name: id, Method: method, Path: path, ServiceID: "service-1", Enabled: true}
}

func TestMatchingAndPrecedence(t *testing.T) {
	snapshot, err := BuildSnapshot([]Service{testService()}, []Route{testRoute("wild", "GET", "/api/users/*"), testRoute("param", "GET", "/api/users/{id}"), testRoute("exact", "GET", "/api/users/me")})
	if err != nil {
		t.Fatal(err)
	}
	match, ok, _ := snapshot.Match("get", "/api/users/me")
	if !ok || match.RouteID != "exact" {
		t.Fatalf("exact match = %+v, %v", match, ok)
	}
	match, ok, _ = snapshot.Match("GET", "/api/users/123")
	if !ok || match.RouteID != "param" || match.PathParameters["id"] != "123" {
		t.Fatalf("parameter match = %+v, %v", match, ok)
	}
	match, ok, _ = snapshot.Match("GET", "/api/users/123/profile")
	if !ok || match.RouteID != "wild" {
		t.Fatalf("wildcard match = %+v, %v", match, ok)
	}
}

func TestValidationAndConflicts(t *testing.T) {
	for _, route := range []Route{testRoute("bad", "TRACE", "/x"), testRoute("bad", "GET", "x"), testRoute("bad", "GET", "/x/{bad"), testRoute("bad", "GET", "/x/*/tail")} {
		if _, err := BuildSnapshot([]Service{testService()}, []Route{route}); err == nil {
			t.Errorf("accepted invalid route %+v", route)
		}
	}
	if _, err := BuildSnapshot([]Service{testService()}, []Route{testRoute("one", "GET", "/x"), testRoute("two", "GET", "/x/")}); err == nil {
		t.Error("accepted duplicate normalized paths")
	}
}

func TestHandler404And405(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Reload([]Service{testService()}, []Route{testRoute("route", "GET", "/items")}); err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(registry, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	for _, test := range []struct {
		method, path string
		status       int
		allow        string
	}{{"GET", "/items", 200, ""}, {"POST", "/items", 405, "GET"}, {"GET", "/missing", 404, ""}} {
		record := httptest.NewRecorder()
		handler.ServeHTTP(record, httptest.NewRequest(test.method, test.path, nil))
		if record.Code != test.status || record.Header().Get("Allow") != test.allow {
			t.Errorf("%s %s => %d allow=%q", test.method, test.path, record.Code, record.Header().Get("Allow"))
		}
	}
}

func TestHandlerIgnoresQueryString(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Reload([]Service{testService()}, []Route{testRoute("route", "GET", "/items")}); err != nil {
		t.Fatal(err)
	}
	record := httptest.NewRecorder()
	NewHandler(registry, http.NotFoundHandler()).ServeHTTP(record, httptest.NewRequest("GET", "/items?filter=active", nil))
	if record.Code != http.StatusOK {
		t.Fatalf("query string changed match status: %d", record.Code)
	}
}

func TestRegistryConcurrentReload(t *testing.T) {
	registry := NewRegistry()
	services := []Service{testService()}
	routes := []Route{testRoute("route", "GET", "/items")}
	var group sync.WaitGroup
	for index := 0; index < 10; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for count := 0; count < 100; count++ {
				_, _, _ = registry.Get().Match("GET", "/items")
			}
		}()
	}
	for index := 0; index < 20; index++ {
		if err := registry.Reload(services, routes); err != nil {
			t.Fatal(err)
		}
	}
	group.Wait()
}

func BenchmarkMatchExact(b *testing.B)     { benchmarkMatch(b, "/api/users", "GET") }
func BenchmarkMatchParameter(b *testing.B) { benchmarkMatch(b, "/api/users/{id}", "GET") }
func BenchmarkMatchWildcard(b *testing.B)  { benchmarkMatch(b, "/api/users/*", "GET") }
func BenchmarkMatchLargeTable(b *testing.B) {
	routes := make([]Route, 100)
	for index := range routes {
		routes[index] = testRoute(strings.Repeat("r", index+1), "GET", "/api/items/"+strings.Repeat("x", index+1))
	}
	snapshot, _ := BuildSnapshot([]Service{testService()}, routes)
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		snapshot.Match("GET", "/api/items/xxxxxxxxxxxxxxxx")
	}
}
func benchmarkMatch(b *testing.B, path, method string) {
	snapshot, _ := BuildSnapshot([]Service{testService()}, []Route{testRoute("route", method, path)})
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		snapshot.Match(method, "/api/users/123")
	}
}
