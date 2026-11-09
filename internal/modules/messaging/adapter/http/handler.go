package http

import (
	"log/slog"
	"net/http"
	"strconv"

	"mars-admin-go/internal/modules/messaging/application"
	"mars-admin-go/internal/modules/messaging/domain"
	"mars-admin-go/internal/platform/httpserver"
	"mars-admin-go/internal/platform/requestctx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *application.Service
	auth    func(http.Handler) http.Handler
	logger  *slog.Logger
	online  func(uuid.UUID) bool
}

func New(service *application.Service, auth func(http.Handler) http.Handler, logger *slog.Logger, online func(uuid.UUID) bool) *Handler {
	return &Handler{service: service, auth: auth, logger: logger, online: online}
}

func (h *Handler) Routes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth)
	router.With(httpserver.RequirePermission("message:read")).Get("/messages/users", h.users)
	router.With(httpserver.RequirePermission("message:send")).Post("/messages", h.send)
	router.With(httpserver.RequirePermission("message:read")).Get("/messages/{peerId}", h.history)
	router.With(httpserver.RequirePermission("message:read")).Post("/messages/{peerId}/read", h.markRead)
	router.With(httpserver.RequirePermission("message:read")).Get("/messages/unread", h.unread)
	router.With(httpserver.RequirePermission("message:send")).Post("/messages/block/{targetId}", h.block)
	router.With(httpserver.RequirePermission("message:send")).Delete("/messages/block/{targetId}", h.unblock)
	router.With(httpserver.RequirePermission("message:read")).Get("/messages/blocked", h.blocked)

	router.With(httpserver.RequirePermission("message:read")).Get("/groups", h.groups)
	router.With(httpserver.RequirePermission("message:send")).Post("/groups", h.createGroup)
	router.With(httpserver.RequirePermission("message:read")).Get("/groups/{groupId}", h.group)
	router.With(httpserver.RequirePermission("message:read")).Get("/groups/{groupId}/members", h.members)
	router.With(httpserver.RequirePermission("message:send")).Post("/groups/{groupId}/members", h.addMembers)
	router.With(httpserver.RequirePermission("message:send")).Delete("/groups/{groupId}/members/{memberId}", h.removeMember)
	router.With(httpserver.RequirePermission("message:send")).Post("/groups/{groupId}/members/{memberId}/role", h.memberRole)
	router.With(httpserver.RequirePermission("message:send")).Post("/groups/{groupId}/members/{memberId}/mute", h.muteMember)
	router.With(httpserver.RequirePermission("message:send")).Post("/groups/{groupId}/transfer/{newOwnerId}", h.transfer)
	router.With(httpserver.RequirePermission("message:send")).Post("/groups/{groupId}/quit", h.quit)
	router.With(httpserver.RequirePermission("message:send")).Delete("/groups/{groupId}", h.dissolve)
	router.With(httpserver.RequirePermission("message:send")).Post("/groups/{groupId}/messages", h.sendGroup)
	router.With(httpserver.RequirePermission("message:read")).Get("/groups/{groupId}/messages", h.groupMessages)
	return router
}

