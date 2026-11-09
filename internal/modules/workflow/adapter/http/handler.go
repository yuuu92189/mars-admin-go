package http

import (
	"log/slog"
	"net/http"

	"mars-admin-go/internal/modules/workflow/application"
	"mars-admin-go/internal/modules/workflow/domain"
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
	router.With(httpserver.RequirePermission("workflow:read")).Get("/definitions", h.definitions)
	router.With(httpserver.RequirePermission("workflow:manage")).Post("/definitions", h.createDefinition)
	router.With(httpserver.RequirePermission("workflow:read")).Get("/definitions/{id}", h.definition)
	router.With(httpserver.RequirePermission("workflow:manage")).Post("/definitions/{id}/publish", h.publish)
	router.With(httpserver.RequirePermission("workflow:read")).Get("/instances", h.instances)
	router.With(httpserver.RequirePermission("workflow:read")).Get("/instances/{id}", h.instance)
	router.With(httpserver.RequirePermission("workflow:manage")).Post("/instances", h.start)
	router.With(httpserver.RequirePermission("workflow:manage")).Post("/instances/{id}/terminate", h.terminate)
	router.With(httpserver.RequirePermission("workflow:read")).Get("/instances/{id}/events", h.events)
	router.With(httpserver.RequirePermission("workflow:read")).Get("/tasks/todo", h.todo)
	router.With(httpserver.RequirePermission("workflow:read")).Get("/tasks/done", h.done)
	router.With(httpserver.RequirePermission("workflow:approve")).Post("/tasks/{action}", h.decide)
	return router
}

func (h *Handler) definitions(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	items, total, err := h.service.Definitions(r.Context(), r.URL.Query().Get("status"), page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, httpserver.NewPage(items, total, page, pageSize))
}

func (h *Handler) createDefinition(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateDefinitionInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	item, err := h.service.CreateDefinition(r.Context(), actor.UserID, input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) definition(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.Definition(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	if err := h.service.Publish(r.Context(), id); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) instances(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	items, total, err := h.service.Instances(r.Context(), workflowActor(actor), page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, httpserver.NewPage(items, total, page, pageSize))
}

func (h *Handler) instance(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	item, err := h.service.Instance(r.Context(), workflowActor(actor), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	var input domain.StartInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	instance, task, err := h.service.Start(r.Context(), actor.UserID, input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, map[string]any{"instance": instance, "task": task})
}

func (h *Handler) terminate(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	var input struct {
		Comment string `json:"comment"`
		Message string `json:"message"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	instance, err := h.service.Terminate(r.Context(), workflowActor(actor), id, firstNonEmpty(input.Comment, input.Message))
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, instance)
}

func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	events, err := h.service.Events(r.Context(), workflowActor(actor), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, events)
}

func (h *Handler) todo(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	tasks, total, err := h.service.Todo(r.Context(), workflowActor(actor), page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, httpserver.NewPage(tasks, total, page, pageSize))
}

func (h *Handler) done(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	tasks, total, err := h.service.Done(r.Context(), workflowActor(actor), page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, httpserver.NewPage(tasks, total, page, pageSize))
}

func (h *Handler) decide(w http.ResponseWriter, r *http.Request) {
	action := chi.URLParam(r, "action")
	h.decideAction(w, r, action)
}

func (h *Handler) decideAction(w http.ResponseWriter, r *http.Request, action string) {
	var input struct {
		TaskID    uuid.UUID      `json:"taskId"`
		Comment   string         `json:"comment"`
		Message   string         `json:"message"`
		Variables map[string]any `json:"variables"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	task, instance, err := h.service.Decide(r.Context(), workflowActor(actor), domain.DecisionInput{
		TaskID: input.TaskID, Action: action, Comment: firstNonEmpty(input.Comment, input.Message), Variables: input.Variables,
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{"task": task, "instance": instance})
}

func workflowActor(actor requestctx.Actor) domain.Actor {
	return domain.Actor{UserID: actor.UserID, Roles: actor.Roles, DataScope: actor.DataScope}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
