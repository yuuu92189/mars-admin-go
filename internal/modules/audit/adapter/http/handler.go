package http

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"mars-admin-go/internal/modules/audit/application"
	"mars-admin-go/internal/modules/audit/domain"
	"mars-admin-go/internal/platform/httpserver"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *application.Service
	auth    func(http.Handler) http.Handler
	logger  *slog.Logger
}

func New(service *application.Service, auth func(http.Handler) http.Handler, logger *slog.Logger) *Handler {
	return &Handler{service: service, auth: auth, logger: logger}
}

func (h *Handler) Routes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth, httpserver.RequirePermission("audit:read"))
	router.Get("/events", h.events)
	router.Get("/logins", h.logins)
	return router
}

func (h *Handler) LegacyRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth, httpserver.RequirePermission("audit:read"))
	router.Get("/operlog/page", h.legacyEvents)
	router.Get("/loginlog/page", h.legacyLogins)
	return router
}

func (h *Handler) legacyEvents(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	events, total, err := h.service.List(r.Context(), domain.Filter{
		Action: r.URL.Query().Get("action"), ResourceType: r.URL.Query().Get("resourceType"),
		From: parseTime(r.URL.Query().Get("from")), To: parseTime(r.URL.Query().Get("to")),
		Page: page, PageSize: pageSize,
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	items := make([]map[string]any, 0, len(events))
	for _, event := range events {
		status := 1
		if event.Outcome == "success" {
			status = 0
		}
		path, _ := event.Metadata["path"].(string)
		duration, _ := event.Metadata["duration_ms"].(float64)
		method := strings.ToUpper(event.Action)
		items = append(items, map[string]any{
			"id": event.ID, "title": event.ResourceType, "businessType": 1,
			"method": event.ResourceType, "requestMethod": method, "operName": event.ActorName,
			"operUrl": path, "operIp": event.IPAddress, "operParam": event.Metadata,
			"jsonResult": "", "status": status, "errorMsg": "", "operTime": event.OccurredAt.Format("2006-01-02 15:04:05"),
			"costTime": int64(duration),
		})
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"list": items, "records": items, "items": items,
		"total": total, "page": page, "pageSize": pageSize,
	})
}

func (h *Handler) legacyLogins(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	events, total, err := h.service.LoginEvents(r.Context(), page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	items := make([]map[string]any, 0, len(events))
	for _, event := range events {
		status := 1
		if event.Outcome == "success" {
			status = 0
		}
		browser, os := browserAndOS(event.UserAgent)
		items = append(items, map[string]any{
			"id": event.ID, "username": event.Username, "ipaddr": event.IPAddress,
			"loginLocation": "", "browser": browser, "os": os,
			"status": status, "msg": event.FailureReason,
			"loginTime": event.OccurredAt.Format("2006-01-02 15:04:05"),
		})
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"list": items, "records": items, "items": items,
		"total": total, "page": page, "pageSize": pageSize,
	})
}

func browserAndOS(userAgent string) (string, string) {
	browser := "Unknown"
	osName := "Unknown"
	switch {
	case strings.Contains(userAgent, "Edg/"):
		browser = "Edge"
	case strings.Contains(userAgent, "Chrome/"):
		browser = "Chrome"
	case strings.Contains(userAgent, "Firefox/"):
		browser = "Firefox"
	case strings.Contains(userAgent, "Safari/"):
		browser = "Safari"
	}
	switch {
	case strings.Contains(userAgent, "Windows"):
		osName = "Windows"
	case strings.Contains(userAgent, "Android"):
		osName = "Android"
	case strings.Contains(userAgent, "iPhone"), strings.Contains(userAgent, "iPad"):
		osName = "iOS"
	case strings.Contains(userAgent, "Mac OS"):
		osName = "macOS"
	case strings.Contains(userAgent, "Linux"):
		osName = "Linux"
	}
	return browser, osName
}

func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	from := parseTime(r.URL.Query().Get("from"))
	to := parseTime(r.URL.Query().Get("to"))
	events, total, err := h.service.List(r.Context(), domain.Filter{
		Action: r.URL.Query().Get("action"), ResourceType: r.URL.Query().Get("resourceType"),
		From: from, To: to, Page: page, PageSize: pageSize,
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, httpserver.NewPage(events, total, page, pageSize))
}

func (h *Handler) logins(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	events, total, err := h.service.LoginEvents(r.Context(), page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, httpserver.NewPage(events, total, page, pageSize))
}

func parseTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	parsed = parsed.UTC()
	return &parsed
}
