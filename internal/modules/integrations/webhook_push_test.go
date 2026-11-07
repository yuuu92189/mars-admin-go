package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDingtalkWebhookSignsAndSends(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("timestamp") == "" || r.URL.Query().Get("sign") == "" {
			t.Fatal("DingTalk request was not signed")
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["msgtype"] != "text" {
			t.Fatalf("unexpected payload %#v", payload)
		}
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	provider, err := NewWebhookPush(WebhookPushConfig{
		Provider: "dingtalk", URL: server.URL, Secret: "secret",
		Now: func() time.Time { return time.UnixMilli(1_700_000_000_000) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := provider.Send(context.Background(), "", "Title", "Body"); err != nil {
		t.Fatal(err)
	}
}
