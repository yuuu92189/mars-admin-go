package s3

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"mars-admin-go/internal/modules/storage/domain"
)

type objectStore struct {
	mu      sync.Mutex
	objects map[string][]byte
	types   map[string]string
}

func (s *objectStore) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/mars-test/")
	if key == r.URL.Path || key == "" {
		http.NotFound(w, r)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch r.Method {
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.objects[key] = body
		s.types[key] = r.Header.Get("Content-Type")
		w.Header().Set("ETag", `"integration-etag"`)
		w.WriteHeader(http.StatusOK)
	case http.MethodHead:
		body, ok := s.objects[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Header().Set("Content-Type", s.types[key])
		w.Header().Set("ETag", `"integration-etag"`)
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		body, ok := s.objects[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Header().Set("Content-Type", s.types[key])
		w.Header().Set("ETag", `"integration-etag"`)
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		_, _ = w.Write(body)
	case http.MethodDelete:
		delete(s.objects, key)
		delete(s.types, key)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func TestBackendS3Protocol(t *testing.T) {
	store := &objectStore{objects: make(map[string][]byte), types: make(map[string]string)}
	server := httptest.NewServer(store)
	defer server.Close()

	backend, err := New(Config{
		Endpoint: server.URL, Region: "us-east-1", Bucket: "mars-test",
		PublicBase: server.URL + "/mars-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("s3 protocol payload")
	info, err := backend.Put(context.Background(), "integration/object.txt", bytes.NewReader(payload), int64(len(payload)), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	if info.Size != int64(len(payload)) {
		t.Fatalf("unexpected size %d", info.Size)
	}
	exists, err := backend.Exists(context.Background(), "integration/object.txt")
	if err != nil || !exists {
		t.Fatalf("expected object to exist, exists=%v err=%v", exists, err)
	}
	reader, objectInfo, err := backend.Get(context.Background(), "integration/object.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	if objectInfo.Size != int64(len(payload)) {
		t.Fatalf("unexpected object info size %d", objectInfo.Size)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("unexpected payload %q", data)
	}
	if err := backend.Delete(context.Background(), "integration/object.txt"); err != nil {
		t.Fatal(err)
	}
	if err := backend.Delete(context.Background(), "integration/object.txt"); err != nil {
		t.Fatalf("repeated delete must be idempotent: %v", err)
	}
	exists, err = backend.Exists(context.Background(), "integration/object.txt")
	if err != nil || exists {
		t.Fatalf("expected object deletion, exists=%v err=%v", exists, err)
	}
}

var _ domain.Backend = (*Backend)(nil)
