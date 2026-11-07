package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App          AppConfig
	Database     DatabaseConfig
	Redis        RedisConfig
	Auth         AuthConfig
	Storage      StorageConfig
	Integrations IntegrationConfig
	Telemetry    TelemetryConfig
	Bootstrap    BootstrapConfig
}

type AppConfig struct {
	Env             string
	Name            string
	Version         string
	Port            int
	PublicBaseURL   string
	LogLevel        string
	ShutdownTimeout time.Duration
	CORSOrigins     []string
	AutoMigrate     bool
}

type DatabaseConfig struct {
	URL               string
	MaxConnections    int32
	MinConnections    int32
	ConnectionTimeout time.Duration
}

type RedisConfig struct {
	URL      string
	Password string
	PoolSize int
}

type AuthConfig struct {
	SessionTTL          time.Duration
	RefreshTTL          time.Duration
	CookieSecure        bool
	CookieSameSite      string
	LoginMaxAttempts    int32
	LoginLockDuration   time.Duration
	CaptchaEnabled      bool
	SensitiveDataKey    string
	TrustedProxyHeaders bool
}

type StorageConfig struct {
	Provider       string
	LocalDir       string
	PublicBaseURL  string
	MaxUploadBytes int64
	S3             S3Config
}

type S3Config struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
}

type IntegrationConfig struct {
	SMSProvider string
	AliyunSMS   AliyunSMSConfig
	WeChatMini  WeChatMiniConfig
	PushWebhook PushWebhookConfig
	WeChatPay   WeChatPayConfig
	Alipay      AlipayConfig
	SMTP        SMTPConfig
}

type AliyunSMSConfig struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	NoticeTemplate  string
}

type WeChatMiniConfig struct {
	AppID     string
	AppSecret string
	Endpoint  string
}

type PushWebhookConfig struct {
	Provider string
	URL      string
	Secret   string
}

type WeChatPayConfig struct {
	Endpoint             string
	MerchantID           string
	AppID                string
	SerialNumber         string
	PrivateKeyPEM        string
	PlatformPublicKeyPEM string
	PlatformSerialNumber string
}

type AlipayConfig struct {
	Endpoint      string
	AppID         string
	PrivateKeyPEM string
	PublicKeyPEM  string
	NotifyURL     string
	ReturnURL     string
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      bool
}

type TelemetryConfig struct {
	OTLPEndpoint string
	ServiceName  string
}

type BootstrapConfig struct {
	AdminUsername string
	AdminPassword string
	AdminEmail    string
}

