package http

import (
	"log/slog"
	"net/http"

	"mars-admin-go/internal/modules/access/application"
	"mars-admin-go/internal/modules/access/domain"
	"mars-admin-go/internal/platform/httpserver"
	"mars-admin-go/internal/platform/requestctx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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
	router.With(httpserver.RequirePermission("role:read")).Get("/roles", h.roles)
	router.With(httpserver.RequirePermission("role:create")).Post("/roles", h.createRole)
	router.With(httpserver.RequirePermission("role:create")).Put("/roles/{id}", h.updateRole)
	router.With(httpserver.RequirePermission("role:create")).Delete("/roles/{id}", h.deleteRole)
	router.With(httpserver.RequirePermission("role:read")).Get("/roles/{id}/menus", h.roleMenus)
	router.With(httpserver.RequirePermission("role:assign")).Put("/roles/{id}/menus", h.setRoleMenus)
	router.With(httpserver.RequirePermission("role:assign")).Put("/roles/{id}/permissions", h.setRolePermissions)
	router.With(httpserver.RequirePermission("role:read")).Get("/permissions", h.permissions)
	router.With(httpserver.RequirePermission("role:read")).Get("/menus", h.menus)
	router.With(httpserver.RequirePermission("role:assign")).Post("/menus", h.createMenu)
	router.With(httpserver.RequirePermission("role:assign")).Put("/menus/{id}", h.updateMenu)
	router.With(httpserver.RequirePermission("role:assign")).Delete("/menus/{id}", h.deleteMenu)
	return router
}

func (h *Handler) roles(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	roles, total, err := h.service.Roles(r.Context(), r.URL.Query().Get("keyword"), page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, httpserver.NewPage(roles, total, page, pageSize))
}

func (h *Handler) createRole(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateRoleInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	role, err := h.service.CreateRole(r.Context(), actor, input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, role)
}

func (h *Handler) updateRole(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	var input domain.UpdateRoleInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	role, err := h.service.UpdateRole(r.Context(), actor, id, input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, role)
}

func (h *Handler) deleteRole(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.service.DeleteRole(r.Context(), actor, id); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) permissions(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.Permissions(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) menus(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	items, err := h.service.Menus(r.Context(), actor.UserID, actor.IsSuperAdmin)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) roleMenus(w http.ResponseWriter, r *http.Request) {
	roleID, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	ids, err := h.service.RoleMenuIDs(r.Context(), roleID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, ids)
}

func (h *Handler) setRoleMenus(w http.ResponseWriter, r *http.Request) {
	roleID, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	var input struct {
		MenuIDs []uuid.UUID `json:"menuIds"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.service.SetRoleMenus(r.Context(), actor, roleID, input.MenuIDs); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) setRolePermissions(w http.ResponseWriter, r *http.Request) {
	roleID, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	var input struct {
		Codes []string `json:"permissions"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.service.SetRolePermissions(r.Context(), actor, roleID, input.Codes); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createMenu(w http.ResponseWriter, r *http.Request) {
	var input domain.MenuInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	menu, err := h.service.CreateMenu(r.Context(), input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, menu)
}

func (h *Handler) updateMenu(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	var input domain.MenuInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	menu, err := h.service.UpdateMenu(r.Context(), id, input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, menu)
}

func (h *Handler) deleteMenu(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	if err := h.service.DeleteMenu(r.Context(), id); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
