package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

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

func (h *Handler) LegacyRolesRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth)
	router.With(httpserver.RequirePermission("role:read")).Get("/page", h.legacyRoles)
	router.With(httpserver.RequirePermission("role:read")).Get("/list", h.legacyRoles)
	router.With(httpserver.RequirePermission("role:read")).Get("/{id}", h.legacyRole)
	router.With(httpserver.RequirePermission("role:create")).Post("/", h.legacyCreateRole)
	router.With(httpserver.RequirePermission("role:create")).Put("/", h.legacyUpdateRole)
	router.With(httpserver.RequirePermission("role:create")).Delete("/{id}", h.deleteRole)
	return router
}

func (h *Handler) LegacyMenuRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth)
	router.With(httpserver.RequirePermission("role:read")).Get("/tree", h.legacyMenus)
	router.With(httpserver.RequirePermission("role:read")).Get("/list", h.legacyMenus)
	router.With(httpserver.RequirePermission("role:read")).Get("/{id}", h.legacyMenu)
	router.With(httpserver.RequirePermission("role:assign")).Post("/", h.legacyCreateMenu)
	router.With(httpserver.RequirePermission("role:assign")).Put("/", h.legacyUpdateMenu)
	router.With(httpserver.RequirePermission("role:assign")).Delete("/{id}", h.deleteMenu)
	return router
}

