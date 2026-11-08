package http

import (
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"mars-admin-go/internal/modules/codegen/application"
	"mars-admin-go/internal/modules/codegen/domain"
	"mars-admin-go/internal/platform/apperror"
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
	router.With(httpserver.RequirePermission("codegen:read")).Get("/tables", h.tables)
	router.With(httpserver.RequirePermission("codegen:manage")).Post("/preview", h.preview)
	router.With(httpserver.RequirePermission("codegen:read")).Get("/imported", h.importedTables)
	router.With(httpserver.RequirePermission("codegen:manage")).Post("/imported", h.importTables)
	router.With(httpserver.RequirePermission("codegen:read")).Get("/imported/{id}", h.importedTable)
	router.With(httpserver.RequirePermission("codegen:manage")).Put("/imported/{id}", h.updateImportedTable)
	router.With(httpserver.RequirePermission("codegen:manage")).Delete("/imported/{id}", h.deleteImportedTable)
	router.With(httpserver.RequirePermission("codegen:manage")).Get("/imported/{id}/preview", h.previewImportedTable)
	router.With(httpserver.RequirePermission("codegen:manage")).Post("/imported/{id}/sync", h.syncImportedTable)
	router.With(httpserver.RequirePermission("codegen:manage")).Get("/download", h.legacyDownload)
	return router
}

func (h *Handler) LegacyRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth)
	router.With(httpserver.RequirePermission("codegen:read")).Get("/page", h.legacyPage)
	router.With(httpserver.RequirePermission("codegen:read")).Get("/db/list", h.legacyTables)
	router.With(httpserver.RequirePermission("codegen:manage")).Post("/import", h.legacyImport)
	router.With(httpserver.RequirePermission("codegen:manage")).Get("/preview/{id}", h.legacyPreview)
	router.With(httpserver.RequirePermission("codegen:manage")).Get("/preview-generate/{id}", h.legacyPreviewGenerate)
	router.With(httpserver.RequirePermission("codegen:manage")).Post("/generate/{id}", h.legacyGenerate)
	router.With(httpserver.RequirePermission("codegen:manage")).Get("/preview-remove/{id}", h.legacyNoFiles)
	router.With(httpserver.RequirePermission("codegen:manage")).Delete("/remove-code/{id}", h.legacyNoFiles)
	router.With(httpserver.RequirePermission("codegen:manage")).Get("/download", h.legacyDownload)
	router.With(httpserver.RequirePermission("codegen:manage")).Post("/sync/{id}", h.legacySync)
	router.With(httpserver.RequirePermission("codegen:read")).Get("/{id}", h.legacyTable)
	router.With(httpserver.RequirePermission("codegen:manage")).Put("/", h.legacyUpdateTable)
	router.With(httpserver.RequirePermission("codegen:manage")).Delete("/{id}", h.legacyDeleteTables)
	router.With(httpserver.RequirePermission("codegen:manage")).Post("/preview", h.preview)
	return router
}

func (h *Handler) legacyPage(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	items, total, err := h.service.ImportedTables(r.Context(), r.URL.Query().Get("tableName"), page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, legacyImportedTable(item))
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"list": result, "records": result, "items": result,
		"total": total, "page": page, "pageSize": pageSize,
	})
}

func (h *Handler) legacyTables(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	items, err := h.service.Tables(r.Context(), r.URL.Query().Get("schema"))
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	filter := r.URL.Query().Get("tableName")
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if filter != "" && !strings.Contains(item.Name, filter) {
			continue
		}
		result = append(result, map[string]any{
			"tableName": item.Name, "tableComment": item.Comment,
			"createTime": "", "updateTime": "",
		})
	}
	start := (page - 1) * pageSize
	if start > len(result) {
		start = len(result)
	}
	end := start + pageSize
	if end > len(result) {
		end = len(result)
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"list": result[start:end], "records": result[start:end], "items": result[start:end],
		"total": len(result), "page": page, "pageSize": pageSize,
	})
}

func (h *Handler) legacyImport(w http.ResponseWriter, r *http.Request) {
	var tables []string
	if !httpserver.DecodeJSON(w, r, &tables) {
		return
	}
	items, err := h.service.ImportTables(r.Context(), "public", tables)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, legacyImportedTable(item))
	}
	httpserver.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) legacyTable(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.ImportedTable(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyImportedTable(item))
}

func (h *Handler) legacyUpdateTable(w http.ResponseWriter, r *http.Request) {
	var input map[string]any
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	id, err := codegenUUID(input["id"])
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.ImportedTable(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	applyString := func(key string, target *string) {
		if value, ok := input[key].(string); ok {
			*target = value
		}
	}
	applyString("className", &item.ClassName)
	applyString("packageName", &item.PackageName)
	applyString("moduleName", &item.ModuleName)
	applyString("businessName", &item.BusinessName)
	applyString("functionName", &item.FunctionName)
	applyString("author", &item.Author)
	applyString("genType", &item.GenType)
	applyString("genPath", &item.GenPath)
	applyString("frontType", &item.FrontType)
	applyString("formLayout", &item.FormLayout)
	applyString("remark", &item.Remark)
	if parent, ok := input["parentMenuId"]; ok {
		item.ParentMenuID = strings.TrimSpace(strings.Trim(strings.Trim(fmt.Sprint(parent), "[]"), " "))
	}
	updated, err := h.service.UpdateImportedTable(r.Context(), item)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyImportedTable(updated))
}

