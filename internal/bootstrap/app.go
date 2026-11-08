package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"mars-admin-go/api"
	accesshttp "mars-admin-go/internal/modules/access/adapter/http"
	accesspostgres "mars-admin-go/internal/modules/access/adapter/postgres"
	accessapp "mars-admin-go/internal/modules/access/application"
	audithttp "mars-admin-go/internal/modules/audit/adapter/http"
	auditpostgres "mars-admin-go/internal/modules/audit/adapter/postgres"
	auditapp "mars-admin-go/internal/modules/audit/application"
	codegenhttp "mars-admin-go/internal/modules/codegen/adapter/http"
	codegenpostgres "mars-admin-go/internal/modules/codegen/adapter/postgres"
	codegenapp "mars-admin-go/internal/modules/codegen/application"
	identityhttp "mars-admin-go/internal/modules/identity/adapter/http"
	identitypostgres "mars-admin-go/internal/modules/identity/adapter/postgres"
	identityapp "mars-admin-go/internal/modules/identity/application"
	identitydomain "mars-admin-go/internal/modules/identity/domain"
	"mars-admin-go/internal/modules/integrations"
	messaginghttp "mars-admin-go/internal/modules/messaging/adapter/http"
	messagingpostgres "mars-admin-go/internal/modules/messaging/adapter/postgres"
	messagingapp "mars-admin-go/internal/modules/messaging/application"
	"mars-admin-go/internal/modules/messaging/realtime"
	"mars-admin-go/internal/modules/monitoring"
	notificationhttp "mars-admin-go/internal/modules/notification/adapter/http"
	notificationpostgres "mars-admin-go/internal/modules/notification/adapter/postgres"
	notificationapp "mars-admin-go/internal/modules/notification/application"
	organizationhttp "mars-admin-go/internal/modules/organization/adapter/http"
	organizationpostgres "mars-admin-go/internal/modules/organization/adapter/postgres"
	organizationapp "mars-admin-go/internal/modules/organization/application"
	schedulerhttp "mars-admin-go/internal/modules/scheduler/adapter/http"
	schedulerpostgres "mars-admin-go/internal/modules/scheduler/adapter/postgres"
	schedulerriver "mars-admin-go/internal/modules/scheduler/adapter/river"
	schedulerapp "mars-admin-go/internal/modules/scheduler/application"
	storagehttp "mars-admin-go/internal/modules/storage/adapter/http"
	storagelocal "mars-admin-go/internal/modules/storage/adapter/local"
	storagepostgres "mars-admin-go/internal/modules/storage/adapter/postgres"
	storages3 "mars-admin-go/internal/modules/storage/adapter/s3"
	storageapp "mars-admin-go/internal/modules/storage/application"
	storagedomain "mars-admin-go/internal/modules/storage/domain"
	systemhttp "mars-admin-go/internal/modules/system/adapter/http"
	systempostgres "mars-admin-go/internal/modules/system/adapter/postgres"
	systemapp "mars-admin-go/internal/modules/system/application"
	workflowhttp "mars-admin-go/internal/modules/workflow/adapter/http"
	workflowpostgres "mars-admin-go/internal/modules/workflow/adapter/postgres"
	workflowapp "mars-admin-go/internal/modules/workflow/application"
	"mars-admin-go/internal/platform/cache"
	"mars-admin-go/internal/platform/compat"
	"mars-admin-go/internal/platform/config"
	"mars-admin-go/internal/platform/database"
	"mars-admin-go/internal/platform/httpserver"
	"mars-admin-go/internal/platform/logging"
	"mars-admin-go/internal/platform/security"
	"mars-admin-go/internal/platform/telemetry"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

