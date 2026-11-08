package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"mars-admin-go/internal/platform/apperror"
	"mars-admin-go/internal/platform/requestctx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const defaultMaxJSONBody = 1 << 20

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Problem struct {
	Type       string            `json:"type"`
	Title      string            `json:"title"`
	Status     int               `json:"status"`
	Detail     string            `json:"detail"`
	Instance   string            `json:"instance,omitempty"`
	Code       string            `json:"code,omitempty"`
	RequestID  string            `json:"requestId,omitempty"`
	TraceID    string            `json:"traceId,omitempty"`
	Violations []ValidationError `json:"violations,omitempty"`
}

func (p Problem) Error() string {
	return p.Detail
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if status == http.StatusNoContent || value == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("encode HTTP response", "error", err)
	}
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, defaultMaxJSONBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		switch {
		case errors.As(err, &maxBytesErr):
			WriteProblem(w, r, NewProblem(http.StatusRequestEntityTooLarge, "request_body_too_large", "Request body is too large."))
		case errors.Is(err, io.EOF):
			WriteProblem(w, r, NewProblem(http.StatusBadRequest, "empty_body", "Request body is required."))
		default:
			WriteProblem(w, r, NewProblem(http.StatusBadRequest, "invalid_json", "Request body is not valid JSON or contains unknown fields."))
		}
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		WriteProblem(w, r, NewProblem(http.StatusBadRequest, "multiple_json_values", "Request body must contain exactly one JSON value."))
		return false
	}
	return true
}

func NewProblem(status int, code, detail string) Problem {
	title := http.StatusText(status)
	if title == "" {
		title = "Request failed"
	}
	return Problem{
		Type:   "about:blank",
		Title:  title,
		Status: status,
		Detail: detail,
		Code:   code,
	}
}

func WriteProblem(w http.ResponseWriter, r *http.Request, problem Problem) {
	if problem.Status == 0 {
		problem.Status = http.StatusInternalServerError
	}
	if problem.Title == "" {
		problem.Title = http.StatusText(problem.Status)
	}
	if problem.Type == "" {
		problem.Type = "about:blank"
	}
	if r != nil {
		problem.Instance = r.URL.Path
		problem.RequestID = requestctx.RequestID(r.Context())
		problem.TraceID = requestctx.TraceID(r.Context())
	}
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(problem.Status)
	_ = json.NewEncoder(w).Encode(problem)
}

func WriteError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	if err == nil {
		return
	}
	var problem Problem
	if errors.As(err, &problem) {
		WriteProblem(w, r, problem)
		return
	}
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		status := apperror.HTTPStatus(err)
		if status >= 500 {
			if problem, ok := databaseProblem(err); ok {
				logger.WarnContext(r.Context(), "database application error", "sqlstate", sqlState(err), "path", r.URL.Path)
				WriteProblem(w, r, problem)
				return
			}
		}
		problem = NewProblem(status, appErr.Code, appErr.Message)
		if appErr.Field != "" {
			problem.Violations = []ValidationError{{Field: appErr.Field, Message: appErr.Message}}
		}
		if status >= 500 {
			logger.ErrorContext(r.Context(), "application error", "error", err, "path", r.URL.Path)
		}
		WriteProblem(w, r, problem)
		return
	}
	if problem, ok := databaseProblem(err); ok {
		logger.WarnContext(r.Context(), "database request failed", "sqlstate", sqlState(err), "path", r.URL.Path)
		WriteProblem(w, r, problem)
		return
	}
	logger.ErrorContext(r.Context(), "request failed", "error", err, "path", r.URL.Path)
	WriteProblem(w, r, NewProblem(http.StatusInternalServerError, "internal_error", "An internal error occurred."))
}

type PageResponse[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

func NewPage[T any](items []T, total int64, page, pageSize int) PageResponse[T] {
	return PageResponse[T]{Items: items, Total: total, Page: page, PageSize: pageSize}
}

func databaseProblem(err error) (Problem, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return Problem{}, false
	}
	switch pgErr.Code {
	case "23505":
		return NewProblem(http.StatusConflict, "resource_conflict", "The resource conflicts with an existing record."), true
	case "23503":
		return NewProblem(http.StatusConflict, "resource_in_use", "The resource is referenced by another record."), true
	case "23502", "23514", "22P02":
		return NewProblem(http.StatusUnprocessableEntity, "validation_failed", "The request contains a value that is not valid for this resource."), true
	case "40001", "40P01":
		return NewProblem(http.StatusConflict, "transaction_conflict", "The request could not be completed because of a concurrent change. Retry the request."), true
	case "57014":
		return NewProblem(http.StatusServiceUnavailable, "request_cancelled", "The database operation was cancelled."), true
	default:
		return Problem{}, false
	}
}

func sqlState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func ParsePagination(r *http.Request, maxPageSize int) (int, int, error) {
	page, err := parsePositiveInt(r.URL.Query().Get("page"), 1)
	if err != nil {
		return 0, 0, NewProblem(http.StatusBadRequest, "invalid_page", "page must be a positive integer.")
	}
	pageSize, err := parsePositiveInt(r.URL.Query().Get("pageSize"), 20)
	if err != nil {
		return 0, 0, NewProblem(http.StatusBadRequest, "invalid_page_size", "pageSize must be a positive integer.")
	}
	if maxPageSize <= 0 {
		maxPageSize = 100
	}
	if pageSize > maxPageSize {
		return 0, 0, NewProblem(http.StatusBadRequest, "invalid_page_size", fmt.Sprintf("pageSize must not exceed %d.", maxPageSize))
	}
	return page, pageSize, nil
}

func parsePositiveInt(raw string, fallback int) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("invalid positive integer")
	}
	return parsed, nil
}

func PathUUID(r *http.Request, key string) (uuid.UUID, error) {
	raw := ""
	if r != nil {
		raw = r.PathValue(key)
		if raw == "" {
			raw = chi.URLParam(r, key)
		}
	}
	if raw == "" {
		return uuid.Nil, NewProblem(http.StatusBadRequest, "missing_identifier", "A valid identifier is required.")
	}
	value, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, NewProblem(http.StatusBadRequest, "invalid_identifier", "The identifier is not a valid UUID.")
	}
	return value, nil
}
