package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDownloadRejectsUnboundedTableSelection(t *testing.T) {
	ids := make([]string, 51)
	for index := range ids {
		ids[index] = uuid.NewString()
	}
	request := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/download?ids="+strings.Join(ids, ","),
		nil,
	)
	recorder := httptest.NewRecorder()

	(&Handler{}).download(recorder, request)

	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
}
