package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"mars-admin-go/internal/platform/apperror"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestPageResponseUsesCanonicalPaginationShape(t *testing.T) {
	payload, err := json.Marshal(NewPage([]string{"a"}, 1, 1, 20))
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal(payload, &body))
	require.Contains(t, body, "items")
	require.Contains(t, body, "total")
	require.Contains(t, body, "page")
	require.Contains(t, body, "pageSize")
	require.NotContains(t, body, "records")
	require.NotContains(t, body, "list")
	require.NotContains(t, body, "pageNum")
	require.NotContains(t, body, "rows")
}

func TestWriteErrorMapsWrappedPostgresSQLState(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/test", nil)
	WriteError(recorder, request, slog.New(slog.NewTextHandler(io.Discard, nil)), apperror.Wrap(
		apperror.KindInternal, "persistence_failed", "Persistence failed.",
		&pgconn.PgError{Code: "23505", ConstraintName: "sensitive_constraint"},
	))

	require.Equal(t, http.StatusConflict, recorder.Code)
	var problem Problem
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&problem))
	require.Equal(t, "resource_conflict", problem.Code)
	require.NotContains(t, problem.Detail, "sensitive_")
}

func TestWriteErrorMapsPostgresSQLState(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		wantStatus int
		wantCode   string
	}{
		{name: "unique", code: "23505", wantStatus: http.StatusConflict, wantCode: "resource_conflict"},
		{name: "foreign key", code: "23503", wantStatus: http.StatusConflict, wantCode: "resource_in_use"},
		{name: "not null", code: "23502", wantStatus: http.StatusUnprocessableEntity, wantCode: "validation_failed"},
		{name: "check", code: "23514", wantStatus: http.StatusUnprocessableEntity, wantCode: "validation_failed"},
		{name: "invalid input", code: "22P02", wantStatus: http.StatusUnprocessableEntity, wantCode: "validation_failed"},
		{name: "serialization", code: "40001", wantStatus: http.StatusConflict, wantCode: "transaction_conflict"},
		{name: "deadlock", code: "40P01", wantStatus: http.StatusConflict, wantCode: "transaction_conflict"},
		{name: "cancelled", code: "57014", wantStatus: http.StatusServiceUnavailable, wantCode: "request_cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/test", nil)
			WriteError(recorder, request, slog.New(slog.NewTextHandler(io.Discard, nil)), &pgconn.PgError{
				Code: tt.code, ConstraintName: "sensitive_constraint", TableName: "sensitive_table",
			})

			require.Equal(t, tt.wantStatus, recorder.Code)
			var problem Problem
			require.NoError(t, json.NewDecoder(recorder.Body).Decode(&problem))
			require.Equal(t, tt.wantCode, problem.Code)
			require.NotContains(t, problem.Detail, "sensitive_")
			require.NotContains(t, problem.Detail, "SQL")
		})
	}
}
