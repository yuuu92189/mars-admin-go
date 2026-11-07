package integrations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWeChatMiniProgramExchange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("js_code") != "login-code" {
			t.Fatalf("unexpected code %q", r.URL.Query().Get("js_code"))
		}
		if r.URL.Query().Get("secret") != "secret" {
			t.Fatal("missing app secret")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"openid":"openid-1","unionid":"unionid-1","session_key":"session"}`))
	}))
	defer server.Close()

	provider, err := NewWeChatMiniProgram(WeChatSocialConfig{
		Endpoint: server.URL, AppID: "app", AppSecret: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	identity, err := provider.Exchange(context.Background(), "login-code")
	if err != nil {
		t.Fatal(err)
	}
	if identity.Subject != "unionid-1" || identity.Provider != "wechat_miniprogram" {
		t.Fatalf("unexpected identity %#v", identity)
	}
}
