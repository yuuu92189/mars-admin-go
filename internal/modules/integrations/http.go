package integrations

import (
	"io"
	"log/slog"
	"net/http"
	"strings"

	"mars-admin-go/internal/platform/apperror"
	"mars-admin-go/internal/platform/httpserver"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	registry *Registry
	auth     func(http.Handler) http.Handler
	logger   *slog.Logger
}

func NewHandler(registry *Registry, auth func(http.Handler) http.Handler, logger *slog.Logger) *Handler {
	return &Handler{registry: registry, auth: auth, logger: logger}
}

func (h *Handler) Routes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth)
	router.With(httpserver.RequirePermission("settings:read")).Get("/", h.statuses)
	router.With(httpserver.RequirePermission("settings:manage")).Post("/payments/{provider}", h.createPayment)
	return router
}

func (h *Handler) CallbackRoutes() chi.Router {
	router := chi.NewRouter()
	router.Post("/{provider}", h.paymentCallback)
	return router
}

func (h *Handler) statuses(w http.ResponseWriter, r *http.Request) {
	if h.registry == nil {
		httpserver.WriteJSON(w, http.StatusOK, []ProviderStatus{})
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, h.registry.Statuses())
}

func (h *Handler) createPayment(w http.ResponseWriter, r *http.Request) {
	var request PaymentRequest
	if !httpserver.DecodeJSON(w, r, &request) {
		return
	}
	provider, err := h.registry.Payment(chi.URLParam(r, "provider"))
	if err != nil {
		httpserver.WriteError(w, r, h.logger, apperror.NotConfigured("payment provider"))
		return
	}
	session, err := provider.Create(r.Context(), request)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, session)
}

func (h *Handler) paymentCallback(w http.ResponseWriter, r *http.Request) {
	provider, err := h.registry.Payment(chi.URLParam(r, "provider"))
	if err != nil {
		httpserver.WriteError(w, r, h.logger, apperror.NotConfigured("payment provider"))
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	headers := make(map[string]string, len(r.Header))
	for key, values := range r.Header {
		headers[key] = strings.Join(values, ",")
	}
	if err := provider.VerifyCallback(r.Context(), headers, body); err != nil {
		h.logger.WarnContext(r.Context(), "payment callback verification failed", "provider", chi.URLParam(r, "provider"), "error", err)
		httpserver.WriteProblem(w, r, httpserver.NewProblem(http.StatusBadRequest, "invalid_payment_callback", "Payment callback verification failed."))
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}
