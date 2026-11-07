package integrations

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type WebhookPushConfig struct {
	Provider   string
	URL        string
	Secret     string
	HTTPClient *http.Client
	Now        func() time.Time
}

type WebhookPush struct {
	config WebhookPushConfig
	client *http.Client
	now    func() time.Time
}

func NewWebhookPush(config WebhookPushConfig) (*WebhookPush, error) {
	if config.URL == "" {
		return nil, ErrNotConfigured
	}
	if _, err := url.ParseRequestURI(config.URL); err != nil {
		return nil, fmt.Errorf("invalid push webhook URL: %w", err)
	}
	if config.Provider == "" {
		config.Provider = "webhook"
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if config.Now == nil {
		config.Now = func() time.Time { return time.Now().UTC() }
	}
	return &WebhookPush{config: config, client: config.HTTPClient, now: config.Now}, nil
}

func (p *WebhookPush) Send(ctx context.Context, _, title, content string) error {
	message := title
	if content != "" {
		message += "\n" + content
	}
	var payload any
	switch p.config.Provider {
	case "feishu":
		payload = map[string]any{"msg_type": "text", "content": map[string]string{"text": message}}
	default:
		payload = map[string]any{"msgtype": "text", "text": map[string]string{"content": message}}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	endpoint, err := url.Parse(p.config.URL)
	if err != nil {
		return err
	}
	if p.config.Provider == "dingtalk" && p.config.Secret != "" {
		timestamp := strconv.FormatInt(p.now().UnixMilli(), 10)
		mac := hmac.New(sha256.New, []byte(p.config.Secret))
		_, _ = mac.Write([]byte(timestamp + "\n" + p.config.Secret))
		query := endpoint.Query()
		query.Set("timestamp", timestamp)
		query.Set("sign", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
		endpoint.RawQuery = query.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("send push webhook: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 256*1024))
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("push webhook HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	text := string(responseBody)
	if strings.Contains(text, `"errcode":`) && !strings.Contains(text, `"errcode":0`) {
		return fmt.Errorf("push webhook rejected message: %s", strings.TrimSpace(text))
	}
	return nil
}

var _ PushProvider = (*WebhookPush)(nil)
