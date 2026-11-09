package monitoring

import (
	"log/slog"
	"net/http"

	"mars-admin-go/internal/platform/httpserver"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
	auth    func(http.Handler) http.Handler
	logger  *slog.Logger
	metrics *httpserver.HTTPMetrics
}

func NewHandler(service *Service, auth func(http.Handler) http.Handler, logger *slog.Logger, metrics ...*httpserver.HTTPMetrics) *Handler {
	handler := &Handler{service: service, auth: auth, logger: logger}
	if len(metrics) > 0 {
		handler.metrics = metrics[0]
	}
	return handler
}

func (h *Handler) Routes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth, httpserver.RequirePermission("audit:read"))
	router.Get("/snapshot", h.snapshot)
	router.Get("/dashboard", h.dashboard)
	return router
}

func (h *Handler) snapshot(w http.ResponseWriter, r *http.Request) {
	httpserver.WriteJSON(w, http.StatusOK, h.service.Snapshot(r.Context()))
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.Dashboard(r.Context(), r.URL.Query().Get("range"))
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, data)
}
