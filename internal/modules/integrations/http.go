package integrations

import (
	"io"
	"log/slog"
	"net/http"
	"strings"

	"mars-admin-go/internal/platform/apperror"
	"mars-admin-go/internal/platform/httpserver"
	"mars-admin-go/internal/platform/requestctx"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	registry *Registry
	payments *PaymentService
	auth     func(http.Handler) http.Handler
	logger   *slog.Logger
}

func NewHandler(registry *Registry, payments *PaymentService, auth func(http.Handler) http.Handler, logger *slog.Logger) *Handler {
	return &Handler{registry: registry, payments: payments, auth: auth, logger: logger}
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
	if h.payments == nil {
		httpserver.WriteError(w, r, h.logger, apperror.NotConfigured("payment provider"))
		return
	}
	var request PaymentRequest
	if !httpserver.DecodeJSON(w, r, &request) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	session, err := h.payments.Create(r.Context(), actor.UserID, chi.URLParam(r, "provider"), request)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, session)
}

func (h *Handler) paymentCallback(w http.ResponseWriter, r *http.Request) {
	if h.payments == nil {
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
	result, err := h.payments.HandleCallback(r.Context(), chi.URLParam(r, "provider"), headers, body)
	if err != nil {
		h.logger.WarnContext(r.Context(), "payment callback rejected", "provider", chi.URLParam(r, "provider"), "error", err)
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "accepted", "orderId": result.Order.MerchantOrderID,
		"duplicate": result.Duplicate,
	})
}
