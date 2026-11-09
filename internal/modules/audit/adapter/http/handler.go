package http

import (
	"log/slog"
	"net/http"
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
