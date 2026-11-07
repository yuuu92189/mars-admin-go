package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAliyunSMSBuildsSignedRequest(t *testing.T) {
	var captured map[string][]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Code":"OK","Message":"OK"}`))
	}))
	defer server.Close()

	provider, err := NewAliyunSMS(AliyunSMSConfig{
		Endpoint: server.URL, AccessKeyID: "key", AccessKeySecret: "secret",
		SignName: "Mars", TemplateCode: "SMS_1",
		Now: func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := provider.SendCode(context.Background(), "13800138000", "123456"); err != nil {
		t.Fatal(err)
	}
	if captured["Action"][0] != "SendSms" || captured["SignName"][0] != "Mars" {
		t.Fatalf("unexpected request: %#v", captured)
	}
	if captured["Signature"][0] == "" || captured["SignatureMethod"][0] != "HMAC-SHA1" {
		t.Fatal("request was not signed")
	}
	var params map[string]string
	if err := json.Unmarshal([]byte(captured["TemplateParam"][0]), &params); err != nil {
		t.Fatal(err)
	}
	if params["code"] != "123456" {
		t.Fatalf("unexpected template params %#v", params)
	}
}
