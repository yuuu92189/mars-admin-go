package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"mars-admin-go/internal/modules/system/application"
	"mars-admin-go/internal/modules/system/domain"
	"mars-admin-go/internal/platform/apperror"
	"mars-admin-go/internal/platform/httpserver"
	"mars-admin-go/internal/platform/requestctx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

func (h *Handler) LegacyRoutes() chi.Router {
	router := chi.NewRouter()
	router.Get("/config-group/public", h.legacyPublicSettings)
	router.Group(func(protected chi.Router) {
		protected.Use(h.auth)
		protected.With(httpserver.RequirePermission("dictionary:read")).Get("/dict/type/page", h.legacyDictionaryTypesPage)
		protected.With(httpserver.RequirePermission("dictionary:read")).Get("/dict/type/list", h.legacyDictionaryTypes)
		protected.With(httpserver.RequirePermission("dictionary:read")).Get("/dict/type/{id}", h.legacyDictionaryType)
		protected.With(httpserver.RequirePermission("dictionary:manage")).Post("/dict/type", h.legacyCreateDictionaryType)
		protected.With(httpserver.RequirePermission("dictionary:manage")).Put("/dict/type", h.legacyUpdateDictionaryType)
		protected.With(httpserver.RequirePermission("dictionary:manage")).Delete("/dict/type/{id}", h.deleteDictionaryType)
		protected.With(httpserver.RequirePermission("dictionary:read")).Get("/dict/data/page", h.legacyDictionaryItemsPage)
		protected.With(httpserver.RequirePermission("dictionary:read")).Get("/dict/data/type/{code}", h.legacyDictionaryItemsByCode)
		protected.With(httpserver.RequirePermission("dictionary:manage")).Post("/dict/data", h.legacyCreateDictionaryItem)
		protected.With(httpserver.RequirePermission("dictionary:manage")).Put("/dict/data", h.legacyUpdateDictionaryItem)
		protected.With(httpserver.RequirePermission("dictionary:manage")).Delete("/dict/data/{id}", h.deleteDictionaryItem)
		protected.With(httpserver.RequirePermission("settings:read")).Get("/config-group/list", h.legacySettingGroups)
		protected.With(httpserver.RequirePermission("settings:read")).Get("/config-group/sms-logs/recent", h.legacyEmptySMSLogs)
		protected.With(httpserver.RequirePermission("settings:read")).Get("/config-group/sms-logs", h.legacyEmptySMSLogsPage)
		protected.With(httpserver.RequirePermission("settings:manage")).Post("/config-group/refresh", h.legacyRefreshSettings)
		protected.With(httpserver.RequirePermission("settings:read")).Get("/config-group/{code}", h.legacySettingGroup)
		protected.With(httpserver.RequirePermission("settings:manage")).Post("/config-group/{code}", h.legacyUpdateSettings)
		protected.With(httpserver.RequirePermission("settings:read")).Get("/file-config/page", h.legacyFileConfigPage)
		protected.With(httpserver.RequirePermission("settings:read")).Get("/file-config/list", h.legacyFileConfigList)
		protected.With(httpserver.RequirePermission("settings:read")).Get("/file-config/master", h.legacyFileConfigMaster)
		protected.With(httpserver.RequirePermission("settings:manage")).Post("/file-config/test", h.legacyFileConfigTest)
		protected.With(httpserver.RequirePermission("settings:manage")).Post("/file-config", h.legacyFileConfigSave)
		protected.With(httpserver.RequirePermission("settings:manage")).Put("/file-config", h.legacyFileConfigSave)
		protected.With(httpserver.RequirePermission("settings:read")).Get("/file-config/{id}", h.legacyFileConfigMaster)
		protected.With(httpserver.RequirePermission("settings:manage")).Delete("/file-config/{id}", h.legacyFileConfigDelete)
		protected.With(httpserver.RequirePermission("settings:manage")).Put("/file-config/master/{id}", h.legacyFileConfigSetMaster)
	})
	return router
}

type legacyDictionaryTypeInput struct {
	ID       any    `json:"id"`
	DictName string `json:"dictName"`
	DictType string `json:"dictType"`
	Status   any    `json:"status"`
	Remark   string `json:"remark"`
}

