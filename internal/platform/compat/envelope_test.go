package compat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLegacyEnvelopeWrapsSuccess(t *testing.T) {
	handler := Envelope(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":"ok"}`))
	}))
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/info", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	var response envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 {
		t.Fatalf("unexpected code %d", response.Code)
	}
}

func TestCanonicalResponseIsUnchanged(t *testing.T) {
	handler := Envelope(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":"ok"}`))
	}))
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/auth/me", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Body.String() != `{"value":"ok"}` {
		t.Fatalf("canonical response changed: %s", recorder.Body.String())
	}
}
