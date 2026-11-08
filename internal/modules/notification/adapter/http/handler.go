package http

import (
	"log/slog"
	"net/http"

	"mars-admin-go/internal/modules/notification/application"
	"mars-admin-go/internal/modules/notification/domain"
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
	router.With(httpserver.RequirePermission("notice:read")).Get("/", h.list)
	router.With(httpserver.RequirePermission("notice:read")).Get("/my", h.my)
	router.With(httpserver.RequirePermission("notice:read")).Get("/unread", h.unread)
	router.With(httpserver.RequirePermission("notice:read")).Get("/{id}", h.get)
	router.With(httpserver.RequirePermission("notice:manage")).Post("/", h.create)
	router.With(httpserver.RequirePermission("notice:manage")).Put("/{id}", h.update)
	router.With(httpserver.RequirePermission("notice:manage")).Post("/{id}/publish", h.publish)
	router.With(httpserver.RequirePermission("notice:manage")).Post("/{id}/withdraw", h.withdraw)
	router.With(httpserver.RequirePermission("notice:read")).Post("/{id}/read", h.read)
	return router
}

func (h *Handler) LegacyRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth)
	router.With(httpserver.RequirePermission("notice:read")).Get("/page", h.legacyList)
	router.With(httpserver.RequirePermission("notice:read")).Get("/my", h.legacyMy)
	router.With(httpserver.RequirePermission("notice:read")).Get("/unread-count", h.legacyUnread)
	router.With(httpserver.RequirePermission("notice:read")).Get("/channels", h.legacyChannels)
	router.With(httpserver.RequirePermission("notice:read")).Post("/read-all", h.legacyReadAll)
	router.With(httpserver.RequirePermission("notice:manage")).Get("/{id}/send-logs", h.legacySendLogs)
	router.With(httpserver.RequirePermission("notice:manage")).Post("/{id}/retry", h.legacyRetry)
	router.With(httpserver.RequirePermission("notice:read")).Get("/{id}", h.legacyGet)
	router.With(httpserver.RequirePermission("notice:manage")).Post("/", h.legacyCreate)
	router.With(httpserver.RequirePermission("notice:manage")).Put("/", h.legacyUpdate)
	router.With(httpserver.RequirePermission("notice:manage")).Post("/{id}/publish", h.publish)
	router.With(httpserver.RequirePermission("notice:read")).Post("/{id}/read", h.read)
	router.With(httpserver.RequirePermission("notice:manage")).Delete("/{id}", h.legacyDelete)
	return router
}

type legacyNoticeInput struct {
	ID         any         `json:"id"`
	Title      string      `json:"title"`
	Content    string      `json:"content"`
	NoticeType int16       `json:"noticeType"`
	Channels   []string    `json:"channels"`
	TargetType any         `json:"targetType"`
	TargetIDs  []uuid.UUID `json:"targetIds"`
}

func (h *Handler) legacyList(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	var (
		items []domain.Notice
		total int64
	)
	if canManageNotices(actor) {
		status := legacyNoticeStatus(r.URL.Query().Get("status"))
		items, total, err = h.service.List(r.Context(), status, page, pageSize)
	} else {
		items, err = h.service.My(r.Context(), actor.UserID, page, pageSize)
		if err == nil {
			total, err = h.service.MyCount(r.Context(), actor.UserID)
		}
	}
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, legacyNotice(item))
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"list": result, "records": result, "items": result,
		"total": total, "page": page, "pageSize": pageSize,
	})
}