func (h *Handler) legacyDeleteTables(w http.ResponseWriter, r *http.Request) {
	raw := chi.URLParam(r, "id")
	parts := strings.Split(raw, ",")
	ids := make([]uuid.UUID, 0, len(parts))
	for _, part := range parts {
		id, err := uuid.Parse(strings.TrimSpace(part))
		if err != nil {
			httpserver.WriteError(w, r, h.logger, err)
			return
		}
		ids = append(ids, id)
	}
	if err := h.service.DeleteImportedTables(r.Context(), ids); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) legacyPreview(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	output, err := h.service.PreviewImported(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, output.Files)
}

func (h *Handler) legacyPreviewGenerate(w http.ResponseWriter, r *http.Request) {
	h.writePreviewFiles(w, r)
}

func (h *Handler) legacyGenerate(w http.ResponseWriter, r *http.Request) {
	h.writePreviewFiles(w, r)
}

func (h *Handler) writePreviewFiles(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	output, err := h.service.PreviewImported(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	files := make([]string, 0, len(output.Files))
	for name := range output.Files {
		files = append(files, name)
	}
	httpserver.WriteJSON(w, http.StatusOK, files)
}

func (h *Handler) legacyNoFiles(w http.ResponseWriter, r *http.Request) {
	httpserver.WriteError(w, r, h.logger, apperror.New(
		apperror.KindNotConfigured,
		"legacy_codegen_files_not_supported",
		"Legacy generated-file inspection and deletion are not supported. Use the imported table preview and delete endpoints.",
	))
}

func (h *Handler) legacySync(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.SyncImported(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyImportedTable(item))
}

func (h *Handler) legacyDownload(w http.ResponseWriter, r *http.Request) {
	ids := strings.Split(r.URL.Query().Get("ids"), ",")
	if len(ids) == 0 || len(ids) > 50 {
		httpserver.WriteError(w, r, h.logger, apperror.Validation("ids", "Select between 1 and 50 imported tables."))
		return
	}
	outputs := make([]domain.Output, 0, len(ids))
	for _, raw := range ids {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			httpserver.WriteError(w, r, h.logger, err)
			return
		}
		output, err := h.service.PreviewImported(r.Context(), id)
		if err != nil {
			httpserver.WriteError(w, r, h.logger, err)
			return
		}
		outputs = append(outputs, output)
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="mars-admin-codegen.zip"`)
	archive := zip.NewWriter(w)
	for _, output := range outputs {
		for name, content := range output.Files {
			entry, err := archive.Create(name)
			if err != nil {
				return
			}
			if _, err := io.WriteString(entry, content); err != nil {
				return
			}
		}
	}
	if err := archive.Close(); err != nil {
		h.logger.ErrorContext(r.Context(), "close codegen zip", "error", err)
		return
	}
}

func legacyImportedTable(item domain.ImportedTable) map[string]any {
	columns := make([]map[string]any, 0, len(item.Columns))
	for _, column := range item.Columns {
		columns = append(columns, map[string]any{
			"columnName": column.Name, "columnComment": column.Comment,
			"columnType": column.DataType, "javaType": column.DataType,
			"javaField": column.Name, "isPk": boolNumber(column.PrimaryKey),
			"isIncrement": 0, "isRequired": boolNumber(!column.Nullable),
			"isInsert": 1, "isEdit": 1, "isList": 1, "isQuery": 0,
			"queryType": "EQ", "htmlType": "input", "dictType": "", "sort": 0,
		})
	}
	return map[string]any{
		"id": item.ID, "tableName": item.Name, "tableComment": item.Comment,
		"className": item.ClassName, "packageName": item.PackageName,
		"moduleName": item.ModuleName, "businessName": item.BusinessName,
		"functionName": item.FunctionName, "author": item.Author,
		"genType": item.GenType, "genPath": item.GenPath, "frontType": item.FrontType,
		"formLayout": item.FormLayout, "parentMenuId": item.ParentMenuID,
		"remark": item.Remark, "columns": columns,
		"createTime": item.CreatedAt.Format("2006-01-02 15:04:05"),
		"updateTime": item.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func codegenUUID(value any) (uuid.UUID, error) {
	if text, ok := value.(string); ok {
		return uuid.Parse(text)
	}
	if id, ok := value.(uuid.UUID); ok {
		return id, nil
	}
	return uuid.Nil, &codegenValidationError{message: "A valid table id is required."}
}

func boolNumber(value bool) int {
	if value {
		return 1
	}
	return 0
}

type codegenValidationError struct{ message string }

func (e *codegenValidationError) Error() string { return e.message }

func (h *Handler) tables(w http.ResponseWriter, r *http.Request) {
	schema := r.URL.Query().Get("schema")
	items, err := h.service.Tables(r.Context(), schema)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) preview(w http.ResponseWriter, r *http.Request) {
	var input domain.GenerateInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	output, err := h.service.Preview(r.Context(), input)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, output)
}

func (h *Handler) importedTables(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	items, total, err := h.service.ImportedTables(r.Context(), r.URL.Query().Get("query"), page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, httpserver.NewPage(items, total, page, pageSize))
}

func (h *Handler) importTables(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Schema string   `json:"schema"`
		Tables []string `json:"tables"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	items, err := h.service.ImportTables(r.Context(), input.Schema, input.Tables)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, items)
}

func (h *Handler) importedTable(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.ImportedTable(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) updateImportedTable(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.ImportedTable(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	if !httpserver.DecodeJSON(w, r, &item) {
		return
	}
	item.ID = id
	updated, err := h.service.UpdateImportedTable(r.Context(), item)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteImportedTable(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	if err := h.service.DeleteImportedTables(r.Context(), []uuid.UUID{id}); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) previewImportedTable(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	output, err := h.service.PreviewImported(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, output)
}

func (h *Handler) syncImportedTable(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	item, err := h.service.SyncImported(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, item)
}
