package integrations

import (
	"bytes"
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
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
	APIv3Key             string
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
	return PaymentSession{
		Provider: "wechat", PaymentURL: result.CodeURL,
		Payload: map[string]string{"codeUrl": result.CodeURL},
	}, nil
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
	timestampValue, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("invalid WeChat Pay timestamp")
	}
	now := p.now().Unix()
	if timestampValue < now-300 || timestampValue > now+300 {
		return errors.New("stale WeChat Pay callback timestamp")
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

func (p *WeChatPay) ParseCallback(ctx context.Context, headers map[string]string, body []byte) (PaymentCallback, error) {
	if err := p.VerifyCallback(ctx, headers, body); err != nil {
		return PaymentCallback{}, err
	}
	var envelope struct {
		ID        string `json:"id"`
		EventType string `json:"event_type"`
		Resource  struct {
			Algorithm      string `json:"algorithm"`
			Ciphertext     string `json:"ciphertext"`
			Nonce          string `json:"nonce"`
			AssociatedData string `json:"associated_data"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return PaymentCallback{}, fmt.Errorf("decode WeChat Pay callback: %w", err)
	}
	if envelope.ID == "" {
		return PaymentCallback{}, errors.New("WeChat Pay callback is missing id")
	}
	plaintext, err := decryptWeChatResource(p.config.APIv3Key, envelope.Resource.Ciphertext, envelope.Resource.Nonce, envelope.Resource.AssociatedData)
	if err != nil {
		return PaymentCallback{}, err
	}
	var resource struct {
		OutTradeNo    string `json:"out_trade_no"`
		TransactionID string `json:"transaction_id"`
		TradeState    string `json:"trade_state"`
		SuccessTime   string `json:"success_time"`
		Amount        struct {
			Total    int64  `json:"total"`
			Currency string `json:"currency"`
		} `json:"amount"`
	}
	if err := json.Unmarshal(plaintext, &resource); err != nil {
		return PaymentCallback{}, fmt.Errorf("decode WeChat Pay resource: %w", err)
	}
	status := paymentStatus(resource.TradeState)
	if status == "" {
		return PaymentCallback{}, fmt.Errorf("unsupported WeChat Pay trade state %q", resource.TradeState)
	}
	paidAt := parseOptionalPaymentTime(resource.SuccessTime)
	return PaymentCallback{
		EventID: envelope.ID, MerchantOrderID: resource.OutTradeNo,
		ProviderTransactionID: resource.TransactionID, AmountCents: resource.Amount.Total,
		Currency: strings.ToUpper(resource.Amount.Currency), Status: status, PaidAt: paidAt,
	}, nil
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
	paymentURL := endpoint + values.Encode()
	return PaymentSession{Provider: "alipay", PaymentURL: paymentURL, Payload: map[string]string{
		"url": paymentURL,
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
	if appID := values.Get("app_id"); appID != "" && appID != p.config.AppID {
		return errors.New("unexpected Alipay app_id")
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

func (p *Alipay) ParseCallback(ctx context.Context, headers map[string]string, body []byte) (PaymentCallback, error) {
	if err := p.VerifyCallback(ctx, headers, body); err != nil {
		return PaymentCallback{}, err
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return PaymentCallback{}, errors.New("invalid Alipay callback form")
	}
	amount, err := parseCurrencyCents(values.Get("total_amount"))
	if err != nil {
		return PaymentCallback{}, err
	}
	status := paymentStatus(values.Get("trade_status"))
	if status == "" {
		return PaymentCallback{}, fmt.Errorf("unsupported Alipay trade status %q", values.Get("trade_status"))
	}
	eventID := values.Get("notify_id")
	if eventID == "" {
		eventID = values.Get("trade_no")
	}
	return PaymentCallback{
		EventID: eventID, MerchantOrderID: values.Get("out_trade_no"),
		ProviderTransactionID: values.Get("trade_no"), AmountCents: amount,
		Currency: "CNY", Status: status, PaidAt: parseOptionalPaymentTime(values.Get("gmt_payment")),
	}, nil
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

func paymentStatus(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "SUCCESS", "TRADE_SUCCESS", "TRADE_FINISHED":
		return "paid"
	case "CLOSED", "REVOKED", "PAYERROR", "TRADE_CLOSED":
		return "failed"
	default:
		return ""
	}
}

func parseOptionalPaymentTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			parsed = parsed.UTC()
			return &parsed
		}
	}
	return nil
}

func parseCurrencyCents(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("payment amount is required")
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, errors.New("invalid payment amount")
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole < 0 {
		return 0, errors.New("invalid payment amount")
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if len(fraction) > 2 {
			return 0, errors.New("payment amount has more than two decimal places")
		}
		fraction += strings.Repeat("0", 2-len(fraction))
	} else {
		fraction = "00"
	}
	cents, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil || whole > (1<<63-1)/100 {
		return 0, errors.New("invalid payment amount")
	}
	return whole*100 + cents, nil
}

func decryptWeChatResource(apiV3Key, ciphertext, nonce, associatedData string) ([]byte, error) {
	key, err := decodeAPIv3Key(apiV3Key)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create WeChat Pay cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create WeChat Pay GCM: %w", err)
	}
	encrypted, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, errors.New("invalid WeChat Pay resource ciphertext")
	}
	nonceBytes, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil || len(nonceBytes) != gcm.NonceSize() {
		return nil, errors.New("invalid WeChat Pay resource nonce")
	}
	plaintext, err := gcm.Open(nil, nonceBytes, encrypted, []byte(associatedData))
	if err != nil {
		return nil, errors.New("WeChat Pay resource decryption failed")
	}
	return plaintext, nil
}

func decodeAPIv3Key(value string) ([]byte, error) {
	if len(value) == 32 {
		return []byte(value), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(decoded) != 32 {
		return nil, errors.New("WECHAT_PAY_API_V3_KEY must be 32 raw bytes or base64-encoded 32 bytes")
	}
	return decoded, nil
}

var _ PaymentProvider = (*WeChatPay)(nil)
var _ PaymentProvider = (*Alipay)(nil)