type legacyRolePayload struct {
	ID          any    `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Remark      string `json:"remark"`
	Sort        int32  `json:"sort"`
	Status      any    `json:"status"`
	DataScope   any    `json:"dataScope"`
}

type legacyRoleMutation struct {
	Role    legacyRolePayload `json:"role"`
	MenuIDs []uuid.UUID       `json:"menuIds"`
	DeptIDs []uuid.UUID       `json:"deptIds"`
}

type legacyMenuPayload struct {
	ID         any    `json:"id"`
	ParentID   any    `json:"parentId"`
	Name       string `json:"name"`
	Type       int16  `json:"type"`
	Path       string `json:"path"`
	Component  string `json:"component"`
	Permission string `json:"permission"`
	Icon       string `json:"icon"`
	Sort       int32  `json:"sort"`
	Visible    any    `json:"visible"`
	IsFrame    any    `json:"isFrame"`
}

func (h *Handler) legacyRoles(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	query := firstAccessString(r.URL.Query().Get("name"), r.URL.Query().Get("keyword"))
	roles, total, err := h.service.Roles(r.Context(), query, page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	items := make([]map[string]any, 0, len(roles))
	for _, role := range roles {
		items = append(items, legacyRole(role))
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"list": items, "records": items, "items": items,
		"total": total, "page": page, "pageSize": pageSize,
	})
}

func (h *Handler) legacyRole(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	role, err := h.service.Role(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	menuIDs, err := h.service.RoleMenuIDs(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"role": legacyRole(role), "menuIds": menuIDs, "deptIds": []uuid.UUID{},
	})
}

func (h *Handler) legacyCreateRole(w http.ResponseWriter, r *http.Request) {
	var input legacyRoleMutation
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	role, err := h.service.CreateRole(r.Context(), actor, domain.CreateRoleInput{
		Name: input.Role.Name, Code: input.Role.Code,
		Description: firstAccessString(input.Role.Description, input.Role.Remark),
		DataScope:   legacyDataScope(input.Role.DataScope),
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	if len(input.MenuIDs) > 0 {
		if err := h.service.SetRoleMenus(r.Context(), actor, role.ID, input.MenuIDs); err != nil {
			httpserver.WriteError(w, r, h.logger, err)
			return
		}
	}
	httpserver.WriteJSON(w, http.StatusCreated, legacyRole(role))
}

func (h *Handler) legacyUpdateRole(w http.ResponseWriter, r *http.Request) {
	var input legacyRoleMutation
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	id, err := accessRequiredUUID(input.Role.ID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	status := "active"
	if legacyAccessStatus(input.Role.Status) == 0 {
		status = "disabled"
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	role, err := h.service.UpdateRole(r.Context(), actor, id, domain.UpdateRoleInput{
		Name: input.Role.Name, Description: firstAccessString(input.Role.Description, input.Role.Remark),
		DataScope: legacyDataScope(input.Role.DataScope), Status: status,
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	if input.MenuIDs != nil {
		if err := h.service.SetRoleMenus(r.Context(), actor, role.ID, input.MenuIDs); err != nil {
			httpserver.WriteError(w, r, h.logger, err)
			return
		}
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyRole(role))
}

func (h *Handler) legacyMenus(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	items, err := h.service.Menus(r.Context(), actor.UserID, actor.IsSuperAdmin)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyAccessMenus(items))
}

func (h *Handler) legacyMenu(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	items, err := h.service.Menus(r.Context(), actor.UserID, actor.IsSuperAdmin)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	if item, ok := findLegacyMenu(items, id); ok {
		httpserver.WriteJSON(w, http.StatusOK, item)
		return
	}
	httpserver.WriteJSON(w, http.StatusNotFound, map[string]string{"detail": "Menu was not found."})
}

func (h *Handler) legacyCreateMenu(w http.ResponseWriter, r *http.Request) {
	var input legacyMenuPayload
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	parentID, err := accessOptionalUUID(input.ParentID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.CreateMenu(r.Context(), domain.MenuInput{
		ParentID: parentID, Name: input.Name, Type: input.Type, Path: input.Path,
		Component: input.Component, Permission: input.Permission, Icon: input.Icon,
		Sort: input.Sort, Visible: legacyBool(input.Visible), IsFrame: legacyBool(input.IsFrame),
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, legacyAccessMenu(item))
}

func (h *Handler) legacyUpdateMenu(w http.ResponseWriter, r *http.Request) {
	var input legacyMenuPayload
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	id, err := accessRequiredUUID(input.ID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	parentID, err := accessOptionalUUID(input.ParentID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.UpdateMenu(r.Context(), id, domain.MenuInput{
		ParentID: parentID, Name: input.Name, Type: input.Type, Path: input.Path,
		Component: input.Component, Permission: input.Permission, Icon: input.Icon,
		Sort: input.Sort, Visible: legacyBool(input.Visible), IsFrame: legacyBool(input.IsFrame),
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyAccessMenu(item))
}

func legacyRole(role domain.Role) map[string]any {
	return map[string]any{
		"id": role.ID, "name": role.Name, "code": role.Code,
		"description": role.Description, "remark": role.Description,
		"sort": 0, "status": legacyAccessStatus(role.Status),
		"dataScope": legacyDataScopeCode(role.DataScope), "createTime": role.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func legacyAccessMenus(items []domain.Menu) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, legacyAccessMenu(item))
	}
	return result
}

func legacyAccessMenu(item domain.Menu) map[string]any {
	result := map[string]any{
		"id": item.ID, "name": item.Name, "type": item.Type, "path": item.Path,
		"component": item.Component, "permission": item.Permission, "icon": item.Icon,
		"sort": item.Sort, "visible": legacyAccessStatus(roleActive(item.Visible)),
		"status": 1, "isFrame": boolToInt(item.IsFrame),
	}
	if item.ParentID != nil {
		result["parentId"] = *item.ParentID
	} else {
		result["parentId"] = 0
	}
	if len(item.Children) > 0 {
		result["children"] = legacyAccessMenus(item.Children)
	}
	return result
}

func roleActive(visible bool) string {
	if visible {
		return "active"
	}
	return "disabled"
}

func findLegacyMenu(items []domain.Menu, id uuid.UUID) (map[string]any, bool) {
	for _, item := range items {
		if item.ID == id {
			return legacyAccessMenu(item), true
		}
		if child, ok := findLegacyMenu(item.Children, id); ok {
			return child, true
		}
	}
	return nil, false
}

func legacyDataScope(value any) string {
	switch typed := value.(type) {
	case float64:
		switch typed {
		case 1:
			return "all"
		case 2:
			return "custom"
		case 3:
			return "department"
		case 4:
			return "department_tree"
		case 5:
			return "self"
		}
	case string:
		switch typed {
		case "1":
			return "all"
		case "2":
			return "custom"
		case "3":
			return "department"
		case "4":
			return "department_tree"
		case "5":
			return "self"
		default:
			return typed
		}
	}
	return "self"
}

func legacyDataScopeCode(value string) int {
	switch value {
	case "all":
		return 1
	case "custom":
		return 2
	case "department":
		return 3
	case "department_tree":
		return 4
	default:
		return 5
	}
}

func legacyAccessStatus(value any) int {
	switch typed := value.(type) {
	case float64:
		if typed == 1 {
			return 1
		}
	case string:
		if typed == "1" || strings.EqualFold(typed, "active") {
			return 1
		}
	case bool:
		if typed {
			return 1
		}
	}
	return 0
}

func legacyBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case float64:
		return typed == 1
	case string:
		return typed == "1" || strings.EqualFold(typed, "true")
	default:
		return false
	}
}

func accessOptionalUUID(value any) (*uuid.UUID, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case string:
		value := strings.TrimSpace(typed)
		if value == "" || value == "0" {
			return nil, nil
		}
		parsed, err := uuid.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("invalid parentId: %w", err)
		}
		return &parsed, nil
	case float64:
		if typed == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("parentId must be a UUID or zero")
	case uuid.UUID:
		if typed == uuid.Nil {
			return nil, nil
		}
		return &typed, nil
	default:
		return nil, fmt.Errorf("invalid parentId")
	}
}

func accessRequiredUUID(value any) (uuid.UUID, error) {
	parsed, err := accessOptionalUUID(value)
	if err != nil {
		return uuid.Nil, err
	}
	if parsed == nil {
		return uuid.Nil, fmt.Errorf("id is required")
	}
	return *parsed, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func firstAccessString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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
