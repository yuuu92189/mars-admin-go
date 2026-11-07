package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"mars-admin-go/internal/modules/organization/application"
	"mars-admin-go/internal/modules/organization/domain"
	"mars-admin-go/internal/platform/httpserver"

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

func (h *Handler) LegacyDepartmentRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth)
	router.With(httpserver.RequirePermission("department:read")).Get("/tree", h.legacyDepartments)
	router.With(httpserver.RequirePermission("department:read")).Get("/list", h.legacyDepartments)
	router.With(httpserver.RequirePermission("department:manage")).Post("/", h.legacyCreateDepartment)
	router.With(httpserver.RequirePermission("department:manage")).Put("/", h.legacyUpdateDepartment)
	router.With(httpserver.RequirePermission("department:manage")).Delete("/{id}", h.deleteDepartment)
	return router
}

func (h *Handler) LegacyPositionRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth)
	router.With(httpserver.RequirePermission("department:read")).Get("/page", h.legacyPositions)
	router.With(httpserver.RequirePermission("department:read")).Get("/list", h.legacyPositions)
	router.With(httpserver.RequirePermission("department:read")).Get("/tree", h.legacyPositions)
	router.With(httpserver.RequirePermission("department:read")).Get("/{id}", h.legacyPosition)
	router.With(httpserver.RequirePermission("department:manage")).Post("/", h.legacyCreatePosition)
	router.With(httpserver.RequirePermission("department:manage")).Put("/", h.legacyUpdatePosition)
	router.With(httpserver.RequirePermission("department:manage")).Delete("/{id}", h.deletePosition)
	return router
}

type legacyDepartmentInput struct {
	ID       any    `json:"id"`
	ParentID any    `json:"parentId"`
	DeptName string `json:"deptName"`
	Name     string `json:"name"`
	Code     string `json:"code"`
	Leader   string `json:"leader"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Sort     int32  `json:"sort"`
	Status   any    `json:"status"`
}

type legacyPositionInput struct {
	ID       any    `json:"id"`
	ParentID any    `json:"parentId"`
	PostCode string `json:"postCode"`
	PostName string `json:"postName"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Sort     int32  `json:"sort"`
	Status   any    `json:"status"`
	Remark   string `json:"remark"`
}

func (h *Handler) legacyDepartments(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.Departments(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyDepartmentTree(items))
}

