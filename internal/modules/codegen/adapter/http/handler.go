package http

import (
	"archive/zip"
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
	router.With(httpserver.RequirePermission("codegen:manage")).Get("/download", h.download)
	return router
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
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
