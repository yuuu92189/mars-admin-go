package config

import (
	"testing"
	"time"
)

func TestValidateRejectsNonPostgresURL(t *testing.T) {
	cfg := Config{
		App:      AppConfig{Env: "development", Port: 8080},
		Database: DatabaseConfig{URL: "mysql://localhost/db", MaxConnections: 2},
		Auth:     AuthConfig{SessionTTL: time.Minute, RefreshTTL: time.Hour, CookieSameSite: "lax"},
		Storage:  StorageConfig{Provider: "local", MaxUploadBytes: 1},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected non-PostgreSQL URL to be rejected")
	}
}

func TestValidateRejectsInsecureSameSiteNone(t *testing.T) {
	cfg := Config{
		App:      AppConfig{Env: "development", Port: 8080},
		Database: DatabaseConfig{URL: "postgres://localhost/db", MaxConnections: 2},
		Auth: AuthConfig{
			SessionTTL:     time.Minute,
			RefreshTTL:     time.Hour,
			CookieSameSite: "none",
			CookieSecure:   false,
		},
		Storage: StorageConfig{Provider: "local", MaxUploadBytes: 1},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected SameSite=None without Secure to fail")
	}
}
