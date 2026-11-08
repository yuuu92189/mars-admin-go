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

func TestValidateRejectsInvalidTrustedProxyCIDR(t *testing.T) {
	cfg := Config{
		App:      AppConfig{Env: "development", Port: 8080},
		Database: DatabaseConfig{URL: "postgres://localhost/db", MaxConnections: 2},
		Auth: AuthConfig{
			SessionTTL: time.Minute, RefreshTTL: time.Hour, CookieSameSite: "lax",
			TrustedProxyCIDRs: []string{"10.0.0.1"},
		},
		Storage: StorageConfig{Provider: "local", MaxUploadBytes: 1},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected bare trusted proxy address to be rejected")
	}
}

func TestValidateRequiresSecureCookiesInProduction(t *testing.T) {
	cfg := Config{
		App:      AppConfig{Env: "production", Port: 8080},
		Database: DatabaseConfig{URL: "postgres://localhost/db", MaxConnections: 2},
		Auth: AuthConfig{
			SessionTTL: time.Minute, RefreshTTL: time.Hour,
			CookieSameSite: "lax", CookieSecure: false,
		},
		Storage: StorageConfig{Provider: "local", MaxUploadBytes: 1},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected insecure production cookies to fail")
	}
}

func TestValidateRejectsUnsafeRefreshGrace(t *testing.T) {
	cfg := Config{
		App:      AppConfig{Env: "development", Port: 8080},
		Database: DatabaseConfig{URL: "postgres://localhost/db", MaxConnections: 2},
		Auth: AuthConfig{
			SessionTTL: time.Minute, RefreshTTL: time.Hour, CookieSameSite: "lax",
			RefreshGracePeriod: 3 * time.Minute, RefreshHistoryRetention: time.Hour,
		},
		Storage: StorageConfig{Provider: "local", MaxUploadBytes: 1},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected excessive refresh grace period to fail")
	}
}

func TestValidateRejectsMissingRefreshHistoryRetention(t *testing.T) {
	cfg := Config{
		App:      AppConfig{Env: "development", Port: 8080},
		Database: DatabaseConfig{URL: "postgres://localhost/db", MaxConnections: 2},
		Auth: AuthConfig{
			SessionTTL: time.Minute, RefreshTTL: time.Hour, CookieSameSite: "lax",
			RefreshGracePeriod: 15 * time.Second,
		},
		Storage: StorageConfig{Provider: "local", MaxUploadBytes: 1},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing refresh history retention to fail")
	}
}

func TestLoadRejectsInvalidEnvironmentValue(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("APP_PORT", "not-a-port")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid APP_PORT to fail configuration loading")
	}
}

func TestValidateRejectsInvalidStorageProvider(t *testing.T) {
	cfg := Config{
		App:      AppConfig{Env: "development", Port: 8080, ShutdownTimeout: time.Second, LogLevel: "info"},
		Database: DatabaseConfig{URL: "postgres://localhost/db", MaxConnections: 2, ConnectionTimeout: time.Second},
		Redis:    RedisConfig{PoolSize: 1},
		Auth: AuthConfig{
			SessionTTL: time.Minute, RefreshTTL: time.Hour, RefreshHistoryRetention: time.Hour,
			CookieSameSite: "lax", LoginMaxAttempts: 1, LoginLockDuration: time.Minute,
		},
		Storage: StorageConfig{Provider: "unknown", MaxUploadBytes: 1},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected unsupported storage provider to be rejected")
	}
}
