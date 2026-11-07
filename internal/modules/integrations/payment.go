package integrations

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type WeChatPayConfig struct {
	Endpoint             string
	MerchantID           string
	AppID                string
	SerialNumber         string
	PrivateKeyPEM        string
	PlatformPublicKeyPEM string
	PlatformSerialNumber string
	HTTPClient           *http.Client
	Now                  func() time.Time
}

type WeChatPay struct {
	config     WeChatPayConfig
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	client     *http.Client
	now        func() time.Time
}

func NewWeChatPay(config WeChatPayConfig) (*WeChatPay, error) {
	if config.MerchantID == "" || config.AppID == "" || config.PrivateKeyPEM == "" {
		return nil, ErrNotConfigured
	}
	privateKey, err := parseRSAPrivateKey(config.PrivateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse WeChat Pay private key: %w", err)
	}
	var publicKey *rsa.PublicKey
	if config.PlatformPublicKeyPEM != "" {
		publicKey, err = parseRSAPublicKey(config.PlatformPublicKeyPEM)
		if err != nil {
			return nil, fmt.Errorf("parse WeChat Pay platform key: %w", err)
		}
	}
	if config.Endpoint == "" {
		config.Endpoint = "https://api.mch.weixin.qq.com"
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	if config.Now == nil {
		config.Now = func() time.Time { return time.Now().UTC() }
	}
	return &WeChatPay{config: config, privateKey: privateKey, publicKey: publicKey, client: config.HTTPClient, now: config.Now}, nil
}

func (p *WeChatPay) Create(ctx context.Context, request PaymentRequest) (PaymentSession, error) {
	if request.OrderID == "" || request.AmountCents <= 0 {
		return PaymentSession{}, errors.New("payment order ID and positive amount are required")
	}
	payload := map[string]any{
		"appid": p.config.AppID, "mchid": p.config.MerchantID,
		"description": request.Subject, "out_trade_no": request.OrderID,
		"notify_url": request.NotifyURL,
		"amount":     map[string]any{"total": request.AmountCents, "currency": "CNY"},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return PaymentSession{}, err
	}
	path := "/v3/pay/transactions/native"
	timestamp := strconv.FormatInt(p.now().Unix(), 10)
	nonceValue, err := randomHex(16)
	if err != nil {
		return PaymentSession{}, err
	}
	signature, err := signRSASHA256(p.privateKey, wechatPayMessage(http.MethodPost, path, timestamp, nonceValue, body))
	if err != nil {
		return PaymentSession{}, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.config.Endpoint, "/")+path, bytes.NewReader(body)) // #nosec G704 -- endpoint is trusted provider configuration.
	if err != nil {
		return PaymentSession{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Authorization", fmt.Sprintf(
		`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",signature="%s",timestamp="%s",serial_no="%s"`,
		p.config.MerchantID, nonceValue, signature, timestamp, p.config.SerialNumber,
	))
	response, err := p.client.Do(httpRequest) // #nosec G704 -- request uses the provider endpoint above.
	if err != nil {
		return PaymentSession{}, fmt.Errorf("create WeChat Pay order: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return PaymentSession{}, err
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return PaymentSession{}, fmt.Errorf("WeChat Pay HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	var result struct {
		CodeURL string `json:"code_url"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return PaymentSession{}, err
	}
	if result.CodeURL == "" {
		return PaymentSession{}, errors.New("WeChat Pay response did not include code_url")
	}
	return PaymentSession{Provider: "wechat", Payload: map[string]string{"codeUrl": result.CodeURL}}, nil
}

func (p *WeChatPay) VerifyCallback(_ context.Context, headers map[string]string, body []byte) error {
	if p.publicKey == nil {
		return ErrNotConfigured
	}
	timestamp := headerValue(headers, "Wechatpay-Timestamp")
	nonceValue := headerValue(headers, "Wechatpay-Nonce")
	signature := headerValue(headers, "Wechatpay-Signature")
	serial := headerValue(headers, "Wechatpay-Serial")
	if timestamp == "" || nonceValue == "" || signature == "" {
		return errors.New("missing WeChat Pay signature headers")
	}
	if p.config.PlatformSerialNumber != "" && serial != p.config.PlatformSerialNumber {
		return errors.New("unexpected WeChat Pay platform certificate serial")
	}
	decodedSignature, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return errors.New("invalid WeChat Pay signature encoding")
	}
	message := timestamp + "\n" + nonceValue + "\n" + string(body) + "\n"
	return verifyRSASHA256(p.publicKey, message, decodedSignature)
}

type AlipayConfig struct {
	Endpoint      string
	AppID         string
	PrivateKeyPEM string
	PublicKeyPEM  string
	NotifyURL     string
	ReturnURL     string
	HTTPClient    *http.Client
	Now           func() time.Time
}

type Alipay struct {
	config     AlipayConfig
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	now        func() time.Time
}

func NewAlipay(config AlipayConfig) (*Alipay, error) {
	if config.AppID == "" || config.PrivateKeyPEM == "" {
		return nil, ErrNotConfigured
	}
	privateKey, err := parseRSAPrivateKey(config.PrivateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse Alipay private key: %w", err)
	}
	var publicKey *rsa.PublicKey
	if config.PublicKeyPEM != "" {
		publicKey, err = parseRSAPublicKey(config.PublicKeyPEM)
		if err != nil {
			return nil, fmt.Errorf("parse Alipay public key: %w", err)
		}
	}
	if config.Endpoint == "" {
		config.Endpoint = "https://openapi.alipay.com/gateway.do"
	}
	if config.Now == nil {
		config.Now = func() time.Time { return time.Now().UTC() }
	}
	return &Alipay{config: config, privateKey: privateKey, publicKey: publicKey, now: config.Now}, nil
}

func (p *Alipay) Create(_ context.Context, request PaymentRequest) (PaymentSession, error) {
	if request.OrderID == "" || request.AmountCents <= 0 {
		return PaymentSession{}, errors.New("payment order ID and positive amount are required")
	}
	bizContent, err := json.Marshal(map[string]string{
		"out_trade_no": request.OrderID,
		"total_amount": fmt.Sprintf("%.2f", float64(request.AmountCents)/100),
		"subject":      request.Subject,
		"product_code": "FAST_INSTANT_TRADE_PAY",
	})
	if err != nil {
		return PaymentSession{}, err
	}
	notifyURL := request.NotifyURL
	if notifyURL == "" {
		notifyURL = p.config.NotifyURL
	}
	values := url.Values{
		"app_id": {p.config.AppID}, "method": {"alipay.trade.page.pay"},
		"format": {"JSON"}, "charset": {"utf-8"}, "sign_type": {"RSA2"},
		"timestamp": {p.now().Format("2006-01-02 15:04:05")}, "version": {"1.0"},
		"notify_url": {notifyURL}, "biz_content": {string(bizContent)},
	}
	if request.ReturnURL != "" {
		values.Set("return_url", request.ReturnURL)
	} else if p.config.ReturnURL != "" {
		values.Set("return_url", p.config.ReturnURL)
	}
	canonical := canonicalAlipayForm(values, false)
	signature, err := signRSASHA256(p.privateKey, canonical)
	if err != nil {
		return PaymentSession{}, err
	}
	values.Set("sign", signature)
	endpoint := p.config.Endpoint
	if strings.Contains(endpoint, "?") {
		endpoint += "&"
	} else {
		endpoint += "?"
	}
	return PaymentSession{Provider: "alipay", Payload: map[string]string{
		"url": endpoint + values.Encode(),
	}}, nil
}

func (p *Alipay) VerifyCallback(_ context.Context, _ map[string]string, body []byte) error {
	if p.publicKey == nil {
		return ErrNotConfigured
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return errors.New("invalid Alipay callback form")
	}
	signatureValue := values.Get("sign")
	if signatureValue == "" {
		return errors.New("missing Alipay callback signature")
	}
	if signType := values.Get("sign_type"); signType != "" && signType != "RSA2" {
		return errors.New("unsupported Alipay signature type")
	}
	signature, err := base64.StdEncoding.DecodeString(signatureValue)
	if err != nil {
		return errors.New("invalid Alipay signature encoding")
	}
	return verifyRSASHA256(p.publicKey, canonicalAlipayForm(values, true), signature)
}

func canonicalAlipayForm(values url.Values, excludeSignature bool) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key == "" || (excludeSignature && (key == "sign" || key == "sign_type")) {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		items := values[key]
		if len(items) == 0 || items[0] == "" {
			continue
		}
		parts = append(parts, key+"="+items[0])
	}
	return strings.Join(parts, "&")
}

func wechatPayMessage(method, path, timestamp, nonceValue string, body []byte) string {
	return method + "\n" + path + "\n" + timestamp + "\n" + nonceValue + "\n" + string(body) + "\n"
}

func signRSASHA256(key *rsa.PrivateKey, message string) (string, error) {
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func verifyRSASHA256(key *rsa.PublicKey, message string, signature []byte) error {
	digest := sha256.Sum256([]byte(message))
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature)
}

func parseRSAPrivateKey(value string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(value))
	if block == nil {
		return nil, errors.New("PEM block not found")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("private key is not RSA")
		}
		return rsaKey, nil
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func parseRSAPublicKey(value string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(value))
	if block == nil {
		return nil, errors.New("PEM block not found")
	}
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not RSA")
		}
		return rsaKey, nil
	}
	return x509.ParsePKCS1PublicKey(block.Bytes)
}

func randomHex(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", buffer), nil
}

func headerValue(headers map[string]string, name string) string {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}

var _ PaymentProvider = (*WeChatPay)(nil)
var _ PaymentProvider = (*Alipay)(nil)
