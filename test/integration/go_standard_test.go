//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	codegenhttp "mars-admin-go/internal/modules/codegen/adapter/http"
	codegenpostgres "mars-admin-go/internal/modules/codegen/adapter/postgres"
	codegenapp "mars-admin-go/internal/modules/codegen/application"
	identitypostgres "mars-admin-go/internal/modules/identity/adapter/postgres"
	identityapp "mars-admin-go/internal/modules/identity/application"
	identitydomain "mars-admin-go/internal/modules/identity/domain"
	messagingpostgres "mars-admin-go/internal/modules/messaging/adapter/postgres"
	messagingapp "mars-admin-go/internal/modules/messaging/application"
	messagingdomain "mars-admin-go/internal/modules/messaging/domain"
	monitoring "mars-admin-go/internal/modules/monitoring"
	notificationpostgres "mars-admin-go/internal/modules/notification/adapter/postgres"
	notificationapp "mars-admin-go/internal/modules/notification/application"
	systempostgres "mars-admin-go/internal/modules/system/adapter/postgres"
	systemapp "mars-admin-go/internal/modules/system/application"
	systemdomain "mars-admin-go/internal/modules/system/domain"
	workflowpostgres "mars-admin-go/internal/modules/workflow/adapter/postgres"
	workflowapp "mars-admin-go/internal/modules/workflow/application"
	workflowdomain "mars-admin-go/internal/modules/workflow/domain"
	settingscache "mars-admin-go/internal/platform/cache"
	"mars-admin-go/internal/platform/config"
	"mars-admin-go/internal/platform/database"
	"mars-admin-go/internal/platform/httpserver"
	"mars-admin-go/internal/platform/requestctx"
	"mars-admin-go/internal/platform/security"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestGovernanceFixesAgainstRealDependencies(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	postgresContainer, err := postgres.Run(ctx, "postgres:18.6-bookworm",
		postgres.WithDatabase("mars_admin_governance"),
		postgres.WithUsername("mars"),
		postgres.WithPassword("mars_test_password"),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, postgresContainer)
	databaseURL, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	redisContainer, err := rediscontainer.Run(ctx, "redis:8.10.1-alpine")
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, redisContainer)
	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	pool, err := database.Open(ctx, config.DatabaseConfig{
		URL: databaseURL, MaxConnections: 20, MinConnections: 2, ConnectionTimeout: 10 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, database.RunMigrations(ctx, pool, "up"))

	redisClient := redis.NewClient(&redis.Options{Addr: redisEndpoint})
	t.Cleanup(func() { _ = redisClient.Close() })
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	identityRepo := identitypostgres.New(pool)

	t.Run("monitoring requires audit read permission", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		service := monitoring.New(pool, redisClient, registry)
		handler := monitoring.NewHandler(service, governanceAuth(requestctx.Actor{
			UserID: uuid.New(), Permissions: []string{"other:read"},
		}), logger)
		response := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/snapshot", nil)
		handler.Routes().ServeHTTP(response, request)
		require.Equal(t, http.StatusForbidden, response.Code)

		allowed := monitoring.NewHandler(service, governanceAuth(requestctx.Actor{
			UserID: uuid.New(), Permissions: []string{"audit:read"},
		}), logger)
		response = httptest.NewRecorder()
		request = httptest.NewRequestWithContext(ctx, http.MethodGet, "/snapshot", nil)
		allowed.Routes().ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code)
	})

	t.Run("profile cannot mutate status or user type", func(t *testing.T) {
		user := createPlainIdentityTestUser(t, ctx, identityRepo, "profile-guard")
		service := identityapp.New(identityRepo, nil, identityapp.Options{})
		_, err := service.UpdateProfile(ctx, user.ID, identitydomain.UpdateUserInput{
			DisplayName: "Changed", Status: "suspended", UserType: "app",
		})
		require.Error(t, err)
		_, err = service.UpdateProfile(ctx, user.ID, identitydomain.UpdateUserInput{
			DisplayName: "Changed", Status: user.Status, UserType: "app",
		})
		require.Error(t, err)
	})

	t.Run("group owner cannot be removed or demoted", func(t *testing.T) {
		owner := createPlainIdentityTestUser(t, ctx, identityRepo, "group-owner")
		admin := createPlainIdentityTestUser(t, ctx, identityRepo, "group-admin")
		repo := messagingpostgres.New(pool)
		service := messagingapp.New(repo)
		group, err := service.CreateGroup(ctx, messagingdomain.CreateGroupInput{
			OwnerID: owner.ID, Name: "governance-group", MemberIDs: []uuid.UUID{admin.ID},
		})
		require.NoError(t, err)
		require.NoError(t, service.SetMemberRole(ctx, group.ID, owner.ID, admin.ID, "admin"))
		require.Error(t, service.RemoveMember(ctx, group.ID, admin.ID, owner.ID))
		require.NoError(t, service.AddMembers(ctx, group.ID, admin.ID, []uuid.UUID{owner.ID}))
		var role string
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT member_role FROM messaging_group_members
			WHERE group_id = $1 AND user_id = $2`, group.ID, owner.ID).Scan(&role))
		require.Equal(t, "owner", role)
		require.NoError(t, service.DissolveGroup(ctx, group.ID, owner.ID))
		_, err = service.SendGroup(ctx, group.ID, owner.ID, "after-dissolve", "text", "must fail", nil)
		require.Error(t, err)
	})

	t.Run("soft deleted username can be reused", func(t *testing.T) {
		hash := "argon2id-placeholder"
		username := "reusable_" + uuid.NewString()[:8]
		first, err := identityRepo.CreateUser(ctx, identitydomain.User{
			ID: uuid.New(), Username: username, DisplayName: username,
			PasswordHash: hash, UserType: "pc", Status: "active", Version: 1,
		}, nil, nil, nil, nil)
		require.NoError(t, err)
		require.NoError(t, identityRepo.SoftDeleteUser(ctx, first.ID))
		second, err := identityRepo.CreateUser(ctx, identitydomain.User{
			ID: uuid.New(), Username: username, DisplayName: username,
			PasswordHash: hash, UserType: "pc", Status: "active", Version: 1,
		}, nil, nil, nil, nil)
		require.NoError(t, err)
		require.NotEqual(t, first.ID, second.ID)
	})

	t.Run("codegen rejects missing table", func(t *testing.T) {
		_, err := codegenpostgres.New(pool).Table(ctx, "public", "does_not_exist_"+uuid.NewString())
		require.ErrorIs(t, err, pgx.ErrNoRows)
	})

	t.Run("codegen writes require manage permission", func(t *testing.T) {
		repo := codegenpostgres.New(pool)
		service := codegenapp.New(repo)
		requestBody := []byte(`{"schema":"public","tables":[]}`)
		readHandler := codegenhttp.New(service, governanceAuth(requestctx.Actor{
			UserID: uuid.New(), Permissions: []string{"codegen:read"},
		}), logger)
		response := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(ctx, http.MethodPost, "/imported", bytes.NewReader(requestBody))
		readHandler.Routes().ServeHTTP(response, request)
		require.Equal(t, http.StatusForbidden, response.Code)

		manageHandler := codegenhttp.New(service, governanceAuth(requestctx.Actor{
			UserID: uuid.New(), Permissions: []string{"codegen:manage"},
		}), logger)
		response = httptest.NewRecorder()
		request = httptest.NewRequestWithContext(ctx, http.MethodPost, "/imported", bytes.NewReader(requestBody))
		manageHandler.Routes().ServeHTTP(response, request)
		require.NotEqual(t, http.StatusForbidden, response.Code)
	})

	t.Run("notification totals and read all cover all pages", func(t *testing.T) {
		user := createPlainIdentityTestUser(t, ctx, identityRepo, "notice-pages")
		repo := notificationpostgres.New(pool)
		service := notificationapp.New(repo)
		noticeIDs := make([]uuid.UUID, 0, 25)
		for index := 0; index < 25; index++ {
			noticeID := uuid.New()
			noticeIDs = append(noticeIDs, noticeID)
			_, err := pool.Exec(ctx, `
				INSERT INTO notification_notices
					(id, title, content, notice_type, channels, target_type, target_ids, status, published_at)
				VALUES ($1, 'title', 'content', 1, '["station"]'::jsonb, 'users', '[]'::jsonb, 'published', now())`,
				noticeID)
			require.NoError(t, err)
			_, err = pool.Exec(ctx, `
				INSERT INTO notification_recipients (notice_id, user_id)
				VALUES ($1, $2)`, noticeID, user.ID)
			require.NoError(t, err)
		}
		count, err := service.MyCount(ctx, user.ID)
		require.NoError(t, err)
		require.Equal(t, int64(25), count)
		page, err := service.My(ctx, user.ID, 1, 20)
		require.NoError(t, err)
		require.Len(t, page, 20)
		require.NoError(t, service.MarkAllRead(ctx, user.ID))
		var unread int64
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT count(*) FROM notification_recipients
			WHERE user_id = $1 AND read_at IS NULL`, user.ID).Scan(&unread))
		require.Zero(t, unread)
		require.NoError(t, service.MarkRead(ctx, noticeIDs[0], user.ID))
	})

	t.Run("sensitive settings are encrypted and batch updates commit", func(t *testing.T) {
		user := createPlainIdentityTestUser(t, ctx, identityRepo, "settings-secret")
		secretBox, err := security.NewSecretBox("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
		require.NoError(t, err)
		service := systemapp.NewSettingsService(
			systempostgres.New(pool), secretBox,
			settingscache.NewJSON(redisClient, "governance:settings", time.Minute),
		)
		require.NoError(t, service.Update(ctx, "system", map[string]any{
			"siteName": "Governance Settings", "accessToken": "must-be-encrypted",
		}, user.ID))
		var sensitive bool
		var stored string
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT s.is_sensitive, s.value_text
			FROM system_settings s
			JOIN system_setting_groups g ON g.id = s.group_id
			WHERE g.code = 'system' AND s.key = 'accessToken'`).Scan(&sensitive, &stored))
		require.True(t, sensitive)
		require.NotEqual(t, "must-be-encrypted", stored)
		group, err := service.Group(ctx, "system", true)
		require.NoError(t, err)
		var encryptedSetting *struct {
			raw  string
			item systemdomain.Setting
		}
		for _, setting := range group.Settings {
			if setting.Key == "accessToken" {
				encryptedSetting = &struct {
					raw  string
					item systemdomain.Setting
				}{raw: string(setting.RawValue), item: setting}
				break
			}
		}
		require.NotNil(t, encryptedSetting)
		decrypted, err := service.Decrypt(encryptedSetting.item, encryptedSetting.raw)
		require.NoError(t, err)
		require.Equal(t, "must-be-encrypted", decrypted)
		var siteName string
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT s.value_text
			FROM system_settings s
			JOIN system_setting_groups g ON g.id = s.group_id
			WHERE g.code = 'system' AND s.key = 'siteName'`).Scan(&siteName))
		require.Equal(t, "Governance Settings", siteName)
	})

	t.Run("rate limit uses a fixed window", func(t *testing.T) {
		limiter := httpserver.NewRateLimit(redisClient, "governance:"+uuid.NewString(), 1, 100*time.Millisecond)
		allowed, err := limiter.Allow(ctx, "subject")
		require.NoError(t, err)
		require.True(t, allowed)
		for index := 0; index < 3; index++ {
			time.Sleep(30 * time.Millisecond)
			allowed, err = limiter.Allow(ctx, "subject")
			require.NoError(t, err)
			require.False(t, allowed)
		}
		time.Sleep(35 * time.Millisecond)
		allowed, err = limiter.Allow(ctx, "subject")
		require.NoError(t, err)
		require.True(t, allowed)
	})

	t.Run("workflow terminate and decide leave no pending task", func(t *testing.T) {
		user := createPlainIdentityTestUser(t, ctx, identityRepo, "workflow-race")
		repo := workflowpostgres.New(pool)
		service := workflowapp.New(repo)
		definition, err := service.CreateDefinition(ctx, user.ID, workflowdomain.CreateDefinitionInput{
			Code: "race_" + uuid.NewString()[:8], Name: "Race Workflow",
			Document: workflowdomain.DefinitionDocument{
				Start: "approve",
				Nodes: []workflowdomain.Node{
					{Code: "approve", Name: "Approve", Type: "approval", AssigneeID: &user.ID, Next: "end"},
					{Code: "end", Name: "End", Type: "end"},
				},
			},
		})
		require.NoError(t, err)
		require.NoError(t, service.Publish(ctx, definition.ID))
		for index := 0; index < 5; index++ {
			instance, task, err := service.Start(ctx, user.ID, workflowdomain.StartInput{
				DefinitionID: definition.ID, BusinessID: "race-" + uuid.NewString(),
			})
			require.NoError(t, err)
			var wait sync.WaitGroup
			wait.Add(2)
			go func() {
				defer wait.Done()
				_, _, _ = service.Decide(ctx, workflowdomain.Actor{UserID: user.ID}, workflowdomain.DecisionInput{
					TaskID: task.ID, Action: "approve",
				})
			}()
			go func() {
				defer wait.Done()
				_, _ = service.Terminate(ctx, instance.ID, user.ID, "race")
			}()
			wait.Wait()
			var status string
			require.NoError(t, pool.QueryRow(ctx, `
				SELECT status FROM workflow_instances WHERE id = $1`, instance.ID).Scan(&status))
			require.Contains(t, []string{"approved", "terminated"}, status)
			var pending int
			require.NoError(t, pool.QueryRow(ctx, `
				SELECT count(*) FROM workflow_tasks
				WHERE instance_id = $1 AND status = 'pending'`, instance.ID).Scan(&pending))
			require.Zero(t, pending)
		}
	})
}

func governanceAuth(actor requestctx.Actor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(requestctx.WithActor(r.Context(), actor)))
		})
	}
}