func (h *Handler) legacyMy(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	items, err := h.service.My(r.Context(), actor.UserID, page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	total, err := h.service.MyCount(r.Context(), actor.UserID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, legacyNotice(item))
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
	actor, _ := requestctx.ActorFrom(r.Context())
	item, err := h.service.GetForActor(r.Context(), actor, id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyNotice(item))
}

func (h *Handler) legacyCreate(w http.ResponseWriter, r *http.Request) {
	var input legacyNoticeInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	item, err := h.service.Create(r.Context(), actor.UserID, domain.Input{
		Title: input.Title, Content: input.Content, NoticeType: input.NoticeType,
		Channels: input.Channels, TargetType: legacyNoticeTarget(input.TargetType), TargetIDs: input.TargetIDs,
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, legacyNotice(item))
}

func (h *Handler) legacyUpdate(w http.ResponseWriter, r *http.Request) {
	var input legacyNoticeInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	id, err := notificationUUID(input.ID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.Update(r.Context(), id, domain.Input{
		Title: input.Title, Content: input.Content, NoticeType: input.NoticeType,
		Channels: input.Channels, TargetType: legacyNoticeTarget(input.TargetType), TargetIDs: input.TargetIDs,
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyNotice(item))
}

func (h *Handler) legacyDelete(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) legacyChannels(w http.ResponseWriter, _ *http.Request) {
	channels := []map[string]any{
		{"code": "station", "name": "站内信", "enabled": true},
		{"code": "email", "name": "邮件", "enabled": false},
		{"code": "sms", "name": "短信", "enabled": false},
		{"code": "dingtalk", "name": "钉钉", "enabled": false},
		{"code": "feishu", "name": "飞书", "enabled": false},
		{"code": "wechat_work", "name": "企业微信", "enabled": false},
	}
	httpserver.WriteJSON(w, http.StatusOK, channels)
}

func (h *Handler) legacyReadAll(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.service.MarkAllRead(r.Context(), actor.UserID); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) legacySendLogs(w http.ResponseWriter, _ *http.Request) {
	httpserver.WriteJSON(w, http.StatusOK, []map[string]any{})
}

func (h *Handler) legacyRetry(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func legacyNotice(item domain.Notice) map[string]any {
	result := map[string]any{
		"id": item.ID, "title": item.Title, "content": item.Content,
		"noticeType": item.NoticeType, "channels": item.Channels,
		"targetType": legacyNoticeTargetCode(item.TargetType), "targetIds": item.TargetIDs,
		"status":     legacyNoticeStatusCode(item.Status),
		"createTime": item.CreatedAt.Format("2006-01-02 15:04:05"),
		"updateTime": item.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if item.CreatedBy != nil {
		result["createBy"] = *item.CreatedBy
	}
	if item.ReadAt != nil {
		result["readTime"] = item.ReadAt.Format("2006-01-02 15:04:05")
	}
	return result
}

func legacyNoticeStatus(value string) string {
	switch value {
	case "0":
		return "draft"
	case "1":
		return "published"
	case "2":
		return "withdrawn"
	default:
		return value
	}
}

func canManageNotices(actor requestctx.Actor) bool {
	return actor.IsSuperAdmin || actor.HasPermission("notice:manage")
}

func legacyNoticeStatusCode(value string) int {
	switch value {
	case "published":
		return 1
	case "withdrawn":
		return 2
	default:
		return 0
	}
}

func legacyNoticeTarget(value any) string {
	switch typed := value.(type) {
	case float64:
		switch typed {
		case 1:
			return "users"
		case 2:
			return "departments"
		default:
			return "all"
		}
	case string:
		switch typed {
		case "1":
			return "users"
		case "2":
			return "departments"
		case "3":
			return "all"
		default:
			return typed
		}
	default:
		return "all"
	}
}

func legacyNoticeTargetCode(value string) int {
	switch value {
	case "users":
		return 1
	case "departments":
		return 2
	default:
		return 3
	}
}

func notificationUUID(value any) (uuid.UUID, error) {
	if text, ok := value.(string); ok {
		return uuid.Parse(text)
	}
	if id, ok := value.(uuid.UUID); ok {
		return id, nil
	}
	return uuid.Nil, &notificationValidationError{message: "A valid notice id is required."}
}

type notificationValidationError struct{ message string }

func (e *notificationValidationError) Error() string { return e.message }

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	var (
		items []domain.Notice
		total int64
	)
	if canManageNotices(actor) {
		items, total, err = h.service.List(r.Context(), r.URL.Query().Get("status"), page, pageSize)
	} else {
		items, err = h.service.My(r.Context(), actor.UserID, page, pageSize)
		if err == nil {
			total, err = h.service.MyCount(r.Context(), actor.UserID)
		}
	}
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, httpserver.NewPage(items, total, page, pageSize))
}

func (h *Handler) my(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	items, err := h.service.My(r.Context(), actor.UserID, page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	total, err := h.service.MyCount(r.Context(), actor.UserID)
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
	actor, _ := requestctx.ActorFrom(r.Context())
	item, err := h.service.GetForActor(r.Context(), actor, id)
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

func (h *Handler) withdraw(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	if err := h.service.Withdraw(r.Context(), id); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) read(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.service.MarkRead(r.Context(), id, actor.UserID); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) unread(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	count, err := h.service.Unread(r.Context(), actor.UserID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]int64{"unread": count, "count": count})
}

func (h *Handler) legacyUnread(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	count, err := h.service.Unread(r.Context(), actor.UserID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, count)
}
