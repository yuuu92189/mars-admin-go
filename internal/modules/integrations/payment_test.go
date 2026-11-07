package integrations

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestWeChatPayCreateAndCallbackVerification(t *testing.T) {
	merchantKey := testRSAKey(t)
	platformKey := testRSAKey(t)
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["out_trade_no"] != "ORDER-1" {
			t.Fatalf("unexpected order body %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code_url":"weixin://wxpay/test"}`))
	}))
	defer server.Close()

	provider, err := NewWeChatPay(WeChatPayConfig{
		Endpoint: server.URL, MerchantID: "merchant", AppID: "app", SerialNumber: "serial",
		PrivateKeyPEM: privateKeyPEM(merchantKey), PlatformPublicKeyPEM: publicKeyPEM(&platformKey.PublicKey),
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := provider.Create(context.Background(), PaymentRequest{
		OrderID: "ORDER-1", AmountCents: 100, Subject: "Test", NotifyURL: "https://example.test/notify",
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.Payload["codeUrl"] != "weixin://wxpay/test" || authorization == "" {
		t.Fatalf("unexpected payment session %#v or authorization %q", session, authorization)
	}

	timestamp := "1700000000"
	nonce := "nonce"
	body := []byte(`{"id":"evt-1"}`)
	signature, err := signRSASHA256(platformKey, timestamp+"\n"+nonce+"\n"+string(body)+"\n")
	if err != nil {
		t.Fatal(err)
	}
	err = provider.VerifyCallback(context.Background(), map[string]string{
		"Wechatpay-Timestamp": timestamp,
		"Wechatpay-Nonce":     nonce,
		"Wechatpay-Signature": signature,
	}, body)
	if err != nil {
		t.Fatal(err)
	}
	body[len(body)-2] = 'X'
	if err := provider.VerifyCallback(context.Background(), map[string]string{
		"Wechatpay-Timestamp": timestamp,
		"Wechatpay-Nonce":     nonce,
		"Wechatpay-Signature": signature,
	}, body); err == nil {
		t.Fatal("expected altered callback to fail verification")
	}
}

func TestAlipayCreateAndCallbackVerification(t *testing.T) {
	merchantKey := testRSAKey(t)
	platformKey := testRSAKey(t)
	provider, err := NewAlipay(AlipayConfig{
		Endpoint: "https://openapi.alipay.test/gateway.do", AppID: "app",
		PrivateKeyPEM: privateKeyPEM(merchantKey), PublicKeyPEM: publicKeyPEM(&platformKey.PublicKey),
		NotifyURL: "https://example.test/notify",
		Now:       func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := provider.Create(context.Background(), PaymentRequest{
		OrderID: "ORDER-2", AmountCents: 1234, Subject: "Alipay test",
	})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(session.Payload["url"])
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("sign") == "" || parsed.Query().Get("app_id") != "app" {
		t.Fatalf("unexpected Alipay URL %s", parsed.String())
	}

	callback := url.Values{
		"app_id": {"app"}, "out_trade_no": {"ORDER-2"}, "trade_status": {"TRADE_SUCCESS"},
		"sign_type": {"RSA2"}, "total_amount": {"12.34"},
	}
	signature, err := signRSASHA256(platformKey, canonicalAlipayForm(callback, true))
	if err != nil {
		t.Fatal(err)
	}
	callback.Set("sign", signature)
	if err := provider.VerifyCallback(context.Background(), nil, []byte(callback.Encode())); err != nil {
		t.Fatal(err)
	}
	callback.Set("total_amount", "99.99")
	if err := provider.VerifyCallback(context.Background(), nil, []byte(callback.Encode())); err == nil {
		t.Fatal("expected altered Alipay callback to fail verification")
	}
}

func testRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func privateKeyPEM(key *rsa.PrivateKey) string {
	encoded, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		panic(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded}))
}

func publicKeyPEM(key *rsa.PublicKey) string {
	encoded, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		panic(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encoded}))
}
