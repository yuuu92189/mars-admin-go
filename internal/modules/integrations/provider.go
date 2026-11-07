package integrations

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"time"
)

var ErrNotConfigured = errors.New("provider is not configured")

type SMSProvider interface {
	SendCode(ctx context.Context, phone, code string) error
	SendNotice(ctx context.Context, phone, template string, params map[string]string) error
}

type SocialIdentity struct {
	Provider string
	Subject  string
	Nickname string
	Avatar   string
	Email    string
	Phone    string
	Raw      map[string]any
}

type SocialLoginProvider interface {
	Exchange(ctx context.Context, code string) (SocialIdentity, error)
}

type PaymentRequest struct {
	OrderID     string
	AmountCents int64
	Subject     string
	ReturnURL   string
	NotifyURL   string
}

type PaymentSession struct {
	Provider string
	Payload  map[string]string
}

type PaymentProvider interface {
	Create(ctx context.Context, request PaymentRequest) (PaymentSession, error)
	VerifyCallback(ctx context.Context, headers map[string]string, body []byte) error
}

type PushProvider interface {
	Send(ctx context.Context, recipient, title, content string) error
}

type MailProvider interface {
	Send(ctx context.Context, to, subject, body string) error
}

type Registry struct {
	sms     map[string]SMSProvider
	social  map[string]SocialLoginProvider
	payment map[string]PaymentProvider
	push    map[string]PushProvider
	mail    map[string]MailProvider
}

func NewRegistry() *Registry {
	return &Registry{
		sms: make(map[string]SMSProvider), social: make(map[string]SocialLoginProvider),
		payment: make(map[string]PaymentProvider), push: make(map[string]PushProvider),
		mail: make(map[string]MailProvider),
	}
}

func (r *Registry) RegisterSMS(name string, provider SMSProvider) { r.sms[name] = provider }
func (r *Registry) RegisterSocial(name string, provider SocialLoginProvider) {
	r.social[name] = provider
}
func (r *Registry) RegisterPayment(name string, provider PaymentProvider) { r.payment[name] = provider }
func (r *Registry) RegisterPush(name string, provider PushProvider)       { r.push[name] = provider }
func (r *Registry) RegisterMail(name string, provider MailProvider)       { r.mail[name] = provider }

func (r *Registry) SMS(name string) (SMSProvider, error) {
	provider := r.sms[name]
	if provider == nil {
		return nil, ErrNotConfigured
	}
	return provider, nil
}

func (r *Registry) Social(name string) (SocialLoginProvider, error) {
	provider := r.social[name]
	if provider == nil {
		return nil, ErrNotConfigured
	}
	return provider, nil
}

func (r *Registry) Payment(name string) (PaymentProvider, error) {
	provider := r.payment[name]
	if provider == nil {
		return nil, ErrNotConfigured
	}
	return provider, nil
}

func (r *Registry) Push(name string) (PushProvider, error) {
	provider := r.push[name]
	if provider == nil {
		return nil, ErrNotConfigured
	}
	return provider, nil
}

func (r *Registry) Mail(name string) (MailProvider, error) {
	provider := r.mail[name]
	if provider == nil {
		return nil, ErrNotConfigured
	}
	return provider, nil
}

func (r *Registry) Statuses() []ProviderStatus {
	now := time.Now().UTC()
	statuses := make([]ProviderStatus, 0, len(r.sms)+len(r.social)+len(r.payment)+len(r.push)+len(r.mail))
	for name := range r.sms {
		statuses = append(statuses, ProviderStatus{Name: "sms/" + name, Configured: true, CheckedAt: now})
	}
	for name := range r.social {
		statuses = append(statuses, ProviderStatus{Name: "social/" + name, Configured: true, CheckedAt: now})
	}
	for name := range r.payment {
		statuses = append(statuses, ProviderStatus{Name: "payment/" + name, Configured: true, CheckedAt: now})
	}
	for name := range r.push {
		statuses = append(statuses, ProviderStatus{Name: "push/" + name, Configured: true, CheckedAt: now})
	}
	for name := range r.mail {
		statuses = append(statuses, ProviderStatus{Name: "mail/" + name, Configured: true, CheckedAt: now})
	}
	sort.Slice(statuses, func(i, j int) bool { return statuses[i].Name < statuses[j].Name })
	return statuses
}

type ConsoleSMS struct {
	logger *slog.Logger
}

func NewConsoleSMS(logger *slog.Logger) *ConsoleSMS {
	return &ConsoleSMS{logger: logger}
}

func (c *ConsoleSMS) SendCode(_ context.Context, phone, code string) error {
	c.logger.Warn("development SMS code generated", "phone_suffix", suffix(phone), "code_length", len(code))
	return nil
}

func (c *ConsoleSMS) SendNotice(_ context.Context, phone, template string, params map[string]string) error {
	c.logger.Warn("development SMS notice", "phone_suffix", suffix(phone), "template", template, "parameter_count", len(params))
	return nil
}

type UnconfiguredProvider struct {
	Name string
}

func (p UnconfiguredProvider) Exchange(context.Context, string) (SocialIdentity, error) {
	return SocialIdentity{}, ErrNotConfigured
}

func (p UnconfiguredProvider) Create(context.Context, PaymentRequest) (PaymentSession, error) {
	return PaymentSession{}, ErrNotConfigured
}

func (p UnconfiguredProvider) VerifyCallback(context.Context, map[string]string, []byte) error {
	return ErrNotConfigured
}

func (p UnconfiguredProvider) Send(context.Context, string, string, string) error {
	return ErrNotConfigured
}

func suffix(value string) string {
	if len(value) <= 4 {
		return value
	}
	return value[len(value)-4:]
}

type ProviderStatus struct {
	Name       string    `json:"name"`
	Configured bool      `json:"configured"`
	CheckedAt  time.Time `json:"checkedAt"`
}