func (h *Handler) users(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	users, err := h.service.Users(r.Context(), actor.UserID, r.URL.Query().Get("keyword"), 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	for i := range users {
		users[i].Online = h.online != nil && h.online(users[i].ID)
	}
	httpserver.WriteJSON(w, http.StatusOK, users)
}

func (h *Handler) send(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ReceiverID      uuid.UUID      `json:"receiverId"`
		ClientMessageID string         `json:"clientMessageId"`
		MessageType     string         `json:"messageType"`
		Content         string         `json:"content"`
		Metadata        map[string]any `json:"metadata"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	message, err := h.service.SendDirect(r.Context(), domain.SendInput{
		SenderID: actor.UserID, ReceiverID: input.ReceiverID, ClientMessageID: input.ClientMessageID,
		MessageType: input.MessageType, Content: input.Content, Metadata: input.Metadata,
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, message)
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	peerID, err := httpserver.PathUUID(r, "peerId")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	cursor := parseInt64(r.URL.Query().Get("before"))
	limit := parseInt(r.URL.Query().Get("limit"), 30)
	messages, err := h.service.DirectHistory(r.Context(), actor.UserID, peerID, cursor, limit)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, messages)
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	peerID, err := httpserver.PathUUID(r, "peerId")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.service.MarkDirectRead(r.Context(), actor.UserID, peerID); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) unread(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	unread, err := h.service.Unread(r.Context(), actor.UserID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	if r.URL.Path == "/api/v1/messages/unread" {
		httpserver.WriteJSON(w, http.StatusOK, unread)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{"count": unread["chat"], "chatCount": unread["chat"]})
}

func (h *Handler) block(w http.ResponseWriter, r *http.Request) {
	targetID, err := httpserver.PathUUID(r, "targetId")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.service.Block(r.Context(), actor.UserID, targetID); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) unblock(w http.ResponseWriter, r *http.Request) {
	targetID, err := httpserver.PathUUID(r, "targetId")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.service.Unblock(r.Context(), actor.UserID, targetID); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) blocked(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	users, err := h.service.Blocked(r.Context(), actor.UserID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, users)
}

func (h *Handler) groups(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	groups, err := h.service.Groups(r.Context(), actor.UserID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, groups)
}

func (h *Handler) createGroup(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateGroupInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	input.OwnerID = actor.UserID
	group, err := h.service.CreateGroup(r.Context(), input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, group)
}

func (h *Handler) group(w http.ResponseWriter, r *http.Request) {
	groupID, err := httpserver.PathUUID(r, "groupId")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	group, err := h.service.Group(r.Context(), groupID, actor.UserID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, group)
}

func (h *Handler) members(w http.ResponseWriter, r *http.Request) {
	groupID, err := httpserver.PathUUID(r, "groupId")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	members, err := h.service.Members(r.Context(), groupID, actor.UserID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, members)
}

func (h *Handler) addMembers(w http.ResponseWriter, r *http.Request) {
	groupID, err := httpserver.PathUUID(r, "groupId")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	var input struct {
		UserIDs []uuid.UUID `json:"userIds"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.service.AddMembers(r.Context(), groupID, actor.UserID, input.UserIDs); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	groupID, memberID, ok := h.groupMemberIDs(w, r)
	if !ok {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.service.RemoveMember(r.Context(), groupID, actor.UserID, memberID); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) memberRole(w http.ResponseWriter, r *http.Request) {
	groupID, memberID, ok := h.groupMemberIDs(w, r)
	if !ok {
		return
	}
	var input struct {
		Role    string `json:"role"`
		IsAdmin bool   `json:"isAdmin"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	role := input.Role
	if role == "" {
		if input.IsAdmin {
			role = "admin"
		} else {
			role = "member"
		}
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.service.SetMemberRole(r.Context(), groupID, actor.UserID, memberID, role); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) muteMember(w http.ResponseWriter, r *http.Request) {
	groupID, memberID, ok := h.groupMemberIDs(w, r)
	if !ok {
		return
	}
	var input struct {
		Muted bool `json:"muted"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.service.MuteMember(r.Context(), groupID, actor.UserID, memberID, input.Muted); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) transfer(w http.ResponseWriter, r *http.Request) {
	groupID, err := httpserver.PathUUID(r, "groupId")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	newOwnerID, err := httpserver.PathUUID(r, "newOwnerId")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.service.TransferOwner(r.Context(), groupID, actor.UserID, newOwnerID); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) quit(w http.ResponseWriter, r *http.Request) {
	groupID, err := httpserver.PathUUID(r, "groupId")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.service.QuitGroup(r.Context(), groupID, actor.UserID); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) dissolve(w http.ResponseWriter, r *http.Request) {
	groupID, err := httpserver.PathUUID(r, "groupId")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.service.DissolveGroup(r.Context(), groupID, actor.UserID); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) sendGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := httpserver.PathUUID(r, "groupId")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	var input struct {
		ClientMessageID string         `json:"clientMessageId"`
		MessageType     string         `json:"messageType"`
		Content         string         `json:"content"`
		Metadata        map[string]any `json:"metadata"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	message, err := h.service.SendGroup(r.Context(), groupID, actor.UserID, input.ClientMessageID, input.MessageType, input.Content, input.Metadata)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, message)
}

func (h *Handler) groupMessages(w http.ResponseWriter, r *http.Request) {
	groupID, err := httpserver.PathUUID(r, "groupId")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	messages, err := h.service.GroupHistory(r.Context(), groupID, actor.UserID, parseInt64(r.URL.Query().Get("before")), parseInt(r.URL.Query().Get("limit"), 50))
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, messages)
}

func (h *Handler) groupMemberIDs(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	groupID, err := httpserver.PathUUID(r, "groupId")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return uuid.Nil, uuid.Nil, false
	}
	memberID, err := httpserver.PathUUID(r, "memberId")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return uuid.Nil, uuid.Nil, false
	}
	return groupID, memberID, true
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseInt64(value string) int64 {
	if value == "" {
		return 0
	}
	parsed, _ := strconv.ParseInt(value, 10, 64)
	return parsed
}
