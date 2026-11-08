package realtime

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mars-admin-go/internal/platform/requestctx"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

type heartbeatAuth struct{}

func (heartbeatAuth) Authenticate(context.Context, string) (requestctx.Actor, error) {
	return requestctx.Actor{UserID: uuid.New(), Username: "heartbeat"}, nil
}

func TestWebSocketServerHeartbeatKeepsConnectionUsable(t *testing.T) {
	hub := New(
		slog.Default(), heartbeatAuth{}, nil, nil, prometheus.NewRegistry(),
		"http://localhost",
	)
	hub.heartbeat = 20 * time.Millisecond
	server := httptest.NewServer(hub)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	connection, response, err := websocket.Dial(
		ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws/message",
		&websocket.DialOptions{HTTPHeader: map[string][]string{"Authorization": {"Bearer test"}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if response != nil && response.Body != nil {
		defer func() { _ = response.Body.Close() }()
	}
	defer func() { _ = connection.CloseNow() }()
	if _, _, err := connection.Read(ctx); err != nil {
		t.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)
	if err := connection.Write(ctx, websocket.MessageText, []byte(`{"type":"ping"}`)); err != nil {
		t.Fatal(err)
	}
	_, payload, err := connection.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"type":"pong"`) {
		t.Fatalf("unexpected websocket response: %s", payload)
	}
}

func TestWebSocketRejectsQueryToken(t *testing.T) {
	hub := New(
		slog.Default(), heartbeatAuth{}, nil, nil, prometheus.NewRegistry(),
		"http://localhost",
	)
	server := httptest.NewServer(hub)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	connection, response, err := websocket.Dial(
		ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws/message?token=test", nil,
	)
	if response != nil {
		defer func() { _ = response.Body.Close() }()
	}
	if err == nil {
		_ = connection.CloseNow()
		t.Fatal("query-token WebSocket must be rejected")
	}
	if response == nil || response.StatusCode != 401 {
		t.Fatalf("status = %v, want 401", response)
	}
}
