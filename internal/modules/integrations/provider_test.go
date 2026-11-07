package integrations

import (
	"context"
	"errors"
	"testing"
)

func TestMissingProviderIsNotConfigured(t *testing.T) {
	registry := NewRegistry()
	_, err := registry.Payment("wechat")
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func TestUnconfiguredProviderDoesNotFakeSuccess(t *testing.T) {
	provider := UnconfiguredProvider{Name: "alipay"}
	err := provider.VerifyCallback(context.Background(), nil, []byte(`{}`))
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}
