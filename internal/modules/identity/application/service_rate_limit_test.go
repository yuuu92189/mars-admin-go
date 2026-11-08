package application

import (
	"context"
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