func (h *Handler) legacyCreateDepartment(w http.ResponseWriter, r *http.Request) {
	var input legacyDepartmentInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	parentID, err := optionalUUID(input.ParentID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	name := firstString(input.DeptName, input.Name)
	code := firstString(input.Code, "dept_"+uuid.NewString())
	item, err := h.service.CreateDepartment(r.Context(), domain.DepartmentInput{
		ParentID: parentID, Name: name, Code: code, LeaderName: input.Leader,
		Phone: input.Phone, Email: input.Email, Sort: input.Sort,
		Status: legacyOrganizationStatus(input.Status),
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, legacyDepartment(item))
}

func (h *Handler) legacyUpdateDepartment(w http.ResponseWriter, r *http.Request) {
	var input legacyDepartmentInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	id, err := requiredUUID(input.ID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	parentID, err := optionalUUID(input.ParentID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.UpdateDepartment(r.Context(), id, domain.DepartmentInput{
		ParentID: parentID, Name: firstString(input.DeptName, input.Name),
		Code: firstString(input.Code, "dept_"+id.String()), LeaderName: input.Leader,
		Phone: input.Phone, Email: input.Email, Sort: input.Sort,
		Status: legacyOrganizationStatus(input.Status),
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyDepartment(item))
}

func (h *Handler) legacyPositions(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.Positions(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	tree := legacyPositionTree(items)
	if r.URL.Path == "/api/sys/post/page" {
		page, pageSize, err := httpserver.ParsePagination(r, 100)
		if err != nil {
			httpserver.WriteError(w, r, h.logger, err)
			return
		}
		httpserver.WriteJSON(w, http.StatusOK, map[string]any{
			"list": tree, "records": tree, "items": tree,
			"total": len(tree), "page": page, "pageSize": pageSize,
		})
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, tree)
}

func (h *Handler) legacyPosition(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	items, err := h.service.Positions(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	for _, item := range items {
		if item.ID == id {
			httpserver.WriteJSON(w, http.StatusOK, legacyPosition(item))
			return
		}
	}
	httpserver.WriteJSON(w, http.StatusNotFound, map[string]string{"detail": "Position was not found."})
}

func (h *Handler) legacyCreatePosition(w http.ResponseWriter, r *http.Request) {
	var input legacyPositionInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	parentID, err := optionalUUID(input.ParentID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.CreatePosition(r.Context(), domain.PositionInput{
		ParentID: parentID, Code: firstString(input.PostCode, input.Code, "post_"+uuid.NewString()),
		Name: firstString(input.PostName, input.Name), Sort: input.Sort,
		Status: legacyOrganizationStatus(input.Status), Description: input.Remark,
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, legacyPosition(item))
}

func (h *Handler) legacyUpdatePosition(w http.ResponseWriter, r *http.Request) {
	var input legacyPositionInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	id, err := requiredUUID(input.ID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	parentID, err := optionalUUID(input.ParentID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.UpdatePosition(r.Context(), id, domain.PositionInput{
		ParentID: parentID, Code: firstString(input.PostCode, input.Code, "post_"+id.String()),
		Name: firstString(input.PostName, input.Name), Sort: input.Sort,
		Status: legacyOrganizationStatus(input.Status), Description: input.Remark,
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyPosition(item))
}

func legacyDepartment(item domain.Department) map[string]any {
	result := map[string]any{
		"id": item.ID, "deptName": item.Name, "name": item.Name, "code": item.Code,
		"ancestors": item.Path, "sort": item.Sort, "leader": item.LeaderName,
		"phone": item.Phone, "email": item.Email, "status": organizationStatusCode(item.Status),
		"createTime": item.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if item.ParentID != nil {
		result["parentId"] = *item.ParentID
	} else {
		result["parentId"] = 0
	}
	if len(item.Children) > 0 {
		children := make([]map[string]any, 0, len(item.Children))
		for _, child := range item.Children {
			children = append(children, legacyDepartment(child))
		}
		result["children"] = children
	}
	return result
}

func legacyDepartmentTree(items []domain.Department) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, legacyDepartment(item))
	}
	return result
}

func legacyPosition(item domain.Position) map[string]any {
	result := map[string]any{
		"id": item.ID, "postCode": item.Code, "postName": item.Name, "code": item.Code,
		"name": item.Name, "sort": item.Sort, "status": organizationStatusCode(item.Status),
		"remark": item.Description, "createTime": item.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if item.ParentID != nil {
		result["parentId"] = *item.ParentID
	} else {
		result["parentId"] = 0
	}
	return result
}

func legacyPositionTree(items []domain.Position) []map[string]any {
	children := make(map[uuid.UUID][]map[string]any)
	roots := make([]map[string]any, 0)
	byID := make(map[uuid.UUID]map[string]any, len(items))
	for _, item := range items {
		byID[item.ID] = legacyPosition(item)
	}
	for _, item := range items {
		node := byID[item.ID]
		if item.ParentID == nil {
			roots = append(roots, node)
			continue
		}
		children[*item.ParentID] = append(children[*item.ParentID], node)
	}
	for id, childNodes := range children {
		if parent, ok := byID[id]; ok {
			parent["children"] = childNodes
		}
	}
	return roots
}

func optionalUUID(value any) (*uuid.UUID, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case float64:
		if typed == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("parentId must be a UUID or zero")
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
	case uuid.UUID:
		if typed == uuid.Nil {
			return nil, nil
		}
		return &typed, nil
	default:
		return nil, fmt.Errorf("invalid parentId type")
	}
}

func requiredUUID(value any) (uuid.UUID, error) {
	parsed, err := optionalUUID(value)
	if err != nil {
		return uuid.Nil, err
	}
	if parsed == nil {
		return uuid.Nil, fmt.Errorf("id is required")
	}
	return *parsed, nil
}

func firstString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func legacyOrganizationStatus(value any) string {
	switch typed := value.(type) {
	case float64:
		if typed == 0 {
			return "disabled"
		}
		return "active"
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "0", "disabled", "inactive":
			return "disabled"
		default:
			return "active"
		}
	default:
		return "active"
	}
}

func organizationStatusCode(value string) int {
	if value == "active" {
		return 1
	}
	return 0
}

func (h *Handler) departments(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.Departments(r.Context())
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
	item, err := h.service.Department(r.Context(), id)
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
	item, err := h.service.CreateDepartment(r.Context(), input)
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
	item, err := h.service.UpdateDepartment(r.Context(), id, input)
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
	if err := h.service.DeleteDepartment(r.Context(), id); err != nil {
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