func Load() (Config, error) {
	var cfg Config
	cfg.App = AppConfig{
		Env:             env("APP_ENV", "development"),
		Name:            env("APP_NAME", "Mars Admin Go Edition"),
		Version:         env("APP_VERSION", "0.1.0"),
		Port:            envInt("APP_PORT", 8080),
		PublicBaseURL:   env("APP_PUBLIC_BASE_URL", "http://localhost:8080"),
		LogLevel:        env("APP_LOG_LEVEL", "info"),
		ShutdownTimeout: envDuration("APP_SHUTDOWN_TIMEOUT", 20*time.Second),
		CORSOrigins:     splitCSV(env("APP_CORS_ORIGINS", "http://localhost:3000,http://localhost:3001")),
		AutoMigrate:     envBool("AUTO_MIGRATE", true),
	}
	cfg.Database = DatabaseConfig{
		URL:               os.Getenv("DATABASE_URL"),
		MaxConnections:    envInt32("DATABASE_MAX_CONNECTIONS", 20),
		MinConnections:    envInt32("DATABASE_MIN_CONNECTIONS", 2),
		ConnectionTimeout: envDuration("DATABASE_CONNECTION_TIMEOUT", 10*time.Second),
	}
	cfg.Redis = RedisConfig{
		URL:      env("REDIS_URL", "redis://localhost:6379/0"),
		Password: os.Getenv("REDIS_PASSWORD"),
		PoolSize: envInt("REDIS_POOL_SIZE", 10),
	}
	cfg.Auth = AuthConfig{
		SessionTTL:          envDuration("AUTH_SESSION_TTL", 30*time.Minute),
		RefreshTTL:          envDuration("AUTH_REFRESH_TTL", 30*24*time.Hour),
		CookieSecure:        envBool("AUTH_COOKIE_SECURE", false),
		CookieSameSite:      strings.ToLower(env("AUTH_COOKIE_SAME_SITE", "lax")),
		LoginMaxAttempts:    envInt32("AUTH_LOGIN_MAX_ATTEMPTS", 5),
		LoginLockDuration:   envDuration("AUTH_LOGIN_LOCK_DURATION", 30*time.Minute),
		CaptchaEnabled:      envBool("AUTH_CAPTCHA_ENABLED", false),
		SensitiveDataKey:    os.Getenv("SETTINGS_ENCRYPTION_KEY"),
		TrustedProxyHeaders: envBool("AUTH_TRUST_PROXY_HEADERS", false),
	}
	cfg.Storage = StorageConfig{
		Provider:       strings.ToLower(env("STORAGE_PROVIDER", "local")),
		LocalDir:       env("STORAGE_LOCAL_DIR", "./uploads"),
		PublicBaseURL:  strings.TrimRight(env("STORAGE_PUBLIC_BASE_URL", "http://localhost:8080/files"), "/"),
		MaxUploadBytes: int64(envInt("STORAGE_MAX_UPLOAD_BYTES", 50<<20)),
		S3: S3Config{
			Endpoint:  os.Getenv("STORAGE_S3_ENDPOINT"),
			Region:    os.Getenv("STORAGE_S3_REGION"),
			Bucket:    os.Getenv("STORAGE_S3_BUCKET"),
			AccessKey: os.Getenv("STORAGE_S3_ACCESS_KEY"),
			SecretKey: os.Getenv("STORAGE_S3_SECRET_KEY"),
		},
	}
	cfg.Integrations = IntegrationConfig{
		SMSProvider: strings.ToLower(env("SMS_PROVIDER", "console")),
		AliyunSMS: AliyunSMSConfig{
			Endpoint:        os.Getenv("ALIYUN_SMS_ENDPOINT"),
			AccessKeyID:     os.Getenv("ALIYUN_SMS_ACCESS_KEY_ID"),
			AccessKeySecret: os.Getenv("ALIYUN_SMS_ACCESS_KEY_SECRET"),
			SignName:        os.Getenv("ALIYUN_SMS_SIGN_NAME"),
			TemplateCode:    os.Getenv("ALIYUN_SMS_TEMPLATE_CODE"),
			NoticeTemplate:  os.Getenv("ALIYUN_SMS_NOTICE_TEMPLATE"),
		},
		WeChatMini: WeChatMiniConfig{
			AppID: os.Getenv("WECHAT_MINIPROGRAM_APP_ID"), AppSecret: os.Getenv("WECHAT_MINIPROGRAM_APP_SECRET"),
			Endpoint: os.Getenv("WECHAT_MINIPROGRAM_ENDPOINT"),
		},
		PushWebhook: PushWebhookConfig{
			Provider: strings.ToLower(env("PUSH_WEBHOOK_PROVIDER", "webhook")),
			URL:      os.Getenv("PUSH_WEBHOOK_URL"), Secret: os.Getenv("PUSH_WEBHOOK_SECRET"),
		},
		WeChatPay: WeChatPayConfig{
			Endpoint: os.Getenv("WECHAT_PAY_ENDPOINT"), MerchantID: os.Getenv("WECHAT_PAY_MERCHANT_ID"),
			AppID: os.Getenv("WECHAT_PAY_APP_ID"), SerialNumber: os.Getenv("WECHAT_PAY_SERIAL_NUMBER"),
			PrivateKeyPEM:        os.Getenv("WECHAT_PAY_PRIVATE_KEY_PEM"),
			PlatformPublicKeyPEM: os.Getenv("WECHAT_PAY_PLATFORM_PUBLIC_KEY_PEM"),
			PlatformSerialNumber: os.Getenv("WECHAT_PAY_PLATFORM_SERIAL_NUMBER"),
		},
		Alipay: AlipayConfig{
			Endpoint: os.Getenv("ALIPAY_ENDPOINT"), AppID: os.Getenv("ALIPAY_APP_ID"),
			PrivateKeyPEM: os.Getenv("ALIPAY_PRIVATE_KEY_PEM"), PublicKeyPEM: os.Getenv("ALIPAY_PUBLIC_KEY_PEM"),
			NotifyURL: os.Getenv("ALIPAY_NOTIFY_URL"), ReturnURL: os.Getenv("ALIPAY_RETURN_URL"),
		},
		SMTP: SMTPConfig{
			Host: os.Getenv("SMTP_HOST"), Port: envInt("SMTP_PORT", 587), Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"), From: os.Getenv("SMTP_FROM"), TLS: envBool("SMTP_TLS", true),
		},
	}
	cfg.Telemetry = TelemetryConfig{
		OTLPEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		ServiceName:  env("OTEL_SERVICE_NAME", "mars-admin-api"),
	}
	cfg.Bootstrap = BootstrapConfig{
		AdminUsername: env("BOOTSTRAP_ADMIN_USERNAME", "admin"),
		AdminPassword: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
		AdminEmail:    env("BOOTSTRAP_ADMIN_EMAIL", "admin@example.invalid"),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Database.URL == "" {
		return errors.New("DATABASE_URL is required")
	}
	u, err := url.Parse(c.Database.URL)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		return errors.New("DATABASE_URL must be a PostgreSQL URL")
	}
	if c.App.Port < 1 || c.App.Port > 65535 {
		return errors.New("APP_PORT must be between 1 and 65535")
	}
	if c.Database.MaxConnections < 1 || c.Database.MinConnections < 0 || c.Database.MinConnections > c.Database.MaxConnections {
		return errors.New("database pool sizes are invalid")
	}
	switch c.App.Env {
	case "development", "test", "staging", "production":
	default:
		return errors.New("APP_ENV must be development, test, staging, or production")
	}
	if c.App.Env == "production" {
		if c.Storage.Provider == "console" {
			return errors.New("console providers are not allowed in production")
		}
		if c.Bootstrap.AdminPassword != "" && len(c.Bootstrap.AdminPassword) < 16 {
			return errors.New("production bootstrap password must be at least 16 characters")
		}
	}
	if c.Auth.SessionTTL >= c.Auth.RefreshTTL {
		return errors.New("AUTH_SESSION_TTL must be shorter than AUTH_REFRESH_TTL")
	}
	switch c.Auth.CookieSameSite {
	case "strict", "lax", "none":
	default:
		return errors.New("AUTH_COOKIE_SAME_SITE must be strict, lax, or none")
	}
	if c.Auth.CookieSameSite == "none" && !c.Auth.CookieSecure {
		return errors.New("SameSite=None requires AUTH_COOKIE_SECURE=true")
	}
	if c.Storage.MaxUploadBytes < 1 {
		return errors.New("STORAGE_MAX_UPLOAD_BYTES must be positive")
	}
	if c.Storage.Provider == "s3" && (c.Storage.S3.Bucket == "" || c.Storage.S3.Region == "") {
		return errors.New("S3 storage requires STORAGE_S3_BUCKET and STORAGE_S3_REGION")
	}
	switch c.Integrations.SMSProvider {
	case "", "console", "aliyun":
	default:
		return errors.New("SMS_PROVIDER must be console or aliyun")
	}
	if c.App.Env == "production" && (c.Integrations.SMSProvider == "" || c.Integrations.SMSProvider == "console") {
		return errors.New("console SMS is not allowed in production")
	}
	if c.Integrations.SMSProvider == "aliyun" && (c.Integrations.AliyunSMS.AccessKeyID == "" ||
		c.Integrations.AliyunSMS.AccessKeySecret == "" || c.Integrations.AliyunSMS.SignName == "" ||
		c.Integrations.AliyunSMS.TemplateCode == "") {
		return errors.New("aliyun SMS provider configuration is incomplete")
	}
	return nil
}

func (c Config) IsProduction() bool {
	return c.App.Env == "production"
}

func (c Config) Address() string {
	return fmt.Sprintf(":%d", c.App.Port)
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envInt32(key string, fallback int32) int32 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return fallback
	}
	return int32(value) // #nosec G115 -- ParseInt with bitSize 32 guarantees the value range.
}

func envBool(key string, fallback bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
