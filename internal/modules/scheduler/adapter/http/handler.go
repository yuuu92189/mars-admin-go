package http

import (
	"log/slog"
	"net/http"

	"mars-admin-go/internal/modules/scheduler/application"
	"mars-admin-go/internal/modules/scheduler/domain"
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
	router.With(httpserver.RequirePermission("scheduler:read")).Get("/jobs", h.list)
	router.With(httpserver.RequirePermission("scheduler:read")).Get("/jobs/{id}", h.get)
	router.With(httpserver.RequirePermission("scheduler:manage")).Post("/jobs", h.create)
	router.With(httpserver.RequirePermission("scheduler:manage")).Put("/jobs/{id}", h.update)
	router.With(httpserver.RequirePermission("scheduler:manage")).Delete("/jobs/{id}", h.delete)
	router.With(httpserver.RequirePermission("scheduler:manage")).Post("/jobs/{id}/pause", h.pause)
	router.With(httpserver.RequirePermission("scheduler:manage")).Post("/jobs/{id}/resume", h.resume)
	router.With(httpserver.RequirePermission("scheduler:manage")).Post("/jobs/{id}/run", h.run)
	router.With(httpserver.RequirePermission("scheduler:read")).Get("/jobs/{id}/runs", h.runs)
	return router
}

func (h *Handler) LegacyRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth)
	router.With(httpserver.RequirePermission("scheduler:read")).Get("/page", h.legacyList)
	router.With(httpserver.RequirePermission("scheduler:read")).Get("/{id}", h.legacyGet)
	router.With(httpserver.RequirePermission("scheduler:manage")).Post("/", h.legacyCreate)
	router.With(httpserver.RequirePermission("scheduler:manage")).Put("/", h.legacyUpdate)
	router.With(httpserver.RequirePermission("scheduler:manage")).Delete("/{id}", h.delete)
	router.With(httpserver.RequirePermission("scheduler:manage")).Post("/run/{id}", h.run)
	router.With(httpserver.RequirePermission("scheduler:manage")).Put("/changeStatus", h.legacyChangeStatus)
	router.With(httpserver.RequirePermission("scheduler:read")).Get("/log/page", h.legacyRunsQuery)
	router.With(httpserver.RequirePermission("scheduler:read")).Get("/log/statistics", h.legacyStatistics)
	router.With(httpserver.RequirePermission("scheduler:manage")).Delete("/log/clean", h.legacyCleanLogs)
	return router
}

type legacyJobInput struct {
	ID             any            `json:"id"`
	JobName        string         `json:"jobName"`
	Name           string         `json:"name"`
	JobGroup       string         `json:"jobGroup"`
	InvokeTarget   string         `json:"invokeTarget"`
	Handler        string         `json:"handler"`
	CronExpression string         `json:"cronExpression"`
	Payload        map[string]any `json:"payload"`
	Status         any            `json:"status"`
	Remark         string         `json:"remark"`
	MaxAttempts    int32          `json:"maxAttempts"`
	TimeoutSeconds int32          `json:"timeoutSeconds"`
	ConcurrencyKey string         `json:"concurrencyKey"`
}

func (h *Handler) legacyList(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	items, total, err := h.service.List(r.Context(), legacySchedulerStatus(r.URL.Query().Get("status")), page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, legacyJob(item))
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"list": result, "records": result, "items": result,
		"total": total, "page": page, "pageSize": pageSize,
	})
}

func (h *Handler) legacyGet(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.Get(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyJob(item))
}

func (h *Handler) legacyCreate(w http.ResponseWriter, r *http.Request) {
	var input legacyJobInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	item, err := h.service.Create(r.Context(), actor.UserID, schedulerInput(input))
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, legacyJob(item))
}

func (h *Handler) legacyUpdate(w http.ResponseWriter, r *http.Request) {
	var input legacyJobInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	id, err := schedulerUUID(input.ID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.Update(r.Context(), id, schedulerInput(input))
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyJob(item))
}

