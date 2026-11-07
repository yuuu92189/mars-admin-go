package integrations

import (
	"context"
	"crypto/hmac"
	"crypto/sha1" // #nosec G505 -- SHA-1 is required by the Aliyun SMS RPC signature protocol.
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type AliyunSMSConfig struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	NoticeTemplate  string
	RegionID        string
	HTTPClient      *http.Client
	Now             func() time.Time
}

type AliyunSMS struct {
	config AliyunSMSConfig
	client *http.Client
	now    func() time.Time
}

func NewAliyunSMS(config AliyunSMSConfig) (*AliyunSMS, error) {
	if config.AccessKeyID == "" || config.AccessKeySecret == "" {
		return nil, ErrNotConfigured
	}
	if config.Endpoint == "" {
		config.Endpoint = "https://dysmsapi.aliyuncs.com/"
	}
	if config.RegionID == "" {
		config.RegionID = "cn-hangzhou"
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if config.Now == nil {
		config.Now = func() time.Time { return time.Now().UTC() }
	}
	return &AliyunSMS{config: config, client: config.HTTPClient, now: config.Now}, nil
}

func (p *AliyunSMS) SendCode(ctx context.Context, phone, code string) error {
	if p.config.TemplateCode == "" || p.config.SignName == "" {
		return ErrNotConfigured
	}
	return p.send(ctx, phone, p.config.TemplateCode, map[string]string{"code": code})
}

func (p *AliyunSMS) SendNotice(ctx context.Context, phone, _ string, params map[string]string) error {
	if p.config.NoticeTemplate == "" || p.config.SignName == "" {
		return ErrNotConfigured
	}
	return p.send(ctx, phone, p.config.NoticeTemplate, params)
}

func (p *AliyunSMS) send(ctx context.Context, phone, template string, params map[string]string) error {
	templateJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}
	timestamp := p.now().UTC().Format("2006-01-02T15:04:05Z")
	values := url.Values{
		"AccessKeyId":      {p.config.AccessKeyID},
		"Action":           {"SendSms"},
		"Format":           {"JSON"},
		"PhoneNumbers":     {phone},
		"RegionId":         {p.config.RegionID},
		"SignName":         {p.config.SignName},
		"SignatureMethod":  {"HMAC-SHA1"},
		"SignatureNonce":   {nonce(timestamp, phone)},
		"SignatureVersion": {"1.0"},
		"TemplateCode":     {template},
		"TemplateParam":    {string(templateJSON)},
		"Timestamp":        {timestamp},
		"Version":          {"2017-05-25"},
	}
	canonical := canonicalAliyunQuery(values)
	stringToSign := "GET&%2F&" + aliyunPercentEncode(canonical)
	mac := hmac.New(sha1.New, []byte(p.config.AccessKeySecret+"&"))
	_, _ = mac.Write([]byte(stringToSign))
	values.Set("Signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))

	endpoint, err := url.Parse(p.config.Endpoint)
	if err != nil {
		return err
	}
	endpoint.RawQuery = canonicalAliyunQuery(values)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}
	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("send Aliyun SMS: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("aliyun SMS HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	var result struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode Aliyun SMS response: %w", err)
	}
	if result.Code != "" && result.Code != "OK" {
		return fmt.Errorf("aliyun SMS rejected request: %s", result.Message)
	}
	return nil
}

func canonicalAliyunQuery(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		items := append([]string(nil), values[key]...)
		sort.Strings(items)
		for _, value := range items {
			parts = append(parts, aliyunPercentEncode(key)+"="+aliyunPercentEncode(value))
		}
	}
	return strings.Join(parts, "&")
}

func aliyunPercentEncode(value string) string {
	encoded := url.QueryEscape(value)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

func nonce(timestamp, phone string) string {
	seed := timestamp + phone
	sum := sha1.Sum([]byte(seed)) // #nosec G401 -- nonce generation, not password or signature security.
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

var _ SMSProvider = (*AliyunSMS)(nil)
