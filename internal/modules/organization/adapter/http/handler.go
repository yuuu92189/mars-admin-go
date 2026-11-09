package http

import (
	"log/slog"
	"net/http"

	"mars-admin-go/internal/modules/organization/application"
	"mars-admin-go/internal/modules/organization/domain"
	"mars-admin-go/internal/platform/httpserver"
	"mars-admin-go/internal/platform/requestctx"

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
	router.Use(h.auth)
	router.With(httpserver.RequirePermission("department:read")).Get("/departments", h.departments)
	router.With(httpserver.RequirePermission("department:read")).Get("/departments/{id}", h.department)
	router.With(httpserver.RequirePermission("department:manage")).Post("/departments", h.createDepartment)
	router.With(httpserver.RequirePermission("department:manage")).Put("/departments/{id}", h.updateDepartment)
	router.With(httpserver.RequirePermission("department:manage")).Delete("/departments/{id}", h.deleteDepartment)
	router.With(httpserver.RequirePermission("department:read")).Get("/positions", h.positions)
	router.With(httpserver.RequirePermission("department:manage")).Post("/positions", h.createPosition)
	router.With(httpserver.RequirePermission("department:manage")).Put("/positions/{id}", h.updatePosition)
	router.With(httpserver.RequirePermission("department:manage")).Delete("/positions/{id}", h.deletePosition)
	return router
}

func (h *Handler) departments(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	items, err := h.service.Departments(r.Context(), actor)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) department(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	item, err := h.service.Department(r.Context(), actor, id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) createDepartment(w http.ResponseWriter, r *http.Request) {
	var input domain.DepartmentInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	item, err := h.service.CreateDepartment(r.Context(), actor, input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) updateDepartment(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	var input domain.DepartmentInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	item, err := h.service.UpdateDepartment(r.Context(), actor, id, input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) deleteDepartment(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.service.DeleteDepartment(r.Context(), actor, id); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) positions(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.Positions(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) createPosition(w http.ResponseWriter, r *http.Request) {
	var input domain.PositionInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	item, err := h.service.CreatePosition(r.Context(), input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) updatePosition(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	var input domain.PositionInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	item, err := h.service.UpdatePosition(r.Context(), id, input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) deletePosition(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	if err := h.service.DeletePosition(r.Context(), id); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