type legacyDictionaryItemInput struct {
	ID        any    `json:"id"`
	TypeID    any    `json:"typeId"`
	DictType  string `json:"dictType"`
	DictLabel string `json:"dictLabel"`
	DictValue string `json:"dictValue"`
	Sort      int32  `json:"sort"`
	CssClass  string `json:"cssClass"`
	ListClass string `json:"listClass"`
	IsDefault any    `json:"isDefault"`
	Status    any    `json:"status"`
	Remark    string `json:"remark"`
}

func (h *Handler) legacyDictionaryTypes(w http.ResponseWriter, r *http.Request) {
	items, err := h.dictionaries.Types(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, legacyDictionaryType(item))
	}
	httpserver.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) legacyDictionaryTypesPage(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	items, err := h.dictionaries.Types(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	filterName := strings.ToLower(r.URL.Query().Get("dictName"))
	filterCode := strings.ToLower(r.URL.Query().Get("dictType"))
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if filterName != "" && !strings.Contains(strings.ToLower(item.Name), filterName) {
			continue
		}
		if filterCode != "" && !strings.Contains(strings.ToLower(item.Code), filterCode) {
			continue
		}
		result = append(result, legacyDictionaryType(item))
	}
	start, end := pageBounds(len(result), page, pageSize)
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"list": result[start:end], "records": result[start:end], "items": result[start:end],
		"total": len(result), "page": page, "pageSize": pageSize,
	})
}

func (h *Handler) legacyDictionaryType(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	items, err := h.dictionaries.Types(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	for _, item := range items {
		if item.ID == id {
			httpserver.WriteJSON(w, http.StatusOK, legacyDictionaryType(item))
			return
		}
	}
	httpserver.WriteJSON(w, http.StatusNotFound, map[string]string{"detail": "Dictionary type was not found."})
}

func (h *Handler) legacyCreateDictionaryType(w http.ResponseWriter, r *http.Request) {
	var input legacyDictionaryTypeInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	item, err := h.dictionaries.CreateType(r.Context(), domain.DictionaryTypeInput{
		Code: input.DictType, Name: input.DictName, Description: input.Remark,
		Status: legacyDictionaryStatus(input.Status),
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, legacyDictionaryType(item))
}

func (h *Handler) legacyUpdateDictionaryType(w http.ResponseWriter, r *http.Request) {
	var input legacyDictionaryTypeInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	id, err := requiredSystemUUID(input.ID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.dictionaries.UpdateType(r.Context(), id, domain.DictionaryTypeInput{
		Code: input.DictType, Name: input.DictName, Description: input.Remark,
		Status: legacyDictionaryStatus(input.Status),
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyDictionaryType(item))
}

func (h *Handler) legacyDictionaryItemsByCode(w http.ResponseWriter, r *http.Request) {
	items, err := h.dictionaryItemsForCode(r.Context(), chi.URLParam(r, "code"))
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, legacyDictionaryItem(item, chi.URLParam(r, "code")))
	}
	httpserver.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) legacyDictionaryItemsPage(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	code := r.URL.Query().Get("dictType")
	items, err := h.dictionaryItemsForCode(r.Context(), code)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, legacyDictionaryItem(item, code))
	}
	start, end := pageBounds(len(result), page, pageSize)
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"list": result[start:end], "records": result[start:end], "items": result[start:end],
		"total": len(result), "page": page, "pageSize": pageSize,
	})
}

