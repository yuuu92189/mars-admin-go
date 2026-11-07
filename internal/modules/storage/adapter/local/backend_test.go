package local

import (
	"bytes"
	"context"
	"testing"
)

func TestBackendRejectsPathTraversal(t *testing.T) {
	backend, err := New(t.TempDir(), "http://localhost/files")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := backend.Put(context.Background(), "../escape", bytes.NewBufferString("x"), 1, "text/plain"); err == nil {
		t.Fatal("expected traversal key to be rejected")
	}
}

func TestBackendRoundTrip(t *testing.T) {
	backend, err := New(t.TempDir(), "http://localhost/files")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	info, err := backend.Put(ctx, "2026/11/object", bytes.NewBufferString("content"), 7, "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	if info.Size != 7 {
		t.Fatalf("unexpected size %d", info.Size)
	}
	reader, _, err := backend.Get(ctx, "2026/11/object")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	data := new(bytes.Buffer)
	_, _ = data.ReadFrom(reader)
	if data.String() != "content" {
		t.Fatalf("unexpected content %q", data.String())
	}
}
