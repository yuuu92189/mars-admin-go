package http

import (
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path"
	"strconv"

	"mars-admin-go/internal/modules/storage/application"
	storagedomain "mars-admin-go/internal/modules/storage/domain"
	"mars-admin-go/internal/platform/apperror"
	"mars-admin-go/internal/platform/httpserver"
	"mars-admin-go/internal/platform/requestctx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *application.Service
	auth    func(http.Handler) http.Handler
	logger  *slog.Logger
	max     int64
}

func New(service *application.Service, auth func(http.Handler) http.Handler, logger *slog.Logger, max int64) *Handler {
	return &Handler{service: service, auth: auth, logger: logger, max: max}
}

func (h *Handler) Routes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth)
	router.With(httpserver.RequirePermission("file:download")).Get("/", h.list)
	router.With(httpserver.RequirePermission("file:upload")).Post("/", h.upload)
	router.With(httpserver.RequirePermission("file:download")).Get("/{id}", h.get)
	router.With(httpserver.RequirePermission("file:download")).Get("/{id}/download", h.download)
	router.With(httpserver.RequirePermission("file:delete")).Delete("/{id}", h.delete)
	return router
}

func (h *Handler) LegacyRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth)
	router.With(httpserver.RequirePermission("file:download")).Get("/page", h.legacyList)
	router.With(httpserver.RequirePermission("file:download")).Get("/page-by-group", h.legacyList)
	router.With(httpserver.RequirePermission("file:upload")).Post("/upload", h.legacyUpload)
	router.With(httpserver.RequirePermission("file:upload")).Post("/upload/image", h.legacyUpload)
	router.With(httpserver.RequirePermission("file:download")).Get("/download/{id}", h.download)
	router.With(httpserver.RequirePermission("file:download")).Get("/preview/{id}", h.legacyPreview)
	router.With(httpserver.RequirePermission("file:download")).Get("/{id}", h.legacyGet)
	router.With(httpserver.RequirePermission("file:delete")).Delete("/{id}", h.delete)
	return router
}

func (h *Handler) LegacyGroupRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth)
	router.With(httpserver.RequirePermission("file:download")).Get("/list", h.legacyFileGroups)
	router.With(httpserver.RequirePermission("file:download")).Get("/{id}", h.legacyFileGroup)
	router.With(httpserver.RequirePermission("file:upload")).Post("/", h.legacyCreateFileGroup)
	router.With(httpserver.RequirePermission("file:upload")).Put("/", h.legacyUpdateFileGroup)
	router.With(httpserver.RequirePermission("file:delete")).Delete("/{id}", h.legacyDeleteFileGroup)
	return router
}

func (h *Handler) legacyFileGroups(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	groups, err := h.service.FileGroups(r.Context(), actor)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	_, total, err := h.service.List(r.Context(), actor, nil, "", 1, 1)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	items := make([]map[string]any, 0, len(groups))
	for _, group := range groups {
		items = append(items, legacyFileGroupObject(group))
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"groups": items, "ungroupedCount": total,
	})
}

func (h *Handler) legacyFileGroup(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	group, err := h.service.FileGroup(r.Context(), id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyFileGroupObject(group))
}

func (h *Handler) legacyCreateFileGroup(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
		Sort int32  `json:"sort"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	group, err := h.service.CreateFileGroup(r.Context(), input.Name, input.Sort)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, legacyFileGroupObject(group))
}

func (h *Handler) legacyUpdateFileGroup(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ID   any    `json:"id"`
		Name string `json:"name"`
		Sort int32  `json:"sort"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	id, err := storageUUID(input.ID)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	group, err := h.service.UpdateFileGroup(r.Context(), id, input.Name, input.Sort)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyFileGroupObject(group))
}

