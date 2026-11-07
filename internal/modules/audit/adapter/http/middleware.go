package http

import (
	"net/http"
	"strings"

	"mars-admin-go/internal/modules/audit/application"
)

func AuditWrites(service *application.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/api/") || r.Method == http.MethodGet || r.Method == http.MethodHead ||
				r.Method == http.MethodOptions || strings.HasPrefix(r.URL.Path, "/api/v1/auth/login") ||
				strings.HasPrefix(r.URL.Path, "/api/v1/auth/refresh") || strings.HasPrefix(r.URL.Path, "/api/v1/auth/password") {
				next.ServeHTTP(w, r)
				return
			}
			recorder := &auditResponseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			outcome := "success"
			if recorder.status == http.StatusForbidden || recorder.status == http.StatusUnauthorized {
				outcome = "denied"
			} else if recorder.status >= 400 {
				outcome = "failure"
			}
			segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			resourceType := "api"
			if len(segments) > 2 {
				resourceType = segments[2]
			}
			service.WriteHTTPOutcome(r.Context(), r, strings.ToLower(r.Method), resourceType, "",
				outcome, map[string]any{"path": r.URL.Path, "status": recorder.status})
		})
	}
}

type auditResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *auditResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *auditResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