type App struct {
	Config       config.Config
	Logger       *slog.Logger
	Pool         *pgxpool.Pool
	Redis        *redis.Client
	Router       http.Handler
	telemetry    *telemetry.Provider
	Integrations *integrations.Registry
	hub          *realtime.Hub
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	logger := logging.New(cfg.App.LogLevel, cfg.App.Name, cfg.App.Version)
	slog.SetDefault(logger)
	pool, err := database.Open(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}
	cleanupOnError := func(cause error) (*App, error) {
		pool.Close()
		return nil, cause
	}
	if cfg.App.AutoMigrate {
		if err := database.RunMigrations(ctx, pool, "up"); err != nil {
			return cleanupOnError(err)
		}
	}
	redisClient, err := cache.Open(ctx, cfg.Redis)
	if err != nil {
		return cleanupOnError(err)
	}
	telemetryProvider, err := telemetry.Setup(ctx, cfg.Telemetry, logger)
	if err != nil {
		_ = redisClient.Close()
		return cleanupOnError(err)
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	httpMetrics := httpserver.NewHTTPMetrics(registry)
	monitorService := monitoring.New(pool, redisClient, registry)

	accessRepo := accesspostgres.New(pool)
	accessCache := cache.NewJSON(redisClient, "mars_admin:identity", 5*time.Minute)
	accessService := accessapp.New(accessRepo, accessCache)
	organizationRepo := organizationpostgres.New(pool)
	systemRepo := systempostgres.New(pool)
	dictionaryCache := cache.NewJSON(redisClient, "mars_admin:dictionary", 5*time.Minute)
	settingsCache := cache.NewJSON(redisClient, "mars_admin:settings", 2*time.Minute)
	dictionaryService := systemapp.NewDictionaryService(systemRepo, dictionaryCache)
	var secretBox *security.SecretBox
	if cfg.Auth.SensitiveDataKey != "" {
		secretBox, err = security.NewSecretBox(cfg.Auth.SensitiveDataKey)
		if err != nil {
			_ = redisClient.Close()
			return cleanupOnError(err)
		}
	}
	settingsService := systemapp.NewSettingsService(systemRepo, secretBox, settingsCache)
	auditRepo := auditpostgres.New(pool)
	auditService := auditapp.New(auditRepo, logger)

	providerRegistry, smsProvider, err := integrations.Configure(cfg.Integrations, logger, cfg.IsProduction())
	if err != nil {
		_ = redisClient.Close()
		return cleanupOnError(err)
	}
	var codeSender identityapp.CodeSender
	if smsProvider != nil {
		codeSender = smsProvider
	}
	identityRepo := identitypostgres.New(pool)
	loginLimiter := httpserver.NewRateLimit(redisClient, "auth:login", int64(cfg.Auth.LoginMaxAttempts+5), 5*time.Minute)
	socialProviders := make(map[string]identitydomain.SocialProvider)
	if wechat, err := providerRegistry.Social("wechat_miniprogram"); err == nil {
		if concrete, ok := wechat.(*integrations.WeChatMiniProgram); ok {
			socialProviders["wechat_miniprogram"] = wechatMiniAdapter{provider: concrete}
		}
	}
	identityService := identityapp.New(identityRepo, loginLimiter, identityapp.Options{
		SessionTTL: cfg.Auth.SessionTTL, RefreshTTL: cfg.Auth.RefreshTTL,
		RefreshGracePeriod: cfg.Auth.RefreshGracePeriod,
		MaxLoginAttempts:   cfg.Auth.LoginMaxAttempts, LockDuration: cfg.Auth.LoginLockDuration,
		CodeSender: codeSender, CaptchaEnabled: cfg.Auth.CaptchaEnabled,
		SecretBox: secretBox, SocialProviders: socialProviders, AccessCache: accessCache,
	})
	organizationService := organizationapp.New(organizationRepo, identityService)
	if cfg.Bootstrap.AdminPassword != "" {
		if _, err := identityService.BootstrapAdmin(ctx, cfg.Bootstrap.AdminUsername, cfg.Bootstrap.AdminPassword, cfg.Bootstrap.AdminEmail); err != nil {
			logger.Warn("bootstrap administrator was not created", "error", err)
		}
	}
	authMiddleware := httpserver.Authenticate(identityService.Authenticate)

	messagingRepo := messagingpostgres.New(pool)
	var hub *realtime.Hub
	messagingService := messagingapp.New(messagingRepo)
	hub = realtime.New(logger, identityService, messagingService, redisClient, registry, cfg.App.CORSOrigins...)
	if err := hub.Start(ctx); err != nil {
		_ = redisClient.Close()
		return cleanupOnError(err)
	}

	notificationRepo := notificationpostgres.New(pool)
	notificationService := notificationapp.New(notificationRepo)

	storageBackend, err := buildStorageBackend(cfg, logger)
	if err != nil {
		_ = redisClient.Close()
		return cleanupOnError(err)
	}
	storageRepo := storagepostgres.New(pool)
	storageService := storageapp.New(storageRepo, storageBackend, cfg.Storage.MaxUploadBytes)

	enqueuer, err := schedulerriver.NewEnqueuer(ctx, pool)
	if err != nil {
		_ = redisClient.Close()
		return cleanupOnError(fmt.Errorf("create scheduler enqueuer: %w", err))
	}
	schedulerRepo := schedulerpostgres.New(pool)
	schedulerService := schedulerapp.New(schedulerRepo, enqueuer, []string{"system.health_probe", "storage.cleanup"})
	workflowRepo := workflowpostgres.New(pool)
	workflowService := workflowapp.New(workflowRepo)
	codegenRepo := codegenpostgres.New(pool)
	codegenService := codegenapp.New(codegenRepo)

	identityHandler := identityhttp.New(identityService, accessService, cfg)
	identityHandler.SetAuthMiddleware(authMiddleware)
	accessHandler := accesshttp.New(accessService, authMiddleware, logger)
	organizationHandler := organizationhttp.New(organizationService, authMiddleware, logger)
	systemHandler := systemhttp.New(dictionaryService, settingsService, authMiddleware, logger)
	auditHandler := audithttp.New(auditService, authMiddleware, logger)
	monitoringHandler := monitoring.NewHandler(monitorService, authMiddleware, logger, httpMetrics)
	storageHandler := storagehttp.New(storageService, authMiddleware, logger, cfg.Storage.MaxUploadBytes)
	messagingHandler := messaginghttp.New(messagingService, authMiddleware, logger, hub.Online)
	notificationHandler := notificationhttp.New(notificationService, authMiddleware, logger)
	schedulerHandler := schedulerhttp.New(schedulerService, authMiddleware, logger)
	workflowHandler := workflowhttp.New(workflowService, authMiddleware, logger)
	codegenHandler := codegenhttp.New(codegenService, authMiddleware, logger)
	integrationHandler := integrations.NewHandler(providerRegistry, authMiddleware, logger)

	router := chi.NewRouter()
	router.Use(httpserver.Tracing(cfg.Telemetry.ServiceName))
	router.Use(httpserver.RequestIDMiddleware)
	router.Use(httpserver.Recover(logger))
	router.Use(httpserver.OptionalAuthenticate(identityService.Authenticate))
	router.Use(audithttp.AuditWrites(auditService))
	router.Use(httpserver.SecurityHeaders(cfg.Auth.TrustedProxyCIDRs))
	router.Use(httpserver.CORS(cfg.App.CORSOrigins))
	router.Use(httpserver.CSRF("X-CSRF-Token", "mars_csrf"))
	router.Use(httpserver.Timeout(45 * time.Second))
	router.Use(httpserver.Logging(logger))
	router.Use(httpserver.Metrics(httpMetrics))
	router.Use(httpserver.BodyLimit(1 << 20))
	router.Use(compat.Envelope)

	router.Get("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		httpserver.WriteJSON(w, http.StatusOK, map[string]string{"status": "live"})
	})
	router.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := database.Health(r.Context(), pool); err != nil {
			httpserver.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "database": "unavailable"})
			return
		}
		if err := cache.Health(r.Context(), redisClient); err != nil {
			httpserver.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "redis": "unavailable"})
			return
		}
		httpserver.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	router.With(httpserver.RequirePermission("audit:read")).Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	router.Get("/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		_, _ = w.Write(api.OpenAPISpec)
	})
	router.Get("/docs", scalarDocs)
	router.Handle("/ws/message", hub)
	router.Get("/api/crypto/config", func(w http.ResponseWriter, _ *http.Request) {
		httpserver.WriteJSON(w, http.StatusOK, map[string]any{
			"enabled": false, "publicKey": "", "aesKey": "",
		})
	})

	router.Route("/api/v1/auth", func(r chi.Router) { r.Mount("/", identityHandler.Routes()) })
	router.Route("/api/v1/admin/users", func(r chi.Router) { r.Mount("/", identityHandler.AdminRoutes()) })
	router.Route("/api/v1/admin/access", func(r chi.Router) { r.Mount("/", accessHandler.Routes()) })
	router.Route("/api/v1/admin/organization", func(r chi.Router) { r.Mount("/", organizationHandler.Routes()) })
	router.Route("/api/v1/admin", func(r chi.Router) { r.Mount("/", systemHandler.Routes()) })
	router.Route("/api/v1/admin/audit", func(r chi.Router) { r.Mount("/", auditHandler.Routes()) })
	router.Route("/api/v1/admin/monitoring", func(r chi.Router) { r.Mount("/", monitoringHandler.Routes()) })
	router.Route("/api/dashboard", func(r chi.Router) { r.Mount("/", monitoringHandler.LegacyRoutes()) })
	router.Route("/api/v1/files", func(r chi.Router) { r.Mount("/", storageHandler.Routes()) })
	router.Route("/api/v1", func(r chi.Router) { r.Mount("/", messagingHandler.Routes()) })
	router.Route("/api/v1/notices", func(r chi.Router) { r.Mount("/", notificationHandler.Routes()) })
	router.Route("/api/v1/admin/scheduler", func(r chi.Router) { r.Mount("/", schedulerHandler.Routes()) })
	router.Route("/api/v1/workflows", func(r chi.Router) { r.Mount("/", workflowHandler.Routes()) })
	router.Route("/api/v1/admin/codegen", func(r chi.Router) { r.Mount("/", codegenHandler.Routes()) })
	router.Route("/api/v1/admin/integrations", func(r chi.Router) { r.Mount("/", integrationHandler.Routes()) })
	router.Route("/api/v1/payments/callbacks", func(r chi.Router) { r.Mount("/", integrationHandler.CallbackRoutes()) })

	router.Route("/api/auth", func(r chi.Router) { r.Mount("/", identityHandler.LegacyRoutes()) })
	router.Route("/api/app/auth", func(r chi.Router) { r.Mount("/", identityHandler.LegacyRoutes()) })
	router.Route("/api/web/auth", func(r chi.Router) { r.Mount("/", identityHandler.LegacyRoutes()) })
	router.Route("/api/sys/user", func(r chi.Router) { r.Mount("/", identityHandler.LegacyAdminRoutes()) })
	router.Route("/api/sys/role", func(r chi.Router) { r.Mount("/", accessHandler.LegacyRolesRoutes()) })
	router.Route("/api/sys/menu", func(r chi.Router) { r.Mount("/", accessHandler.LegacyMenuRoutes()) })
	router.Route("/api/sys/dept", func(r chi.Router) { r.Mount("/", organizationHandler.LegacyDepartmentRoutes()) })
	router.Route("/api/sys/post", func(r chi.Router) { r.Mount("/", organizationHandler.LegacyPositionRoutes()) })
	router.Route("/api/sys", func(r chi.Router) { r.Mount("/", systemHandler.LegacyRoutes()) })
	router.Route("/api/monitor", func(r chi.Router) { r.Mount("/", auditHandler.LegacyRoutes()) })
	router.Route("/api/monitor/server", func(r chi.Router) { r.Mount("/", monitoringHandler.LegacyRoutes()) })
	router.Route("/api/monitor/job", func(r chi.Router) { r.Mount("/", schedulerHandler.LegacyRoutes()) })
	router.Route("/api/monitor/online", func(r chi.Router) { r.Mount("/", identityHandler.LegacyOnlineRoutes()) })
	router.Route("/api/monitor/cache", func(r chi.Router) { r.Mount("/", monitoringHandler.LegacyCacheRoutes()) })
	router.Route("/api/monitor/api-access", func(r chi.Router) { r.Mount("/", monitoringHandler.LegacyAPIAccessRoutes()) })
	router.Route("/api/sys/file", func(r chi.Router) { r.Mount("/", storageHandler.LegacyRoutes()) })
	router.Route("/api/sys/file-group", func(r chi.Router) { r.Mount("/", storageHandler.LegacyGroupRoutes()) })
	router.Route("/api/sys/chat", func(r chi.Router) { r.Mount("/", messagingHandler.LegacyDirectRoutes()) })
	router.Route("/api/chat/group", func(r chi.Router) { r.Mount("/", messagingHandler.LegacyGroupRoutes()) })
	router.Route("/api/sys/notice", func(r chi.Router) { r.Mount("/", notificationHandler.LegacyRoutes()) })
	router.Route("/api/workflow", func(r chi.Router) { r.Mount("/", workflowHandler.LegacyRoutes()) })
	router.Route("/api/tool/gen", func(r chi.Router) { r.Mount("/", codegenHandler.LegacyRoutes()) })

	return &App{
		Config: cfg, Logger: logger, Pool: pool, Redis: redisClient,
		Router: router, telemetry: telemetryProvider, Integrations: providerRegistry,
		hub: hub,
	}, nil
}

