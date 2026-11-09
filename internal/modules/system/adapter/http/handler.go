package http

import (
	"log/slog"
	"net/http"

	"mars-admin-go/internal/modules/system/application"
	"mars-admin-go/internal/modules/system/domain"
	"mars-admin-go/internal/platform/httpserver"
	"mars-admin-go/internal/platform/requestctx"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	dictionaries *application.DictionaryService
	settings     *application.SettingsService
	auth         func(http.Handler) http.Handler
	logger       *slog.Logger
}

func New(dictionaries *application.DictionaryService, settings *application.SettingsService, auth func(http.Handler) http.Handler, logger *slog.Logger) *Handler {
	return &Handler{dictionaries: dictionaries, settings: settings, auth: auth, logger: logger}
}

func (h *Handler) Routes() chi.Router {
	router := chi.NewRouter()
	router.Get("/settings/public", h.publicSettings)
	router.Group(func(protected chi.Router) {
		protected.Use(h.auth)
		protected.With(httpserver.RequirePermission("dictionary:read")).Get("/dictionaries", h.dictionaryTypes)
		protected.With(httpserver.RequirePermission("dictionary:read")).Get("/dictionaries/{id}/items", h.dictionaryItems)
		protected.With(httpserver.RequirePermission("dictionary:manage")).Post("/dictionaries", h.createDictionaryType)
		protected.With(httpserver.RequirePermission("dictionary:manage")).Put("/dictionaries/{id}", h.updateDictionaryType)
		protected.With(httpserver.RequirePermission("dictionary:manage")).Delete("/dictionaries/{id}", h.deleteDictionaryType)
		protected.With(httpserver.RequirePermission("dictionary:manage")).Post("/dictionaries/items", h.createDictionaryItem)
		protected.With(httpserver.RequirePermission("dictionary:manage")).Put("/dictionaries/items/{id}", h.updateDictionaryItem)
		protected.With(httpserver.RequirePermission("dictionary:manage")).Delete("/dictionaries/items/{id}", h.deleteDictionaryItem)
		protected.With(httpserver.RequirePermission("settings:read")).Get("/settings", h.settingGroups)
		protected.With(httpserver.RequirePermission("settings:read")).Get("/settings/{code}", h.settingGroup)
		protected.With(httpserver.RequirePermission("settings:manage")).Put("/settings/{code}", h.updateSettings)
	})
	return router
}

func (h *Handler) dictionaryTypes(w http.ResponseWriter, r *http.Request) {
	items, err := h.dictionaries.Types(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) dictionaryItems(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	items, err := h.dictionaries.Items(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) createDictionaryType(w http.ResponseWriter, r *http.Request) {
	var input domain.DictionaryTypeInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	item, err := h.dictionaries.CreateType(r.Context(), input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) updateDictionaryType(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	var input domain.DictionaryTypeInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	item, err := h.dictionaries.UpdateType(r.Context(), id, input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) deleteDictionaryType(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	if err := h.dictionaries.DeleteType(r.Context(), id); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createDictionaryItem(w http.ResponseWriter, r *http.Request) {
	var input domain.DictionaryItemInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	item, err := h.dictionaries.CreateItem(r.Context(), input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) updateDictionaryItem(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	var input domain.DictionaryItemInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	item, err := h.dictionaries.UpdateItem(r.Context(), id, input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) deleteDictionaryItem(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	if err := h.dictionaries.DeleteItem(r.Context(), id); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) settingGroups(w http.ResponseWriter, r *http.Request) {
	items, err := h.settings.Groups(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) settingGroup(w http.ResponseWriter, r *http.Request) {
	group, err := h.settings.Group(r.Context(), chi.URLParam(r, "code"), true)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, group)
}

func (h *Handler) publicSettings(w http.ResponseWriter, r *http.Request) {
	values, err := h.settings.Public(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, values)
}

func (h *Handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	var input map[string]any
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.settings.Update(r.Context(), chi.URLParam(r, "code"), input, actor.UserID); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	group, err := h.settings.Group(r.Context(), chi.URLParam(r, "code"), true)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, group)
}
