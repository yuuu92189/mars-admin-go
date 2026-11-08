package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"mars-admin-go/internal/modules/identity/domain"
)

type recordedPolicy struct {
	key    string
	limit  int64
	window time.Duration
}

type recordingRateLimiter struct {
	policies []recordedPolicy
}

type staticCaptchaPolicy struct {
	enabled bool
	err     error
}

func (p staticCaptchaPolicy) Enabled(context.Context) (bool, error) {
	return p.enabled, p.err
}

func (r *recordingRateLimiter) Allow(context.Context, string) (bool, error) {
	return true, nil
}

func (r *recordingRateLimiter) AllowPolicy(
	_ context.Context,
	key string,
	limit int64,
	window time.Duration,
) (bool, error) {
	r.policies = append(r.policies, recordedPolicy{key: key, limit: limit, window: window})
	return true, nil
}

func TestEnforceLoginSourceRateLimitsUsesConfiguredLimit(t *testing.T) {
	limiter := &recordingRateLimiter{}
	service := &Service{rateLimiter: limiter, loginRateLimit: 100}

	if err := service.enforceLoginSourceRateLimits(context.Background(), domain.LoginInput{
		IPAddress:  "127.0.0.1",
		DeviceName: "e2e",
		UserAgent:  "playwright",
	}); err != nil {
		t.Fatalf("enforce login source rate limits: %v", err)
	}

	if len(limiter.policies) != 2 {
		t.Fatalf("expected 2 login source policies, got %d", len(limiter.policies))
	}
	for _, policy := range limiter.policies {
		if policy.limit != 100 {
			t.Fatalf("expected configured limit 100 for %s, got %d", policy.key, policy.limit)
		}
		if policy.window != 5*time.Minute {
			t.Fatalf("expected 5 minute window for %s, got %s", policy.key, policy.window)
		}
	}
}

func TestCaptchaPolicyOverridesStaticConfiguration(t *testing.T) {
	service := &Service{
		captchaEnabled: false,
		captchaPolicy:  staticCaptchaPolicy{enabled: true},
	}
	enabled, err := service.captchaRequired(context.Background())
	if err != nil {
		t.Fatalf("captcha policy: %v", err)
	}
	if !enabled {
		t.Fatal("expected dynamic captcha policy to enable captcha")
	}

	expected := errors.New("settings unavailable")
	service.captchaPolicy = staticCaptchaPolicy{err: expected}
	if _, err := service.captchaRequired(context.Background()); !errors.Is(err, expected) {
		t.Fatalf("expected policy error, got %v", err)
	}
}

func TestRegistrationPolicyFailsClosed(t *testing.T) {
	service := &Service{
		registrationPolicy: staticCaptchaPolicy{enabled: false},
	}
	if _, err := service.Register(context.Background(), domain.RegisterInput{
		Username: "registration-test",
		Password: "identity-test-password-123",
	}); err == nil {
		t.Fatal("expected disabled registration policy to reject registration")
	}
}
