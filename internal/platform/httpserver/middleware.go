package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mars-admin-go/internal/platform/requestctx"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type HTTPMetrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inflight *prometheus.GaugeVec

	recentMu sync.Mutex
	recent   []APICall
	nextID   atomic.Int64
}

type APICall struct {
	ID         int64
	Path       string
	Method     string
	StatusCode int
	DurationMS int64
	IP         string
	UserID     string
	StartedAt  time.Time
}

func NewHTTPMetrics(registry prometheus.Registerer) *HTTPMetrics {
	m := &HTTPMetrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mars_admin",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "HTTP requests grouped by route, method and status.",
		}, []string{"route", "method", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "mars_admin",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"route", "method"}),
		inflight: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "mars_admin",
			Subsystem: "http",
			Name:      "requests_in_flight",
			Help:      "HTTP requests currently being served.",
		}, []string{"method"}),
	}
	registry.MustRegister(m.requests, m.duration, m.inflight)
	return m
}

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" || len(requestID) > 80 {
			requestID = NewRequestID()
		}
		traceID := ""
		if span := trace.SpanContextFromContext(r.Context()); span.IsValid() {
			traceID = span.TraceID().String()
		}
		ctx := ContextWithRequestMetadata(r.Context(), requestID, traceID)
		w.Header().Set("X-Request-ID", requestID)
		if traceID != "" {
			w.Header().Set("X-Trace-ID", traceID)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(r.Context(), "panic serving request", "panic", recovered, "stack", string(debug.Stack()))
					WriteProblem(w, r, NewProblem(http.StatusInternalServerError, "panic", "An internal error occurred."))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; object-src 'none'; base-uri 'self'")
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func CORS(origins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[origin] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if _, ok := allowed[origin]; !ok {
					if r.Method == http.MethodOptions {
						WriteProblem(w, r, NewProblem(http.StatusForbidden, "origin_not_allowed", "The request origin is not allowed."))
						return
					}
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-CSRF-Token, X-Request-ID")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Max-Age", "600")
				}
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func CSRF(headerName, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
				next.ServeHTTP(w, r)
				return
			}
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			sessionCookie, err := r.Cookie("mars_session")
			if err != nil || sessionCookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}
			csrfCookie, err := r.Cookie(cookieName)
			if err != nil || csrfCookie.Value == "" {
				WriteProblem(w, r, NewProblem(http.StatusForbidden, "csrf_missing", "CSRF token is missing."))
				return
			}
			if r.Header.Get(headerName) == "" || r.Header.Get(headerName) != csrfCookie.Value {
				WriteProblem(w, r, NewProblem(http.StatusForbidden, "csrf_invalid", "CSRF token is invalid."))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			requestID, traceID := requestMetadata(r.Context())
			logger.InfoContext(r.Context(), "http request",
				"request_id", requestID,
				"trace_id", traceID,
				"method", r.Method,
				"route", routePattern(r),
				"status", recorder.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

func Metrics(metrics *HTTPMetrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			metrics.inflight.WithLabelValues(r.Method).Inc()
			defer func() {
				metrics.inflight.WithLabelValues(r.Method).Dec()
				route := routePattern(r)
				status := strconv.Itoa(recorder.status)
				metrics.requests.WithLabelValues(route, r.Method, status).Inc()
				duration := time.Since(start)
				metrics.duration.WithLabelValues(route, r.Method).Observe(duration.Seconds())
				metrics.recordAPICall(r, recorder.status, duration, start)
			}()
			next.ServeHTTP(recorder, r)
		})
	}
}

func (m *HTTPMetrics) recordAPICall(r *http.Request, status int, duration time.Duration, started time.Time) {
	if !strings.HasPrefix(r.URL.Path, "/api/") {
		return
	}
	userID := ""
	if actor, ok := requestctx.ActorFrom(r.Context()); ok {
		userID = actor.UserID.String()
	}
	call := APICall{
		ID: m.nextID.Add(1), Path: r.URL.Path, Method: r.Method,
		StatusCode: status, DurationMS: duration.Milliseconds(),
		IP: r.RemoteAddr, UserID: userID, StartedAt: started.UTC(),
	}
	m.recentMu.Lock()
	m.recent = append(m.recent, call)
	if len(m.recent) > 1000 {
		m.recent = append([]APICall(nil), m.recent[len(m.recent)-1000:]...)
	}
	m.recentMu.Unlock()
}

