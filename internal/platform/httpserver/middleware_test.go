package httpserver

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func TestRequestIDRejectsUntrustedCharacters(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID, _ := requestMetadata(r.Context())
		if requestID == "" {
			t.Fatal("expected server request id")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", "bad value\r\ninjected")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected server request id header")
	}
	if recorder.Header().Get("X-External-Request-ID") != "" {
		t.Fatal("malicious external request id must not be preserved")
	}
}

func TestRequestIDSeparatesExternalIdentifier(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", "gateway-request_123")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Header().Get("X-Request-ID") == "gateway-request_123" {
		t.Fatal("server request id must not reuse the external id")
	}
	if got := recorder.Header().Get("X-External-Request-ID"); got != "gateway-request_123" {
		t.Fatalf("external request id = %q", got)
	}
}

func TestCORSRejectsDisallowedOriginForEveryMethod(t *testing.T) {
	called := false
	handler := CORS([]string{"https://trusted.example"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/messages", nil)
	request.Header.Set("Origin", "https://evil.example")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden || called {
		t.Fatalf("status=%d called=%v, want rejected origin", recorder.Code, called)
	}
}

func TestCSRFCoversLegacyAndRefreshCookieFlows(t *testing.T) {
	handler := CSRF("X-CSRF-Token", "mars_csrf")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	tests := []struct {
		name     string
		path     string
		cookies  []*http.Cookie
		headers  map[string]string
		wantCode int
	}{
		{
			name: "legacy session cookie requires token", path: "/api/sys/chat/send",
			cookies: []*http.Cookie{{
				Name: "mars_session", Value: "session", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode,
			}},
			wantCode: http.StatusForbidden,
		},
		{
			name: "refresh cookie requires token", path: "/api/v1/auth/refresh",
			cookies: []*http.Cookie{{
				Name: "mars_refresh", Value: "refresh", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode,
			}},
			wantCode: http.StatusForbidden,
		},
		{
			name: "bearer token is not ambient", path: "/api/sys/chat/send",
			headers: map[string]string{"Authorization": "Bearer header-token"}, wantCode: http.StatusNoContent,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, test.path, bytes.NewReader([]byte(`{}`)))
			for _, cookie := range test.cookies {
				request.AddCookie(cookie)
			}
			for key, value := range test.headers {
				request.Header.Set(key, value)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != test.wantCode {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantCode)
			}
		})
	}
}

func TestRecoverDoesNotAppendErrorAfterHeadersCommitted(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := Recover(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		panic("after write")
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want committed status", recorder.Code)
	}
}

func TestTimeoutSkipsWebSocketUpgrade(t *testing.T) {
	handler := Timeout(time.Nanosecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, hasDeadline := r.Context().Deadline(); hasDeadline {
			t.Fatal("websocket upgrade received HTTP timeout")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws/message", nil)
	request.Header.Set("Upgrade", "websocket")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestBodyLimitAllowsMultipartToReachUploadHandler(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), 2<<20)
	handler := BodyLimit(1 << 20)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if len(body) != len(payload) {
			t.Fatalf("body length = %d, want %d", len(body), len(payload))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/upload", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "multipart/form-data; boundary=test")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestSecurityHeadersOnlyTrustForwardedProtoFromProxy(t *testing.T) {
	handler := SecurityHeaders([]string{"10.0.0.0/8"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	untrusted := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	untrusted.RemoteAddr = "203.0.113.10:1234"
	untrusted.Header.Set("X-Forwarded-Proto", "https")
	untrustedRecorder := httptest.NewRecorder()
	handler.ServeHTTP(untrustedRecorder, untrusted)
	if untrustedRecorder.Header().Get("Strict-Transport-Security") != "" {
		t.Fatal("untrusted client must not control HSTS")
	}

	trusted := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	trusted.RemoteAddr = "10.0.0.5:1234"
	trusted.Header.Set("X-Forwarded-Proto", "https")
	trustedRecorder := httptest.NewRecorder()
	handler.ServeHTTP(trustedRecorder, trusted)
	if trustedRecorder.Header().Get("Strict-Transport-Security") == "" {
		t.Fatal("trusted proxy HTTPS request should receive HSTS")
	}
}

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
