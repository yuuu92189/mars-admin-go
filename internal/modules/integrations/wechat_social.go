package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type WeChatSocialConfig struct {
	Endpoint   string
	AppID      string
	AppSecret  string
	HTTPClient *http.Client
}

type WeChatMiniProgram struct {
	config WeChatSocialConfig
	client *http.Client
}

func NewWeChatMiniProgram(config WeChatSocialConfig) (*WeChatMiniProgram, error) {
	if config.AppID == "" || config.AppSecret == "" {
		return nil, ErrNotConfigured
	}
	if config.Endpoint == "" {
		config.Endpoint = "https://api.weixin.qq.com/sns/jscode2session"
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &WeChatMiniProgram{config: config, client: config.HTTPClient}, nil
}

func (p *WeChatMiniProgram) Exchange(ctx context.Context, code string) (SocialIdentity, error) {
	if strings.TrimSpace(code) == "" {
		return SocialIdentity{}, fmt.Errorf("WeChat login code is required")
	}
	endpoint, err := url.Parse(p.config.Endpoint)
	if err != nil {
		return SocialIdentity{}, err
	}
	query := endpoint.Query()
	query.Set("appid", p.config.AppID)
	query.Set("secret", p.config.AppSecret)
	query.Set("js_code", code)
	query.Set("grant_type", "authorization_code")
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return SocialIdentity{}, err
	}
	response, err := p.client.Do(request)
	if err != nil {
		return SocialIdentity{}, fmt.Errorf("exchange WeChat mini program code: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return SocialIdentity{}, err
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return SocialIdentity{}, fmt.Errorf("WeChat mini program HTTP %d", response.StatusCode)
	}
	var result struct {
		OpenID     string `json:"openid"`
		UnionID    string `json:"unionid"`
		SessionKey string `json:"session_key"`
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return SocialIdentity{}, fmt.Errorf("decode WeChat mini program response: %w", err)
	}
	if result.ErrCode != 0 {
		return SocialIdentity{}, fmt.Errorf("WeChat mini program rejected login: %s (%d)", result.ErrMsg, result.ErrCode)
	}
	if result.OpenID == "" {
		return SocialIdentity{}, fmt.Errorf("WeChat mini program response did not contain openid")
	}
	subject := result.OpenID
	if result.UnionID != "" {
		subject = result.UnionID
	}
	return SocialIdentity{
		Provider: "wechat_miniprogram", Subject: subject,
		Raw: map[string]any{"openid": result.OpenID, "unionid": result.UnionID},
	}, nil
}

var _ SocialLoginProvider = (*WeChatMiniProgram)(nil)