func (h *Handler) legacyDeleteFileGroup(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	if err := h.service.DeleteFileGroup(r.Context(), id); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func legacyFileGroupObject(group storagedomain.FileGroup) map[string]any {
	return map[string]any{
		"id": group.ID, "name": group.Name, "sort": group.Sort,
		"fileCount": group.FileCount, "createTime": group.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func storageUUID(value any) (uuid.UUID, error) {
	if text, ok := value.(string); ok {
		return uuid.Parse(text)
	}
	if id, ok := value.(uuid.UUID); ok {
		return id, nil
	}
	return uuid.Nil, errors.New("a valid id is required")
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	groupID, _ := uuid.Parse(r.URL.Query().Get("groupId"))
	actor, _ := requestctx.ActorFrom(r.Context())
	items, total, err := h.service.List(r.Context(), actor, optionalUUID(groupID), r.URL.Query().Get("keyword"), page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, httpserver.NewPage(items, total, page, pageSize))
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	created, err := h.receiveUpload(w, r)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, map[string]any{
		"file": created, "url": h.service.URL(created),
	})
}

func (h *Handler) legacyUpload(w http.ResponseWriter, r *http.Request) {
	created, err := h.receiveUpload(w, r)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, legacyFile(created, h.service.URL(created)))
}

func (h *Handler) legacyList(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	groupID, _ := uuid.Parse(r.URL.Query().Get("groupId"))
	query := r.URL.Query().Get("originalName")
	if query == "" {
		query = r.URL.Query().Get("keyword")
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	items, total, err := h.service.List(r.Context(), actor, optionalUUID(groupID), query, page, pageSize)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, legacyFile(item, h.service.URL(item)))
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
	file, err := h.service.Get(r.Context(), actor, id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyFile(file, h.service.URL(file)))
}

func (h *Handler) legacyPreview(w http.ResponseWriter, r *http.Request) {
	h.serveFile(w, r, true)
}

func legacyFile(file storagedomain.File, url string) map[string]any {
	suffix := path.Ext(file.OriginalName)
	result := map[string]any{
		"id": file.ID, "originalName": file.OriginalName, "fileName": file.OriginalName,
		"filePath": url, "url": url, "fileSize": file.SizeBytes,
		"fileType": file.ContentType, "fileSuffix": suffix,
		"storageType": file.StorageProvider, "bucketName": "",
		"createTime": file.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if file.GroupID != nil {
		result["groupId"] = *file.GroupID
	}
	if file.OwnerUserID != nil {
		result["createBy"] = *file.OwnerUserID
	}
	return result
}

func (h *Handler) receiveUpload(w http.ResponseWriter, r *http.Request) (storagedomain.File, error) {
	r.Body = http.MaxBytesReader(w, r.Body, h.max+(1<<20))
	if err := r.ParseMultipartForm(h.max + (1 << 20)); err != nil { // #nosec G120 -- body is bounded by MaxBytesReader and h.max.
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return storagedomain.File{}, apperror.New(
				apperror.KindPayloadTooLarge,
				"upload_too_large",
				"The multipart upload exceeds the configured size limit.",
			)
		}
		return storagedomain.File{}, errors.New("invalid multipart upload")
	}
	if r.MultipartForm != nil {
		defer func() { _ = r.MultipartForm.RemoveAll() }()
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return storagedomain.File{}, errors.New("file field is required")
	}
	defer func() { _ = file.Close() }()
	actor, _ := requestctx.ActorFrom(r.Context())
	groupID := (*uuid.UUID)(nil)
	if raw := r.FormValue("groupId"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return storagedomain.File{}, errors.New("groupId must be a UUID")
		}
		groupID = &parsed
	}
	created, err := h.service.Upload(r.Context(), storagedomain.Upload{
		OwnerUserID: actor.UserID, GroupID: groupID, Filename: header.Filename,
		ContentType: header.Header.Get("Content-Type"), Reader: file,
	})
	return created, err
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	file, err := h.service.Get(r.Context(), actor, id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, file)
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	h.serveFile(w, r, false)
}

func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request, preview bool) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	file, reader, _, err := h.service.Open(r.Context(), actor, id)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	defer func() { _ = reader.Close() }()
	disposition := "attachment"
	if preview || r.URL.Query().Get("inline") == "true" {
		disposition = "inline"
	}
	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": file.OriginalName}))
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.service.Delete(r.Context(), actor, id); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func optionalUUID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}
