package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRoutePatternIncludesMountedPrefix(t *testing.T) {
	router := chi.NewRouter()
	subrouter := chi.NewRouter()
	subrouter.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		got := routePattern(r)
		if got != "/api/v1/users/{id}" {
			t.Fatalf("unexpected route pattern %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	router.Mount("/api/v1", subrouter)

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/users/123", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("unexpected status %d", recorder.Code)
	}
}