func (h *Handler) legacyChangeStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ID     any `json:"id"`
		Status any `json:"status"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	id, err := schedulerUUID(input.ID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	var item domain.Job
	if legacySchedulerStatusCode(input.Status) == 1 {
		item, err = h.service.Resume(r.Context(), id)
	} else {
		item, err = h.service.Pause(r.Context(), id)
	}
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyJob(item))
}

func (h *Handler) legacyRunsQuery(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.URL.Query().Get("jobId"))
	if err != nil {
		page, pageSize, parseErr := httpserver.ParsePagination(r, 100)
		if parseErr != nil {
			httpserver.WriteError(w, r, h.logger, parseErr)
			return
		}
		httpserver.WriteJSON(w, http.StatusOK, map[string]any{
			"list": []map[string]any{}, "records": []map[string]any{},
			"items": []map[string]any{}, "total": 0, "page": page, "pageSize": pageSize,
		})
		return
	}
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	runs, err := h.service.Runs(r.Context(), id, page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	items := make([]map[string]any, 0, len(runs))
	for _, run := range runs {
		items = append(items, legacyRun(run))
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"list": items, "records": items, "items": items,
		"total": len(items), "page": page, "pageSize": pageSize,
	})
}

func (h *Handler) legacyStatistics(w http.ResponseWriter, _ *http.Request) {
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"totalCount": 0, "successCount": 0, "failCount": 0, "dailyStats": []map[string]any{},
	})
}

func (h *Handler) legacyCleanLogs(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func legacyJob(item domain.Job) map[string]any {
	status := 0
	if item.Status == "active" {
		status = 1
	}
	return map[string]any{
		"id": item.ID, "jobName": item.Name, "name": item.Name, "jobGroup": "DEFAULT",
		"invokeTarget": item.Handler, "handler": item.Handler,
		"cronExpression": item.CronExpression, "misfirePolicy": 1, "concurrent": 1,
		"status": status, "remark": "", "payload": item.Payload,
		"createTime": item.CreatedAt.Format("2006-01-02 15:04:05"),
		"updateTime": item.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func legacyRun(run domain.Run) map[string]any {
	jobMessage := ""
	if run.ErrorMessage != "" {
		jobMessage = run.ErrorMessage
	}
	result := map[string]any{
		"id": run.ID, "jobId": run.JobID, "jobName": "", "jobGroup": "DEFAULT",
		"invokeTarget": "", "jobMessage": jobMessage, "exceptionInfo": run.ErrorMessage,
		"status": legacySchedulerStatusCode(run.Status), "createTime": run.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if run.StartedAt != nil {
		result["startTime"] = run.StartedAt.Format("2006-01-02 15:04:05")
	}
	if run.FinishedAt != nil {
		result["stopTime"] = run.FinishedAt.Format("2006-01-02 15:04:05")
	}
	return result
}

func schedulerInput(input legacyJobInput) domain.Input {
	maxAttempts := input.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 3
	}
	timeout := input.TimeoutSeconds
	if timeout == 0 {
		timeout = 60
	}
	return domain.Input{
		Name: firstSchedulerString(input.JobName, input.Name), Handler: firstSchedulerString(input.InvokeTarget, input.Handler),
		CronExpression: input.CronExpression, Payload: input.Payload, MaxAttempts: maxAttempts,
		TimeoutSeconds: timeout, ConcurrencyKey: input.ConcurrencyKey,
	}
}

func legacySchedulerStatus(value string) string {
	switch value {
	case "1", "active":
		return "active"
	case "0", "paused":
		return "paused"
	default:
		return value
	}
}

func legacySchedulerStatusCode(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case string:
		if typed == "active" || typed == "success" || typed == "completed" {
			return 1
		}
		if typed == "1" {
			return 1
		}
		return 0
	case bool:
		if typed {
			return 1
		}
	}
	return 0
}

func schedulerUUID(value any) (uuid.UUID, error) {
	switch typed := value.(type) {
	case string:
		return uuid.Parse(typed)
	case uuid.UUID:
		return typed, nil
	default:
		return uuid.Nil, domainValidation("id is required")
	}
}

func firstSchedulerString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func domainValidation(message string) error {
	return &schedulerValidationError{message: message}
}

type schedulerValidationError struct{ message string }

func (e *schedulerValidationError) Error() string { return e.message }

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	items, total, err := h.service.List(r.Context(), r.URL.Query().Get("status"), page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, httpserver.NewPage(items, total, page, pageSize))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.Get(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var input domain.Input
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	item, err := h.service.Create(r.Context(), actor.UserID, input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	var input domain.Input
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	item, err := h.service.Update(r.Context(), id, input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) pause(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.Pause(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) resume(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.Resume(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) run(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	run, err := h.service.RunNow(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusAccepted, run)
}

func (h *Handler) runs(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	h.writeRuns(w, r, id)
}

func (h *Handler) writeRuns(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	runs, err := h.service.Runs(r.Context(), id, page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, httpserver.NewPage(runs, int64(len(runs)), page, pageSize))
}