func (h *Handler) legacyCreateDictionaryItem(w http.ResponseWriter, r *http.Request) {
	var input legacyDictionaryItemInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	typeID, err := h.dictionaryTypeID(r.Context(), input.DictType, input.TypeID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.dictionaries.CreateItem(r.Context(), domain.DictionaryItemInput{
		TypeID: typeID, Value: input.DictValue, Label: input.DictLabel, Sort: input.Sort,
		TagStyle: firstText(input.ListClass, input.CssClass), IsDefault: truthy(input.IsDefault),
		Status: legacyDictionaryStatus(input.Status),
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, legacyDictionaryItem(item, input.DictType))
}

func (h *Handler) legacyUpdateDictionaryItem(w http.ResponseWriter, r *http.Request) {
	var input legacyDictionaryItemInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	id, err := requiredSystemUUID(input.ID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	typeID, err := h.dictionaryTypeID(r.Context(), input.DictType, input.TypeID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.dictionaries.UpdateItem(r.Context(), id, domain.DictionaryItemInput{
		TypeID: typeID, Value: input.DictValue, Label: input.DictLabel, Sort: input.Sort,
		TagStyle: firstText(input.ListClass, input.CssClass), IsDefault: truthy(input.IsDefault),
		Status: legacyDictionaryStatus(input.Status),
	})
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyDictionaryItem(item, input.DictType))
}

func (h *Handler) dictionaryItemsForCode(ctx context.Context, code string) ([]domain.DictionaryItem, error) {
	types, err := h.dictionaries.Types(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range types {
		if item.Code == code {
			return h.dictionaries.Items(ctx, item.ID)
		}
	}
	return []domain.DictionaryItem{}, nil
}

func (h *Handler) dictionaryTypeID(ctx context.Context, code string, raw any) (uuid.UUID, error) {
	if raw != nil {
		if id, err := requiredSystemUUID(raw); err == nil {
			return id, nil
		}
	}
	types, err := h.dictionaries.Types(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	for _, item := range types {
		if item.Code == code {
			return item.ID, nil
		}
	}
	return uuid.Nil, &systemValidationError{message: "Dictionary type was not found."}
}

func legacyDictionaryType(item domain.DictionaryType) map[string]any {
	return map[string]any{
		"id": item.ID, "dictName": item.Name, "dictType": item.Code,
		"status": systemStatusCode(item.Status), "remark": item.Description,
		"createTime": item.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func legacyDictionaryItem(item domain.DictionaryItem, code string) map[string]any {
	result := map[string]any{
		"id": item.ID, "typeId": item.TypeID, "dictType": code, "dictLabel": item.Label,
		"dictValue": item.Value, "sort": item.Sort, "listClass": item.TagStyle,
		"cssClass": item.TagStyle, "isDefault": boolNumber(item.IsDefault),
		"status": systemStatusCode(item.Status), "remark": "",
	}
	return result
}

func legacyDictionaryStatus(value any) string {
	switch typed := value.(type) {
	case float64:
		if typed == 0 {
			return "disabled"
		}
	case string:
		if typed == "0" || strings.EqualFold(typed, "disabled") {
			return "disabled"
		}
	}
	return "active"
}

func systemStatusCode(value string) int {
	if value == "active" {
		return 1
	}
	return 0
}

func boolNumber(value bool) int {
	if value {
		return 1
	}
	return 0
}

func truthy(value any) bool {
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

func requiredSystemUUID(value any) (uuid.UUID, error) {
	if text, ok := value.(string); ok {
		return uuid.Parse(strings.TrimSpace(text))
	}
	if id, ok := value.(uuid.UUID); ok {
		return id, nil
	}
	return uuid.Nil, &systemValidationError{message: "A valid UUID is required."}
}

func pageBounds(total, page, pageSize int) (int, int) {
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return start, end
}

func firstText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type systemValidationError struct{ message string }

func (e *systemValidationError) Error() string { return e.message }

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

func (h *Handler) legacySettingGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.settings.Groups(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	items := make([]map[string]any, 0, len(groups))
	for _, group := range groups {
		if group.Code == "site" {
			continue
		}
		items = append(items, legacySettingGroupObject(group))
	}
	httpserver.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) legacySettingGroup(w http.ResponseWriter, r *http.Request) {
	group, err := h.settings.Group(r.Context(), chi.URLParam(r, "code"), true)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item := legacySettingGroupObject(group)
	item["configValue"] = legacyConfigValue(group.Settings)
	httpserver.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) legacyUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var input map[string]any
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	values := make(map[string]any)
	flattenSettings("", input, values)
	for key, value := range values {
		if application.IsSensitiveSettingKey(key) {
			if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
				delete(values, key)
			}
		}
	}
	if len(values) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	code := chi.URLParam(r, "code")
	if err := h.settings.Update(r.Context(), code, values, actor.UserID); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	group, err := h.settings.Group(r.Context(), code, true)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item := legacySettingGroupObject(group)
	item["configValue"] = legacyConfigValue(group.Settings)
	httpserver.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) legacyRefreshSettings(w http.ResponseWriter, r *http.Request) {
	h.settings.Invalidate(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

const legacyStorageConfigID = "019b0000-0000-7000-8000-000000020009"

func (h *Handler) legacyFileConfigPage(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.legacyFileConfigObject(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"list": []map[string]any{item}, "records": []map[string]any{item},
		"items": []map[string]any{item}, "total": 1, "page": page, "pageSize": pageSize,
	})
}

func (h *Handler) legacyFileConfigList(w http.ResponseWriter, r *http.Request) {
	item, err := h.legacyFileConfigObject(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, []map[string]any{item})
}

func (h *Handler) legacyFileConfigMaster(w http.ResponseWriter, r *http.Request) {
	item, err := h.legacyFileConfigObject(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) legacyFileConfigObject(ctx context.Context) (map[string]any, error) {
	group, err := h.settings.Group(ctx, "storage", false)
	if err != nil {
		return nil, err
	}
	values := group.Values
	storageType, _ := values["provider"].(string)
	if storageType == "" {
		storageType = "local"
	}
	domain, _ := values["domain"].(string)
	basePath, _ := values["localPath"].(string)
	item := map[string]any{
		"id": legacyStorageConfigID, "name": "Default Storage", "storageType": storageType,
		"master": 1, "domain": domain, "basePath": basePath, "bucketName": "",
		"accessKey": "", "secretKey": "", "endpoint": "", "region": "",
		"status": 1, "remark": "",
	}
	for _, key := range []string{"bucketName", "endpoint", "region", "accessKey"} {
		if value, ok := values[key]; ok {
			item[key] = value
		}
	}
	return item, nil
}

func (h *Handler) legacyFileConfigSave(w http.ResponseWriter, r *http.Request) {
	var input map[string]any
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	values := make(map[string]any)
	if storageType, ok := input["storageType"].(string); ok && storageType != "" {
		values["provider"] = storageType
	}
	mappings := map[string]string{
		"domain": "domain", "basePath": "localPath", "bucketName": "bucketName",
		"endpoint": "endpoint", "region": "region",
	}
	for source, target := range mappings {
		if value, ok := input[source]; ok {
			values[target] = value
		}
	}
	for _, key := range []string{"accessKey", "secretKey"} {
		if value, ok := input[key].(string); ok && strings.TrimSpace(value) != "" {
			values[key] = value
		}
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.settings.Update(r.Context(), "storage", values, actor.UserID); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.legacyFileConfigObject(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) legacyFileConfigDelete(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) legacyFileConfigSetMaster(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) legacyFileConfigTest(w http.ResponseWriter, r *http.Request) {
	httpserver.WriteError(w, r, h.logger, apperror.New(
		apperror.KindNotConfigured,
		"file_config_test_not_configured",
		"Storage connectivity testing is not configured.",
	))
}

func (h *Handler) legacyEmptySMSLogs(w http.ResponseWriter, _ *http.Request) {
	httpserver.WriteJSON(w, http.StatusOK, []map[string]any{})
}

func (h *Handler) legacyEmptySMSLogsPage(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"records": []map[string]any{}, "list": []map[string]any{},
		"total": 0, "pages": 0, "page": page, "pageSize": pageSize,
	})
}

func (h *Handler) legacyPublicSettings(w http.ResponseWriter, r *http.Request) {
	values, err := h.settings.Public(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	maxRetry := 5
	if value, ok := asInt(values["security.auth.login_max_attempts"]); ok {
		maxRetry = value
	}
	response := map[string]any{
		"system": map[string]any{
			"siteName": "Mars Admin Go Edition", "siteDescription": "现代化后台管理系统",
			"siteLogo": "", "copyright": "", "icp": "",
			"watermarkEnabled": true, "watermarkType": "username", "watermarkCustomText": "", "watermarkOpacity": 0.1,
		},
		"login": map[string]any{
			"captchaEnabled": false, "captchaType": "image", "maxRetryCount": maxRetry,
			"rememberMe": true, "lockTime": 30, "singleLogin": false,
		},
		"register": map[string]any{
			"enabled": false, "verifyEmail": false, "verifyPhone": false,
			"needAudit": false, "defaultRole": "user",
		},
		"password": map[string]any{
			"minLength": 8, "maxLength": 64, "requireUppercase": false, "requireLowercase": false,
			"requireNumber": false, "requireSpecial": false,
		},
		"storage": map[string]any{
			"maxSize":    10,
			"allowTypes": "jpg,jpeg,png,gif,webp,pdf,doc,docx,xls,xlsx,ppt,pptx,txt,md,mp4,mp3,zip",
		},
		"security": map[string]any{"disableDevtool": false},
	}
	for key, value := range values {
		switch key {
		case "system.siteName":
			setNested(response, "system", "siteName", value)
		case "system.siteDescription":
			setNested(response, "system", "siteDescription", value)
		case "system.siteLogo":
			setNested(response, "system", "siteLogo", value)
		case "system.copyright":
			setNested(response, "system", "copyright", value)
		case "system.icp":
			setNested(response, "system", "icp", value)
		case "system.watermarkEnabled":
			setNested(response, "system", "watermarkEnabled", value)
		case "system.watermarkType":
			setNested(response, "system", "watermarkType", value)
		case "system.watermarkCustomText":
			setNested(response, "system", "watermarkCustomText", value)
		case "system.watermarkOpacity":
			setNested(response, "system", "watermarkOpacity", value)
		case "site.site.name":
			setNested(response, "system", "siteName", value)
		case "site.site.description":
			setNested(response, "system", "siteDescription", value)
		case "site.site.logo":
			setNested(response, "system", "siteLogo", value)
		case "site.site.copyright":
			setNested(response, "system", "copyright", value)
		case "site.site.icp":
			setNested(response, "system", "icp", value)
		case "site.site.watermark_enabled":
			setNested(response, "system", "watermarkEnabled", value)
		case "site.site.watermark_type":
			setNested(response, "system", "watermarkType", value)
		case "site.site.watermark_custom_text":
			setNested(response, "system", "watermarkCustomText", value)
		case "site.site.watermark_opacity":
			setNested(response, "system", "watermarkOpacity", value)
		case "site.site.registration_enabled":
			setNested(response, "register", "enabled", value)
		case "security.auth.captcha_enabled":
			setNested(response, "login", "captchaEnabled", value)
		case "security.auth.captcha_type":
			setNested(response, "login", "captchaType", value)
		case "security.auth.remember_me":
			setNested(response, "login", "rememberMe", value)
		case "security.disable_devtool":
			setNested(response, "security", "disableDevtool", value)
		default:
			parts := strings.SplitN(key, ".", 2)
			if len(parts) == 2 {
				setNested(response, parts[0], parts[1], value)
			}
		}
	}
	httpserver.WriteJSON(w, http.StatusOK, response)
}

func legacySettingGroupObject(group domain.SettingGroup) map[string]any {
	return map[string]any{
		"id": group.ID, "groupCode": group.Code, "groupName": legacySettingGroupName(group.Code, group.Name),
		"groupIcon": "", "sort": group.Sort, "status": 1, "remark": group.Description,
	}
}

func legacyConfigValue(settings []domain.Setting) string {
	values := make(map[string]any)
	for _, setting := range settings {
		if setting.Sensitive {
			continue
		}
		deepSet(values, setting.Key, setting.Value)
	}
	encoded, _ := json.Marshal(values)
	return string(encoded)
}

func flattenSettings(prefix string, input map[string]any, output map[string]any) {
	for key, value := range input {
		next := key
		if prefix != "" {
			next = prefix + "." + key
		}
		if nested, ok := value.(map[string]any); ok {
			flattenSettings(next, nested, output)
			continue
		}
		output[next] = value
	}
}

func deepSet(target map[string]any, key string, value any) {
	parts := strings.Split(key, ".")
	current := target
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]any)
		if !ok {
			next = make(map[string]any)
			current[part] = next
		}
		current = next
	}
	current[parts[len(parts)-1]] = value
}

func setNested(target map[string]any, group, key string, value any) {
	values, ok := target[group].(map[string]any)
	if !ok {
		values = make(map[string]any)
		target[group] = values
	}
	deepSet(values, key, value)
}

func asInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}

func legacySettingGroupName(code, fallback string) string {
	// #nosec G101 -- these are UI labels, not credentials.
	names := map[string]string{
		"system": "系统设置", "register": "注册设置", "login": "登录设置", "password": "密码策略",
		"email": "邮件配置", "emailTemplate": "邮件模板", "sms": "短信配置", "smsTemplate": "短信模板",
		"storage": "存储配置", "push": "推送配置", "thirdParty": "第三方登录", "wechatMiniProgram": "微信小程序",
		"wechatMp": "微信公众号", "payment": "支付配置", "security": "安全配置", "other": "其他配置",
		"site": "站点设置",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return fallback
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
