package bootstrap

import (
	"context"
	"errors"
	"testing"
)

type fakePublicSettings struct {
	values map[string]any
	err    error
}

func (f fakePublicSettings) Public(context.Context) (map[string]any, error) {
	return f.values, f.err
}

func TestCaptchaSettingsPolicyUsesDatabaseSetting(t *testing.T) {
	policy := publicSettingPolicy{
		settings: fakePublicSettings{
			values: map[string]any{"security.auth.captcha_enabled": true},
		},
		fallback: false,
	}
	enabled, err := policy.Enabled(context.Background())
	if err != nil {
		t.Fatalf("captcha policy: %v", err)
	}
	if !enabled {
		t.Fatal("expected database captcha setting to take precedence")
	}
}

func TestCaptchaSettingsPolicyFallsBackAndPropagatesErrors(t *testing.T) {
	policy := publicSettingPolicy{
		settings: fakePublicSettings{values: map[string]any{}},
		fallback: true,
	}
	enabled, err := policy.Enabled(context.Background())
	if err != nil {
		t.Fatalf("captcha policy fallback: %v", err)
	}
	if !enabled {
		t.Fatal("expected configured fallback to be used")
	}

	expected := errors.New("settings unavailable")
	policy.settings = fakePublicSettings{err: expected}
	if _, err := policy.Enabled(context.Background()); !errors.Is(err, expected) {
		t.Fatalf("expected settings error, got %v", err)
	}
}

func TestRegistrationPolicyReadsRegistrationSetting(t *testing.T) {
	policy := publicSettingPolicy{
		settings: fakePublicSettings{
			values: map[string]any{"site.site.registration_enabled": true},
		},
		key: "site.site.registration_enabled",
	}
	enabled, err := policy.Enabled(context.Background())
	if err != nil {
		t.Fatalf("registration policy: %v", err)
	}
	if !enabled {
		t.Fatal("expected registration setting to be read from public settings")
	}
}