func (m *HTTPMetrics) RecentAPICalls() []APICall {
	m.recentMu.Lock()
	defer m.recentMu.Unlock()
	result := make([]APICall, len(m.recent))
	copy(result, m.recent)
	return result
}

func Tracing(serviceName string) func(http.Handler) http.Handler {
	tracer := otel.Tracer(serviceName)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
			ctx, span := tracer.Start(ctx, r.Method+" "+routePattern(r),
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.request.method", r.Method),
					attribute.String("url.path", r.URL.Path),
					attribute.String("server.address", r.Host),
				),
			)
			defer span.End()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func BodyLimit(bytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > bytes {
				WriteProblem(w, r, NewProblem(http.StatusRequestEntityTooLarge, "body_too_large", fmt.Sprintf("Request body must not exceed %d bytes.", bytes)))
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, bytes)
			next.ServeHTTP(w, r)
		})
	}
}

func Authenticate(resolve func(context.Context, string) (requestctx.Actor, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if existing, ok := requestctx.ActorFrom(r.Context()); ok {
				if _, fromCookie := accessToken(r); fromCookie {
					existing.ViaCookie = true
				}
				next.ServeHTTP(w, r.WithContext(requestctx.WithActor(r.Context(), existing)))
				return
			}
			token, fromCookie := accessToken(r)
			if token == "" {
				WriteProblem(w, r, NewProblem(http.StatusUnauthorized, "authentication_required", "Authentication is required."))
				return
			}
			actor, err := resolve(r.Context(), token)
			if err != nil {
				WriteProblem(w, r, NewProblem(http.StatusUnauthorized, "invalid_session", "The session is invalid or expired."))
				return
			}
			actor.ViaCookie = fromCookie
			next.ServeHTTP(w, r.WithContext(requestctx.WithActor(r.Context(), actor)))
		})
	}
}

func OptionalAuthenticate(resolve func(context.Context, string) (requestctx.Actor, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, fromCookie := accessToken(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}
			actor, err := resolve(r.Context(), token)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			actor.ViaCookie = fromCookie
			next.ServeHTTP(w, r.WithContext(requestctx.WithActor(r.Context(), actor)))
		})
	}
}

func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := requestctx.ActorFrom(r.Context())
			if !ok {
				WriteProblem(w, r, NewProblem(http.StatusUnauthorized, "authentication_required", "Authentication is required."))
				return
			}
			if !actor.HasPermission(permission) {
				WriteProblem(w, r, NewProblem(http.StatusForbidden, "permission_denied", "You do not have permission to perform this action."))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAnyPermission(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := requestctx.ActorFrom(r.Context())
			if !ok {
				WriteProblem(w, r, NewProblem(http.StatusUnauthorized, "authentication_required", "Authentication is required."))
				return
			}
			for _, permission := range permissions {
				if actor.HasPermission(permission) {
					next.ServeHTTP(w, r)
					return
				}
			}
			WriteProblem(w, r, NewProblem(http.StatusForbidden, "permission_denied", "You do not have permission to perform this action."))
		})
	}
}

func accessToken(r *http.Request) (string, bool) {
	if cookie, err := r.Cookie("mars_session"); err == nil && cookie.Value != "" {
		return cookie.Value, true
	}
	raw := strings.TrimSpace(r.Header.Get("Authorization"))
	if raw == "" {
		return "", false
	}
	if parts := strings.Fields(raw); len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1], false
	}
	return raw, false
}

func routePattern(r *http.Request) string {
	if routeContext := chi.RouteContext(r.Context()); routeContext != nil {
		if len(routeContext.RoutePatterns) > 0 {
			parts := make([]string, 0, len(routeContext.RoutePatterns))
			for _, pattern := range routeContext.RoutePatterns {
				parts = append(parts, strings.TrimSuffix(pattern, "/*"))
			}
			route := strings.Join(parts, "")
			if route != "" {
				return route
			}
		}
	}
	if r.Pattern != "" {
		return r.Pattern
	}
	return "unmatched"
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}