type wechatMiniAdapter struct {
	provider *integrations.WeChatMiniProgram
}

func (a wechatMiniAdapter) Exchange(ctx context.Context, code string) (identitydomain.SocialIdentity, error) {
	identity, err := a.provider.Exchange(ctx, code)
	if err != nil {
		return identitydomain.SocialIdentity{}, err
	}
	return identitydomain.SocialIdentity{
		Provider: identity.Provider, Subject: identity.Subject, Nickname: identity.Nickname,
		Avatar: identity.Avatar, Email: identity.Email, Phone: identity.Phone, Raw: identity.Raw,
	}, nil
}

func (a *App) Close(ctx context.Context) error {
	var errs []error
	if a.hub != nil {
		errs = append(errs, a.hub.Close(ctx))
	}
	if a.telemetry != nil {
		errs = append(errs, a.telemetry.Shutdown(ctx))
	}
	if a.Redis != nil {
		errs = append(errs, a.Redis.Close())
	}
	if a.Pool != nil {
		a.Pool.Close()
	}
	return errors.Join(errs...)
}

func buildStorageBackend(cfg config.Config, logger *slog.Logger) (storagedomain.Backend, error) {
	switch cfg.Storage.Provider {
	case "local":
		return storagelocal.New(cfg.Storage.LocalDir, cfg.Storage.PublicBaseURL)
	case "s3", "minio":
		return storages3.New(storages3.Config{
			Endpoint: cfg.Storage.S3.Endpoint, Region: cfg.Storage.S3.Region,
			Bucket: cfg.Storage.S3.Bucket, AccessKey: cfg.Storage.S3.AccessKey,
			SecretKey: cfg.Storage.S3.SecretKey, PublicBase: cfg.Storage.PublicBaseURL,
		})
	default:
		logger.Error("unsupported storage provider", "provider", cfg.Storage.Provider)
		return nil, fmt.Errorf("unsupported storage provider %q", cfg.Storage.Provider)
	}
}

func scalarDocs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html>
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Mars Admin Go API</title></head>
<body><script id="api-reference" data-url="/openapi.yaml"></script>
<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script></body>
</html>`))
}
