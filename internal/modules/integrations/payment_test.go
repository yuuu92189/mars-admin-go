package integrations

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func TestWeChatPayCreateAndCallbackVerification(t *testing.T) {
	merchantKey := testRSAKey(t)
	platformKey := testRSAKey(t)
	now := time.Unix(1_700_000_000, 0).UTC()
	apiV3Key := "0123456789abcdef0123456789abcdef"
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
		APIv3Key: apiV3Key, Now: func() time.Time { return now },
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

	timestamp := strconv.FormatInt(now.Unix(), 10)
	nonce := "nonce"
	associatedData := "transaction"
	resource := `{"out_trade_no":"ORDER-1","transaction_id":"wx-tx-1","trade_state":"SUCCESS","success_time":"2023-11-14T22:13:20Z","amount":{"total":100,"currency":"CNY"}}`
	ciphertext, resourceNonce := encryptWeChatResource(t, apiV3Key, resource, associatedData)
	body, err := json.Marshal(map[string]any{
		"id": "evt-1", "event_type": "TRANSACTION.SUCCESS",
		"resource": map[string]string{
			"algorithm": "AEAD_AES_256_GCM", "ciphertext": ciphertext,
			"nonce": resourceNonce, "associated_data": associatedData,
		},
	})
	requireNoError(t, err)
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
	callback, err := provider.ParseCallback(context.Background(), map[string]string{
		"Wechatpay-Timestamp": timestamp,
		"Wechatpay-Nonce":     nonce,
		"Wechatpay-Signature": signature,
	}, body)
	if err != nil {
		t.Fatal(err)
	}
	if callback.EventID != "evt-1" || callback.MerchantOrderID != "ORDER-1" ||
		callback.ProviderTransactionID != "wx-tx-1" || callback.AmountCents != 100 ||
		callback.Status != "paid" {
		t.Fatalf("unexpected WeChat Pay callback %#v", callback)
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
		"sign_type": {"RSA2"}, "total_amount": {"12.34"}, "trade_no": {"ali-tx-1"},
		"notify_id": {"notify-1"}, "gmt_payment": {"2023-11-14 22:13:20"},
	}
	signature, err := signRSASHA256(platformKey, canonicalAlipayForm(callback, true))
	if err != nil {
		t.Fatal(err)
	}
	callback.Set("sign", signature)
	if err := provider.VerifyCallback(context.Background(), nil, []byte(callback.Encode())); err != nil {
		t.Fatal(err)
	}
	parsedCallback, err := provider.ParseCallback(context.Background(), nil, []byte(callback.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	if parsedCallback.EventID != "notify-1" || parsedCallback.MerchantOrderID != "ORDER-2" ||
		parsedCallback.ProviderTransactionID != "ali-tx-1" || parsedCallback.AmountCents != 1234 ||
		parsedCallback.Status != "paid" {
		t.Fatalf("unexpected Alipay callback %#v", parsedCallback)
	}
	callback.Set("total_amount", "99.99")
	if err := provider.VerifyCallback(context.Background(), nil, []byte(callback.Encode())); err == nil {
		t.Fatal("expected altered Alipay callback to fail verification")
	}
}

func encryptWeChatResource(t *testing.T, key, plaintext, associatedData string) (string, string) {
	t.Helper()
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), []byte(associatedData))
	return base64.StdEncoding.EncodeToString(ciphertext), base64.StdEncoding.EncodeToString(nonce)
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
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
