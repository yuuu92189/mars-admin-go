//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	accesspostgres "mars-admin-go/internal/modules/access/adapter/postgres"
	accessapp "mars-admin-go/internal/modules/access/application"
	accessdomain "mars-admin-go/internal/modules/access/domain"
	identitypostgres "mars-admin-go/internal/modules/identity/adapter/postgres"
	identityapp "mars-admin-go/internal/modules/identity/application"
	identitydomain "mars-admin-go/internal/modules/identity/domain"
	notificationpostgres "mars-admin-go/internal/modules/notification/adapter/postgres"
	notificationapp "mars-admin-go/internal/modules/notification/application"
	notificationdomain "mars-admin-go/internal/modules/notification/domain"
	organizationpostgres "mars-admin-go/internal/modules/organization/adapter/postgres"
	organizationapp "mars-admin-go/internal/modules/organization/application"
	organizationdomain "mars-admin-go/internal/modules/organization/domain"
	storagelocal "mars-admin-go/internal/modules/storage/adapter/local"
	storagepostgres "mars-admin-go/internal/modules/storage/adapter/postgres"
	storageapp "mars-admin-go/internal/modules/storage/application"
	storagedomain "mars-admin-go/internal/modules/storage/domain"
	workflowpostgres "mars-admin-go/internal/modules/workflow/adapter/postgres"
	workflowapp "mars-admin-go/internal/modules/workflow/application"
	workflowdomain "mars-admin-go/internal/modules/workflow/domain"
	"mars-admin-go/internal/platform/apperror"
	identitycache "mars-admin-go/internal/platform/cache"
	"mars-admin-go/internal/platform/config"
	"mars-admin-go/internal/platform/database"
	"mars-admin-go/internal/platform/httpserver"
	"mars-admin-go/internal/platform/requestctx"
	"mars-admin-go/internal/platform/security"

	redisclient "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIdentitySecurityAgainstRealPostgresAndRedis(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	postgresContainer, err := postgres.Run(ctx, "postgres:18.6-bookworm",
		postgres.WithDatabase("mars_admin_security"),
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

	repo := identitypostgres.New(pool)
	redisClient := redisclient.NewClient(&redisclient.Options{Addr: redisEndpoint, PoolSize: 20})
	t.Cleanup(func() { _ = redisClient.Close() })
	limiter := httpserver.NewRateLimit(redisClient, "security-test", 20, 10*time.Minute)
	service := identityapp.New(repo, limiter, identityapp.Options{
		SessionTTL: 30 * time.Minute, RefreshTTL: 24 * time.Hour,
		RefreshGracePeriod: 10 * time.Second,
		MaxLoginAttempts:   5, LockDuration: 10 * time.Minute,
	})
	secondService := identityapp.New(repo, limiter, identityapp.Options{
		SessionTTL: 30 * time.Minute, RefreshTTL: 24 * time.Hour,
		RefreshGracePeriod: 10 * time.Second,
		MaxLoginAttempts:   5, LockDuration: 10 * time.Minute,
	})

	user := createIdentityTestUser(t, ctx, repo, "refresh")
	initial := loginIdentityTestUser(t, ctx, service, user.Username)

	t.Run("refresh CAS accepts exactly one of twenty requests", func(t *testing.T) {
		type result struct {
			login identitydomain.LoginResult
			err   error
		}
		start := make(chan struct{})
		results := make(chan result, 20)
		var wait sync.WaitGroup
		for index := range 20 {
			wait.Add(1)
			go func(index int) {
				defer wait.Done()
				<-start
				instance := service
				if index%2 == 1 {
					instance = secondService
				}
				rotated, refreshErr := instance.Refresh(ctx, initial.RefreshToken)
				results <- result{login: rotated, err: refreshErr}
			}(index)
		}
		close(start)
		wait.Wait()
		close(results)

		var successes []identitydomain.LoginResult
		for result := range results {
			if result.err == nil {
				successes = append(successes, result.login)
				continue
			}
			require.Equal(t, 401, apperror.HTTPStatus(result.err), result.err)
		}
		require.Len(t, successes, 1)

		success := successes[0]
		var storedHash []byte
		var generation int64
		var revokedAt *time.Time
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT refresh_token_hash, refresh_generation, revoked_at
			FROM identity_sessions
			WHERE id = $1`, success.SessionID).Scan(&storedHash, &generation, &revokedAt))
		require.Equal(t, security.HashToken(success.RefreshToken), storedHash)
		require.Equal(t, int64(1), generation)
		require.Nil(t, revokedAt, "concurrent refresh must not revoke the winning session")

		_, err := service.Authenticate(ctx, success.AccessToken)
		require.NoError(t, err)
		secondRotation, err := service.Refresh(ctx, success.RefreshToken)
		require.NoError(t, err)
		_, err = secondService.Authenticate(ctx, secondRotation.AccessToken)
		require.NoError(t, err)
	})

	t.Run("refresh grace distinguishes concurrent retries from expired replay", func(t *testing.T) {
		graceUser := createIdentityTestUser(t, ctx, repo, "refresh-grace")
		first := loginIdentityTestUser(t, ctx, service, graceUser.Username)
		rotated, err := service.Refresh(ctx, first.RefreshToken)
		require.NoError(t, err)

		_, err = secondService.Refresh(ctx, first.RefreshToken)
		require.Error(t, err)
		require.Equal(t, 401, apperror.HTTPStatus(err))
		var revokedAt *time.Time
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT revoked_at FROM identity_sessions WHERE id = $1`, rotated.SessionID).Scan(&revokedAt))
		require.Nil(t, revokedAt, "within-grace retry must not revoke the session")
		_, err = service.Authenticate(ctx, rotated.AccessToken)
		require.NoError(t, err)

		secondRotation, err := service.Refresh(ctx, rotated.RefreshToken)
		require.NoError(t, err)
		_, err = service.Authenticate(ctx, secondRotation.AccessToken)
		require.NoError(t, err)

		_, err = pool.Exec(ctx, `
			UPDATE identity_refresh_token_history
			SET rotated_at = now() - interval '1 minute'
			WHERE token_hash = $1`, security.HashToken(first.RefreshToken))
		require.NoError(t, err)
		_, err = secondService.Refresh(ctx, first.RefreshToken)
		require.Error(t, err)
		require.Equal(t, 401, apperror.HTTPStatus(err))
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT revoked_at FROM identity_sessions WHERE id = $1`, rotated.SessionID).Scan(&revokedAt))
		require.NotNil(t, revokedAt, "expired replay must revoke the session family")
		assertAccessTokenRejected(t, ctx, service, secondRotation.AccessToken)
		_, err = service.Refresh(ctx, secondRotation.RefreshToken)
		require.Error(t, err)
	})

	t.Run("refresh history cleanup preserves active families", func(t *testing.T) {
		expiredUser := createPlainIdentityTestUser(t, ctx, repo, "refresh-cleanup-expired")
		expiredLogin := loginIdentityTestUser(t, ctx, service, expiredUser.Username)
		_, err := service.Refresh(ctx, expiredLogin.RefreshToken)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `
			UPDATE identity_sessions SET revoked_at = now() - interval '2 hours' WHERE id = $1`,
			expiredLogin.SessionID)
		require.NoError(t, err)

		activeUser := createPlainIdentityTestUser(t, ctx, repo, "refresh-cleanup-active")
		activeLogin := loginIdentityTestUser(t, ctx, service, activeUser.Username)
		_, err = service.Refresh(ctx, activeLogin.RefreshToken)
		require.NoError(t, err)

		deleted, err := repo.CleanupRefreshHistory(ctx, time.Hour)
		require.NoError(t, err)
		require.GreaterOrEqual(t, deleted, int64(1))
		var expiredCount, activeCount int
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT count(*) FROM identity_refresh_token_history WHERE session_id = $1`,
			expiredLogin.SessionID).Scan(&expiredCount))
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT count(*) FROM identity_refresh_token_history WHERE session_id = $1`,
			activeLogin.SessionID).Scan(&activeCount))
		require.Zero(t, expiredCount)
		require.Equal(t, 1, activeCount)
	})

	t.Run("revoked and expired sessions cannot refresh", func(t *testing.T) {
		revoked := loginIdentityTestUser(t, ctx, service, user.Username)
		require.NoError(t, repo.RevokeSession(ctx, revoked.SessionID))
		_, err := service.Refresh(ctx, revoked.RefreshToken)
		require.Error(t, err)
		require.Equal(t, 401, apperror.HTTPStatus(err))

		expired := loginIdentityTestUser(t, ctx, service, user.Username)
		_, err = pool.Exec(ctx, `
			UPDATE identity_sessions
			SET refresh_expires_at = now() - interval '1 second'
			WHERE id = $1`, expired.SessionID)
		require.NoError(t, err)
		_, err = service.Refresh(ctx, expired.RefreshToken)
		require.Error(t, err)
		require.Equal(t, 401, apperror.HTTPStatus(err))

		frozenUser := createPlainIdentityTestUser(t, ctx, repo, "refresh-frozen")
		frozen := loginIdentityTestUser(t, ctx, service, frozenUser.Username)
		_, err = pool.Exec(ctx, `UPDATE identity_users SET status = 'suspended' WHERE id = $1`, frozenUser.ID)
		require.NoError(t, err)
		_, err = service.Refresh(ctx, frozen.RefreshToken)
		require.Error(t, err)
		require.Equal(t, 401, apperror.HTTPStatus(err))

		deletedUser := createPlainIdentityTestUser(t, ctx, repo, "refresh-deleted")
		deleted := loginIdentityTestUser(t, ctx, service, deletedUser.Username)
		_, err = pool.Exec(ctx, `UPDATE identity_users SET deleted_at = now() WHERE id = $1`, deletedUser.ID)
		require.NoError(t, err)
		_, err = service.Refresh(ctx, deleted.RefreshToken)
		require.Error(t, err)
		require.Equal(t, 401, apperror.HTTPStatus(err))

		updateFailure := loginIdentityTestUser(t, ctx, service, user.Username)
		newAccess, err := security.GenerateToken(32)
		require.NoError(t, err)
		newRefresh, err := security.GenerateToken(48)
		require.NoError(t, err)
		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()
		err = repo.RotateSession(canceledCtx, security.HashToken(updateFailure.RefreshToken), identitydomain.Session{
			ID: updateFailure.SessionID, TokenHash: security.HashToken(newAccess),
			RefreshTokenHash: security.HashToken(newRefresh),
			ExpiresAt:        time.Now().UTC().Add(time.Hour), RefreshExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		})
		require.Error(t, err)
		var storedRefreshHash []byte
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT refresh_token_hash FROM identity_sessions WHERE id = $1`, updateFailure.SessionID).Scan(&storedRefreshHash))
		require.Equal(t, security.HashToken(updateFailure.RefreshToken), storedRefreshHash)

		brokenRedis := redisclient.NewClient(&redisclient.Options{Addr: redisEndpoint})
		require.NoError(t, brokenRedis.Close())
		brokenCacheService := identityapp.New(repo, limiter, identityapp.Options{
			SessionTTL: 30 * time.Minute, RefreshTTL: 24 * time.Hour,
			RefreshGracePeriod: 10 * time.Second,
			AccessCache:        identitycache.NewJSON(brokenRedis, "broken-access", time.Minute),
		})
		cacheFailureLogin := loginIdentityTestUser(t, ctx, brokenCacheService, user.Username)
		cacheFailureRotation, err := brokenCacheService.Refresh(ctx, cacheFailureLogin.RefreshToken)
		require.NoError(t, err)
		_, err = brokenCacheService.Authenticate(ctx, cacheFailureRotation.AccessToken)
		require.NoError(t, err)
	})

	t.Run("password changes and account lifecycle revoke every device", func(t *testing.T) {
		lifecycleUser := createIdentityTestUser(t, ctx, repo, "lifecycle")
		actor := requestctx.Actor{UserID: lifecycleUser.ID, Username: lifecycleUser.Username}
		first := loginIdentityTestUser(t, ctx, service, lifecycleUser.Username)
		second := loginIdentityTestUser(t, ctx, service, lifecycleUser.Username)

		require.NoError(t, service.ChangePassword(ctx, lifecycleUser.ID, "identity-test-password-123", "identity-password-two"))
		assertAccessTokenRejected(t, ctx, service, first.AccessToken)
		assertAccessTokenRejected(t, ctx, service, second.AccessToken)
		require.Equal(t, int64(0), countActiveSessions(t, ctx, pool, lifecycleUser.ID))

		third := loginWithPassword(t, ctx, service, lifecycleUser.Username, "identity-password-two")
		fourth := loginWithPassword(t, ctx, service, lifecycleUser.Username, "identity-password-two")
		_, err := service.ResetPassword(ctx, actor, lifecycleUser.ID, "identity-password-three")
		require.NoError(t, err)
		assertAccessTokenRejected(t, ctx, service, third.AccessToken)
		assertAccessTokenRejected(t, ctx, service, fourth.AccessToken)

		fifth := loginWithPassword(t, ctx, service, lifecycleUser.Username, "identity-password-three")
		current, err := repo.FindUserByID(ctx, lifecycleUser.ID)
		require.NoError(t, err)
		_, err = service.UpdateUser(ctx, actor, lifecycleUser.ID, identitydomain.UpdateUserInput{
			DisplayName: current.DisplayName, Email: current.Email, Phone: current.Phone,
			AvatarURL: current.AvatarURL, UserType: current.UserType, Status: "suspended",
			Gender: current.Gender, Version: current.Version,
		})
		require.NoError(t, err)
		assertAccessTokenRejected(t, ctx, service, fifth.AccessToken)

		current, err = repo.FindUserByID(ctx, lifecycleUser.ID)
		require.NoError(t, err)
		_, err = service.UpdateUser(ctx, actor, lifecycleUser.ID, identitydomain.UpdateUserInput{
			DisplayName: current.DisplayName, Email: current.Email, Phone: current.Phone,
			AvatarURL: current.AvatarURL, UserType: current.UserType, Status: "active",
			Gender: current.Gender, Version: current.Version,
		})
		require.NoError(t, err)
		sixth := loginWithPassword(t, ctx, service, lifecycleUser.Username, "identity-password-three")
		require.NoError(t, service.DeleteUser(ctx, actor, lifecycleUser.ID))
		assertAccessTokenRejected(t, ctx, service, sixth.AccessToken)
		var statusEvents int
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT count(*) FROM outbox_events
			WHERE aggregate_type = 'identity_user'
			  AND aggregate_id = $1
			  AND event_type = 'identity.user.status_changed'`,
			lifecycleUser.ID.String()).Scan(&statusEvents))
		require.GreaterOrEqual(t, statusEvents, 3)
	})

	t.Run("session revocation is scoped to the current user", func(t *testing.T) {
		firstUser := createPlainIdentityTestUser(t, ctx, repo, "session-owner")
		secondUser := createPlainIdentityTestUser(t, ctx, repo, "session-other")
		firstSession := loginIdentityTestUser(t, ctx, service, firstUser.Username)
		otherSession := loginIdentityTestUser(t, ctx, service, secondUser.Username)
		firstActor := requestctx.Actor{UserID: firstUser.ID, SessionID: firstSession.SessionID}

		err := service.RevokeOwnedSession(ctx, firstActor, otherSession.SessionID)
		require.Error(t, err)
		require.Equal(t, 404, apperror.HTTPStatus(err))
		_, err = service.Authenticate(ctx, otherSession.AccessToken)
		require.NoError(t, err)

		secondOwnSession := loginIdentityTestUser(t, ctx, service, firstUser.Username)
		require.NoError(t, service.RevokeOwnedSession(ctx, firstActor, secondOwnSession.SessionID))
		assertAccessTokenRejected(t, ctx, service, secondOwnSession.AccessToken)
		_, err = service.Authenticate(ctx, firstSession.AccessToken)
		require.NoError(t, err)
	})

	t.Run("social first login is concurrency safe and last auth method is retained", func(t *testing.T) {
		subject := "concurrent-social-subject"
		socialService := identityapp.New(repo, limiter, identityapp.Options{
			SessionTTL: 30 * time.Minute, RefreshTTL: 24 * time.Hour,
			SocialProviders: map[string]identitydomain.SocialProvider{
				"test_provider": staticSocialProvider{identity: identitydomain.SocialIdentity{
					Provider: "test_provider", Subject: subject, Nickname: "Concurrent Social",
				}},
			},
		})

		start := make(chan struct{})
		results := make(chan identitydomain.LoginResult, 20)
		errs := make(chan error, 20)
		var wait sync.WaitGroup
		for range 20 {
			wait.Add(1)
			go func() {
				defer wait.Done()
				<-start
				result, err := socialService.LoginSocial(
					ctx, "test_provider", "valid-code", "app", "social-device", "203.0.113.60", "integration-test",
				)
				if err != nil {
					errs <- err
					return
				}
				results <- result
			}()
		}
		close(start)
		wait.Wait()
		close(results)
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}
		var userID uuid.UUID
		count := 0
		for result := range results {
			count++
			if userID == uuid.Nil {
				userID = result.User.ID
			}
			require.Equal(t, userID, result.User.ID)
		}
		require.Equal(t, 20, count)

		var identityCount int
		var passwordLoginEnabled bool
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT count(*), count(DISTINCT user_id)
			FROM identity_social_identities
			WHERE provider = 'test_provider' AND subject = $1`, subject).Scan(&identityCount, &count))
		require.Equal(t, 1, identityCount)
		require.Equal(t, 1, count)
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT password_login_enabled FROM identity_users WHERE id = $1`, userID).Scan(&passwordLoginEnabled))
		require.False(t, passwordLoginEnabled)

		actor := requestctx.Actor{UserID: userID}
		err := socialService.UnbindSocialIdentity(ctx, actor, "test_provider", subject)
		require.Error(t, err)
		require.Equal(t, 409, apperror.HTTPStatus(err))
		require.NoError(t, repo.LinkSocialIdentity(ctx, userID, "backup_provider", "backup-subject", nil))
		require.NoError(t, socialService.UnbindSocialIdentity(ctx, actor, "test_provider", subject))
		err = socialService.UnbindSocialIdentity(ctx, actor, "backup_provider", "backup-subject")
		require.Error(t, err)
		require.Equal(t, 409, apperror.HTTPStatus(err))
	})

	t.Run("permission cache epoch rejects stale Redis entries", func(t *testing.T) {
		user := createPlainIdentityTestUser(t, ctx, repo, "permission-cache")
		roleID := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO access_roles (id, name, code, description, data_scope, status)
			VALUES ($1, 'Permission Cache Role', $2, '', 'self', 'active')`,
			roleID, "permission_cache_"+uuid.NewString()[:8])
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `
			INSERT INTO access_user_roles (user_id, role_id) VALUES ($1, $2)`,
			user.ID, roleID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `
			INSERT INTO access_role_permissions (role_id, permission_id)
			SELECT $1, id FROM access_permissions WHERE code = 'user:read'`, roleID)
		require.NoError(t, err)

		cachePrefix := "authz:" + uuid.NewString()
		accessCache := identitycache.NewJSON(redisClient, cachePrefix, time.Minute)
		cachedService := identityapp.New(repo, limiter, identityapp.Options{
			SessionTTL: 30 * time.Minute, RefreshTTL: 24 * time.Hour, AccessCache: accessCache,
		})
		login := loginWithPassword(t, ctx, cachedService, user.Username, "identity-test-password-123")
		require.Contains(t, login.Access.Permissions, "user:read")
		cacheKeys, err := redisClient.Keys(ctx, cachePrefix+"*").Result()
		require.NoError(t, err)
		require.NotEmpty(t, cacheKeys)

		var epochBefore int64
		require.NoError(t, pool.QueryRow(ctx, `SELECT version FROM access_cache_epoch WHERE singleton`).Scan(&epochBefore))
		_, err = pool.Exec(ctx, `DELETE FROM access_role_permissions WHERE role_id = $1`, roleID)
		require.NoError(t, err)
		var epochAfter int64
		require.NoError(t, pool.QueryRow(ctx, `SELECT version FROM access_cache_epoch WHERE singleton`).Scan(&epochAfter))
		require.Greater(t, epochAfter, epochBefore)

		actor, err := cachedService.Authenticate(ctx, login.AccessToken)
		require.NoError(t, err)
		require.NotContains(t, actor.Permissions, "user:read")
	})

	t.Run("data scope blocks horizontal and vertical user access", func(t *testing.T) {
		suffix := uuid.NewString()[:8]
		departmentA := createTestDepartment(t, ctx, pool, "dept_a_"+suffix, "dept_a_"+suffix)
		departmentB := createTestDepartment(t, ctx, pool, "dept_b_"+suffix, "dept_b_"+suffix)
		plainPassword, err := security.HashPassword("identity-test-password-123")
		require.NoError(t, err)

		actorUser := createPlainIdentityTestUser(t, ctx, repo, "scope-actor")
		require.NoError(t, repo.SetUserRoles(ctx, actorUser.ID, nil))
		_, err = pool.Exec(ctx, `
			INSERT INTO organization_memberships (user_id, department_id, is_primary)
			VALUES ($1, $2, true)`, actorUser.ID, departmentA)
		require.NoError(t, err)
		roleID := uuid.New()
		_, err = pool.Exec(ctx, `
			INSERT INTO access_roles (id, name, code, description, data_scope, status)
			VALUES ($1, 'Department Scope Role', $2, '', 'department', 'active')`,
			roleID, "department_scope_"+suffix)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `INSERT INTO access_user_roles (user_id, role_id) VALUES ($1, $2)`, actorUser.ID, roleID)
		require.NoError(t, err)

		targetA := createIdentityUserInDepartment(t, ctx, repo, "scope-target-a", plainPassword, departmentA, nil)
		targetB := createIdentityUserInDepartment(t, ctx, repo, "scope-target-b", plainPassword, departmentB, nil)
		superTarget := createIdentityUserInDepartment(t, ctx, repo, "scope-super", plainPassword, departmentA, nil)
		superRoleID, err := repo.FindRoleByCode(ctx, "admin")
		require.NoError(t, err)
		require.NoError(t, repo.SetUserRoles(ctx, superTarget.ID, []uuid.UUID{superRoleID}))

		actor := requestctx.Actor{UserID: actorUser.ID, Username: actorUser.Username}
		_, err = service.GetUser(ctx, actor, targetB.ID)
		require.Error(t, err)
		require.Equal(t, 403, apperror.HTTPStatus(err))

		visible, total, err := service.ListUsers(ctx, actor, identitydomain.UserFilter{Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.GreaterOrEqual(t, total, int64(2))
		seen := make(map[uuid.UUID]struct{}, len(visible))
		for _, user := range visible {
			seen[user.ID] = struct{}{}
		}
		_, seesActor := seen[actorUser.ID]
		_, seesTargetA := seen[targetA.ID]
		_, seesTargetB := seen[targetB.ID]
		require.True(t, seesActor)
		require.True(t, seesTargetA)
		require.False(t, seesTargetB)

		organizationService := organizationapp.New(organizationpostgres.New(pool), service)
		visibleDepartments, err := organizationService.Departments(ctx, actor)
		require.NoError(t, err)
		require.Len(t, visibleDepartments, 1)
		require.Equal(t, departmentA, visibleDepartments[0].ID)
		_, err = organizationService.Department(ctx, actor, departmentB)
		require.Error(t, err)
		require.Equal(t, 404, apperror.HTTPStatus(err))
		_, err = organizationService.CreateDepartment(ctx, actor, organizationdomain.DepartmentInput{
			Name: "Scoped Root", Code: "scoped_root_" + suffix, Status: "active",
		})
		require.Error(t, err)
		require.Equal(t, 403, apperror.HTTPStatus(err))
		_, err = organizationService.CreateDepartment(ctx, actor, organizationdomain.DepartmentInput{
			ParentID: &departmentB, Name: "Scoped Child", Code: "scoped_child_" + suffix, Status: "active",
		})
		require.Error(t, err)
		require.Equal(t, 403, apperror.HTTPStatus(err))
		_, err = organizationService.UpdateDepartment(ctx, actor, departmentA, organizationdomain.DepartmentInput{
			ParentID: &departmentB, Name: "Department A", Code: "dept_a_" + suffix, Status: "active",
		})
		require.Error(t, err)
		require.Equal(t, 403, apperror.HTTPStatus(err))

		currentA, err := repo.FindUserByID(ctx, targetA.ID)
		require.NoError(t, err)
		_, err = service.UpdateUser(ctx, actor, targetA.ID, identitydomain.UpdateUserInput{
			DisplayName: currentA.DisplayName, Email: currentA.Email, Phone: currentA.Phone,
			AvatarURL: currentA.AvatarURL, UserType: currentA.UserType, Status: currentA.Status,
			Gender: currentA.Gender, Version: currentA.Version, DepartmentIDs: []uuid.UUID{departmentB},
		})
		require.Error(t, err)
		require.Equal(t, 403, apperror.HTTPStatus(err))

		_, err = service.UpdateUser(ctx, actor, targetA.ID, identitydomain.UpdateUserInput{
			DisplayName: currentA.DisplayName, Email: currentA.Email, Phone: currentA.Phone,
			AvatarURL: currentA.AvatarURL, UserType: currentA.UserType, Status: currentA.Status,
			Gender: currentA.Gender, Version: currentA.Version, RoleIDs: []uuid.UUID{superRoleID},
		})
		require.Error(t, err)
		require.Equal(t, 403, apperror.HTTPStatus(err))

		currentSuper, err := repo.FindUserByID(ctx, superTarget.ID)
		require.NoError(t, err)
		_, err = service.UpdateUser(ctx, actor, superTarget.ID, identitydomain.UpdateUserInput{
			DisplayName: currentSuper.DisplayName, Email: currentSuper.Email, Phone: currentSuper.Phone,
			AvatarURL: currentSuper.AvatarURL, UserType: currentSuper.UserType, Status: currentSuper.Status,
			Gender: currentSuper.Gender, Version: currentSuper.Version,
		})
		require.Error(t, err)
		require.Equal(t, 403, apperror.HTTPStatus(err))
	})

	t.Run("all data scope modes execute in PostgreSQL", func(t *testing.T) {
		suffix := uuid.NewString()[:8]
		rootID := createTestDepartment(t, ctx, pool, "scope_root_"+suffix, "scope_root_"+suffix)
		childID := createTestDepartment(t, ctx, pool, "scope_child_"+suffix, "scope_root_"+suffix+".scope_child_"+suffix)
		passwordHash, err := security.HashPassword("identity-test-password-123")
		require.NoError(t, err)

		treeActor := createIdentityUserInDepartment(t, ctx, repo, "tree-actor", passwordHash, rootID, nil)
		treeTarget := createIdentityUserInDepartment(t, ctx, repo, "tree-target", passwordHash, childID, nil)
		treeRole := createTestRole(t, ctx, pool, "tree_scope_"+suffix, "department_tree")
		require.NoError(t, repo.SetUserRoles(ctx, treeActor.ID, []uuid.UUID{treeRole}))
		treeActorContext := requestctx.Actor{UserID: treeActor.ID}
		users, _, err := service.ListUsers(ctx, treeActorContext, identitydomain.UserFilter{Page: 1, PageSize: 100})
		require.NoError(t, err)
		require.Contains(t, userIDs(users), treeTarget.ID)
		organizationService := organizationapp.New(organizationpostgres.New(pool), service)
		departments, err := organizationService.Departments(ctx, treeActorContext)
		require.NoError(t, err)
		require.ElementsMatch(t, []uuid.UUID{rootID, childID}, departmentIDs(departments))

		customActor := createIdentityUserInDepartment(t, ctx, repo, "custom-actor", passwordHash, rootID, nil)
		customTarget := createIdentityUserInDepartment(t, ctx, repo, "custom-target", passwordHash, rootID, nil)
		customRole := createTestRole(t, ctx, pool, "custom_scope_"+suffix, "custom")
		_, err = pool.Exec(ctx, `
			INSERT INTO access_role_departments (role_id, department_id) VALUES ($1, $2)`,
			customRole, rootID)
		require.NoError(t, err)
		require.NoError(t, repo.SetUserRoles(ctx, customActor.ID, []uuid.UUID{customRole}))
		customActorContext := requestctx.Actor{UserID: customActor.ID}
		users, _, err = service.ListUsers(ctx, customActorContext, identitydomain.UserFilter{Page: 1, PageSize: 100})
		require.NoError(t, err)
		require.Contains(t, userIDs(users), customTarget.ID)
		require.NotContains(t, userIDs(users), treeTarget.ID)

		selfActor := createIdentityUserInDepartment(t, ctx, repo, "self-actor", passwordHash, childID, nil)
		users, total, err := service.ListUsers(ctx, requestctx.Actor{UserID: selfActor.ID}, identitydomain.UserFilter{Page: 1, PageSize: 100})
		require.NoError(t, err)
		require.Equal(t, int64(1), total)
		require.Equal(t, []uuid.UUID{selfActor.ID}, userIDs(users))

		allActor := createIdentityUserInDepartment(t, ctx, repo, "all-actor", passwordHash, rootID, nil)
		allRole := createTestRole(t, ctx, pool, "all_scope_"+suffix, "all")
		require.NoError(t, repo.SetUserRoles(ctx, allActor.ID, []uuid.UUID{allRole}))
		users, _, err = service.ListUsers(ctx, requestctx.Actor{UserID: allActor.ID}, identitydomain.UserFilter{Page: 1, PageSize: 100})
		require.NoError(t, err)
		require.Contains(t, userIDs(users), treeTarget.ID)
		require.Contains(t, userIDs(users), customTarget.ID)
	})

	t.Run("regular administrators cannot modify the super administrator role", func(t *testing.T) {
		accessService := accessapp.New(accesspostgres.New(pool))
		adminRoleID, err := repo.FindRoleByCode(ctx, "admin")
		require.NoError(t, err)
		regularAdmin := createPlainIdentityTestUser(t, ctx, repo, "regular-role-admin")
		actor := requestctx.Actor{UserID: regularAdmin.ID}

		_, err = accessService.UpdateRole(ctx, actor, adminRoleID, accessdomain.UpdateRoleInput{
			Name: "Super Administrator", Description: "should not change", DataScope: "all", Status: "active",
		})
		require.Error(t, err)
		require.Equal(t, 403, apperror.HTTPStatus(err))
		err = accessService.SetRolePermissions(ctx, actor, adminRoleID, []string{"user:delete"})
		require.Error(t, err)
		require.Equal(t, 403, apperror.HTTPStatus(err))
		err = accessService.DeleteRole(ctx, actor, adminRoleID)
		require.Error(t, err)
		require.Equal(t, 403, apperror.HTTPStatus(err))
	})

	t.Run("file ownership blocks horizontal access", func(t *testing.T) {
		owner := createPlainIdentityTestUser(t, ctx, repo, "file-owner")
		other := createPlainIdentityTestUser(t, ctx, repo, "file-other")
		backend, err := storagelocal.New(filepath.Join(t.TempDir(), "uploads"), "http://example.test/files")
		require.NoError(t, err)
		storageService := storageapp.New(storagepostgres.New(pool), backend, 1<<20)
		file, err := storageService.Upload(ctx, storagedomain.Upload{
			OwnerUserID: owner.ID, Filename: "private.txt", ContentType: "text/plain",
			Reader: bytes.NewReader([]byte("private file")),
		})
		require.NoError(t, err)

		otherActor := requestctx.Actor{UserID: other.ID}
		items, total, err := storageService.List(ctx, otherActor, nil, "", 1, 20)
		require.NoError(t, err)
		require.Zero(t, total)
		require.Empty(t, items)
		_, err = storageService.Get(ctx, otherActor, file.ID)
		require.Error(t, err)
		require.Equal(t, 404, apperror.HTTPStatus(err))
		require.Error(t, storageService.Delete(ctx, otherActor, file.ID))

		ownerActor := requestctx.Actor{UserID: owner.ID}
		got, err := storageService.Get(ctx, ownerActor, file.ID)
		require.NoError(t, err)
		require.Equal(t, file.ID, got.ID)
		superActor := requestctx.Actor{UserID: owner.ID, IsSuperAdmin: true}
		superItems, superTotal, err := storageService.List(ctx, superActor, nil, "", 1, 20)
		require.NoError(t, err)
		require.Equal(t, int64(1), superTotal)
		require.Len(t, superItems, 1)
	})

	t.Run("targeted notices are visible only to recipients and managers", func(t *testing.T) {
		creator := createPlainIdentityTestUser(t, ctx, repo, "notice-creator")
		recipient := createPlainIdentityTestUser(t, ctx, repo, "notice-recipient")
		other := createPlainIdentityTestUser(t, ctx, repo, "notice-other")
		noticeService := notificationapp.New(notificationpostgres.New(pool))
		notice, err := noticeService.Create(ctx, creator.ID, notificationdomain.Input{
			Title: "Targeted notice", Content: "Only the recipient may read this.",
			NoticeType: 1, Channels: []string{"station"}, TargetType: "users",
			TargetIDs: []uuid.UUID{recipient.ID},
		})
		require.NoError(t, err)
		require.NoError(t, noticeService.Publish(ctx, notice.ID))

		recipientActor := requestctx.Actor{UserID: recipient.ID}
		got, err := noticeService.GetForActor(ctx, recipientActor, notice.ID)
		require.NoError(t, err)
		require.Equal(t, notice.ID, got.ID)

		otherActor := requestctx.Actor{UserID: other.ID}
		_, err = noticeService.GetForActor(ctx, otherActor, notice.ID)
		require.Error(t, err)
		require.Equal(t, 404, apperror.HTTPStatus(err))

		managerActor := requestctx.Actor{UserID: creator.ID, Permissions: []string{"notice:manage"}}
		_, err = noticeService.GetForActor(ctx, managerActor, notice.ID)
		require.NoError(t, err)
	})

	t.Run("workflow start and approval are idempotent under concurrency", func(t *testing.T) {
		workflowService := workflowapp.New(workflowpostgres.New(pool))
		initiator := createPlainIdentityTestUser(t, ctx, repo, "workflow")
		assigneeID := initiator.ID
		definition, err := workflowService.CreateDefinition(ctx, initiator.ID, workflowdomain.CreateDefinitionInput{
			Code: "integration_approval_" + uuid.NewString()[:8], Name: "Integration Approval", Version: 1,
			Document: workflowdomain.DefinitionDocument{
				Start: "approval",
				Nodes: []workflowdomain.Node{
					{Code: "approval", Name: "Approval", Type: "approval", AssigneeID: &assigneeID, Next: "end"},
					{Code: "end", Name: "End", Type: "end"},
				},
			},
		})
		require.NoError(t, err)
		require.NoError(t, workflowService.Publish(ctx, definition.ID))

		type startResult struct {
			instance workflowdomain.Instance
			task     workflowdomain.Task
			err      error
		}
		start := make(chan struct{})
		startResults := make(chan startResult, 2)
		var startWait sync.WaitGroup
		for range 2 {
			startWait.Add(1)
			go func() {
				defer startWait.Done()
				<-start
				instance, task, err := workflowService.Start(ctx, initiator.ID, workflowdomain.StartInput{
					DefinitionID: definition.ID, BusinessID: "business-" + definition.ID.String(),
				})
				startResults <- startResult{instance: instance, task: task, err: err}
			}()
		}
		close(start)
		startWait.Wait()
		close(startResults)
		var createdTask workflowdomain.Task
		startSuccesses := 0
		for result := range startResults {
			if result.err != nil {
				require.Equal(t, 409, apperror.HTTPStatus(result.err), result.err)
				continue
			}
			startSuccesses++
			createdTask = result.task
		}
		require.Equal(t, 1, startSuccesses)
		require.NotEqual(t, uuid.Nil, createdTask.ID)

		actor := workflowdomain.Actor{UserID: initiator.ID, Roles: []string{"admin"}}
		type decisionResult struct {
			task workflowdomain.Task
			err  error
		}
		decisionStart := make(chan struct{})
		decisionResults := make(chan decisionResult, 2)
		var decisionWait sync.WaitGroup
		for range 2 {
			decisionWait.Add(1)
			go func() {
				defer decisionWait.Done()
				<-decisionStart
				task, _, err := workflowService.Decide(ctx, actor, workflowdomain.DecisionInput{
					TaskID: createdTask.ID, Action: "approve", Comment: "concurrent approval",
				})
				decisionResults <- decisionResult{task: task, err: err}
			}()
		}
		close(decisionStart)
		decisionWait.Wait()
		close(decisionResults)
		decisionSuccesses := 0
		for result := range decisionResults {
			if result.err != nil {
				require.Equal(t, 409, apperror.HTTPStatus(result.err), result.err)
				continue
			}
			decisionSuccesses++
		}
		require.Equal(t, 1, decisionSuccesses)
		var outboxCount int
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT count(*) FROM outbox_events
			WHERE aggregate_type = 'workflow_instance'
			  AND event_type = 'workflow.task.decided'
			  AND deduplication_key = $1`, createdTask.ID.String()).Scan(&outboxCount))
		require.Equal(t, 1, outboxCount)
	})

	t.Run("department hierarchy rejects cycles and rolls back", func(t *testing.T) {
		organizationService := organizationapp.New(organizationpostgres.New(pool))
		actor := requestctx.Actor{UserID: uuid.New(), IsSuperAdmin: true}
		root, err := organizationService.CreateDepartment(ctx, actor, organizationdomain.DepartmentInput{
			Name: "Cycle Root", Code: "cycle_root_" + uuid.NewString()[:8], Status: "active",
		})
		require.NoError(t, err)
		child, err := organizationService.CreateDepartment(ctx, actor, organizationdomain.DepartmentInput{
			ParentID: &root.ID, Name: "Cycle Child", Code: "cycle_child_" + uuid.NewString()[:8], Status: "active",
		})
		require.NoError(t, err)
		_, err = organizationService.UpdateDepartment(ctx, actor, root.ID, organizationdomain.DepartmentInput{
			ParentID: &child.ID, Name: root.Name, Code: root.Code, Status: root.Status,
		})
		require.Error(t, err)

		reloaded, err := organizationService.Department(ctx, actor, root.ID)
		require.NoError(t, err)
		require.Nil(t, reloaded.ParentID)
	})

	t.Run("SMS attempts increment atomically and exhaust at five", func(t *testing.T) {
		phone := "13900000001"
		insertTestCode(t, ctx, repo, phone, "123456", 5*time.Minute)

		for attempt := 1; attempt <= 4; attempt++ {
			result, attemptErr := repo.AttemptOneTimeCode(ctx, "login_sms", phone, security.HashToken("000000"))
			require.NoError(t, attemptErr)
			require.False(t, result.Valid)
			require.Equal(t, int32(attempt), result.Attempts)
		}
		var attempts int32
		var consumedAt *time.Time
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT attempts, consumed_at
			FROM identity_one_time_codes
			WHERE purpose = 'login_sms' AND subject = $1
			ORDER BY created_at DESC LIMIT 1`, phone).Scan(&attempts, &consumedAt))
		require.Equal(t, int32(4), attempts)
		require.Nil(t, consumedAt)

		result, err := repo.AttemptOneTimeCode(ctx, "login_sms", phone, security.HashToken("000000"))
		require.NoError(t, err)
		require.False(t, result.Valid)
		require.Equal(t, int32(5), result.Attempts)
		require.True(t, result.Consumed)

		result, err = repo.AttemptOneTimeCode(ctx, "login_sms", phone, security.HashToken("123456"))
		require.NoError(t, err)
		require.False(t, result.Valid)
		require.True(t, result.Consumed)
	})

	t.Run("SMS concurrent correct code has one winner", func(t *testing.T) {
		phone := "13900000002"
		insertTestCode(t, ctx, repo, phone, "654321", 5*time.Minute)

		start := make(chan struct{})
		var winners atomic.Int32
		errs := make(chan error, 20)
		var wait sync.WaitGroup
		for range 20 {
			wait.Add(1)
			go func() {
				defer wait.Done()
				<-start
				result, err := repo.AttemptOneTimeCode(ctx, "login_sms", phone, security.HashToken("654321"))
				if err != nil {
					errs <- err
					return
				}
				if result.Valid {
					winners.Add(1)
				}
			}()
		}
		close(start)
		wait.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}
		require.Equal(t, int32(1), winners.Load())
	})

	t.Run("expired and consumed codes fail", func(t *testing.T) {
		expiredPhone := "13900000003"
		insertTestCode(t, ctx, repo, expiredPhone, "111111", -time.Second)
		result, err := repo.AttemptOneTimeCode(ctx, "login_sms", expiredPhone, security.HashToken("111111"))
		require.NoError(t, err)
		require.False(t, result.Valid)

		consumedPhone := "13900000004"
		insertTestCode(t, ctx, repo, consumedPhone, "222222", 5*time.Minute)
		result, err = repo.AttemptOneTimeCode(ctx, "login_sms", consumedPhone, security.HashToken("222222"))
		require.NoError(t, err)
		require.True(t, result.Valid)
		result, err = repo.AttemptOneTimeCode(ctx, "login_sms", consumedPhone, security.HashToken("222222"))
		require.NoError(t, err)
		require.False(t, result.Valid)
	})

	t.Run("IP verification quota cannot be bypassed by changing phones", func(t *testing.T) {
		ip := "203.0.113.44"
		for attempt := 0; attempt < 20; attempt++ {
			phone := fmt.Sprintf("1391000%04d", attempt)
			insertTestCode(t, ctx, repo, phone, "333333", 5*time.Minute)
			_, err := service.LoginSMS(ctx, phone, "000000", "app", "device-ip-bypass", ip, "integration-test")
			require.Error(t, err)
			require.Equal(t, 401, apperror.HTTPStatus(err))
		}

		phone := "13910009999"
		insertTestCode(t, ctx, repo, phone, "333333", 5*time.Minute)
		_, err := service.LoginSMS(ctx, phone, "000000", "app", "device-ip-bypass", ip, "integration-test")
		require.Error(t, err)
		require.Equal(t, 429, apperror.HTTPStatus(err))
		var attempts int32
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT attempts FROM identity_one_time_codes
			WHERE purpose = 'login_sms' AND subject = $1
			ORDER BY created_at DESC LIMIT 1`, phone).Scan(&attempts))
		require.Zero(t, attempts)

		otherPhone := "13910008888"
		insertTestCode(t, ctx, repo, otherPhone, "444444", 5*time.Minute)
		_, err = service.LoginSMS(ctx, otherPhone, "000000", "app", "other-device", "203.0.113.45", "integration-test")
		require.Error(t, err)
		require.Equal(t, 401, apperror.HTTPStatus(err))
	})

	t.Run("Redis failure fails SMS closed before consuming a code", func(t *testing.T) {
		phone := "13900000005"
		insertTestCode(t, ctx, repo, phone, "555555", 5*time.Minute)
		brokenRedis := redisclient.NewClient(&redisclient.Options{Addr: redisEndpoint})
		require.NoError(t, brokenRedis.Close())
		brokenLimiter := httpserver.NewRateLimit(brokenRedis, "broken", 10, time.Minute)
		brokenService := identityapp.New(repo, brokenLimiter, identityapp.Options{
			SessionTTL: 30 * time.Minute, RefreshTTL: 24 * time.Hour,
		})

		_, err := brokenService.LoginSMS(
			ctx, phone, "555555", "app", "redis-failure", "203.0.113.46", "integration-test",
		)
		require.Error(t, err)
		require.Equal(t, 503, apperror.HTTPStatus(err))

		var attempts int32
		var consumedAt *time.Time
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT attempts, consumed_at
			FROM identity_one_time_codes
			WHERE purpose = 'login_sms' AND subject = $1
			ORDER BY created_at DESC LIMIT 1`, phone).Scan(&attempts, &consumedAt))
		require.Zero(t, attempts)
		require.Nil(t, consumedAt)
	})
}

type staticSocialProvider struct {
	identity identitydomain.SocialIdentity
}

func (p staticSocialProvider) Exchange(context.Context, string) (identitydomain.SocialIdentity, error) {
	return p.identity, nil
}

func createIdentityTestUser(t *testing.T, ctx context.Context, repo *identitypostgres.Repository, suffix string) identitydomain.User {
	t.Helper()
	user := createPlainIdentityTestUser(t, ctx, repo, suffix)
	roleID, err := repo.FindRoleByCode(ctx, "admin")
	require.NoError(t, err)
	require.NoError(t, repo.SetUserRoles(ctx, user.ID, []uuid.UUID{roleID}))
	return user
}

func createPlainIdentityTestUser(t *testing.T, ctx context.Context, repo *identitypostgres.Repository, suffix string) identitydomain.User {
	t.Helper()
	hash, err := security.HashPassword("identity-test-password-123")
	require.NoError(t, err)
	now := time.Now().UTC()
	user, err := repo.CreateUser(ctx, identitydomain.User{
		ID:           uuid.New(),
		Username:     "identity_" + suffix + "_" + uuid.NewString()[:8],
		PasswordHash: hash,
		DisplayName:  "Identity Test",
		UserType:     "admin",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
		Version:      1,
	}, nil, nil, nil, nil)
	require.NoError(t, err)
	return user
}

func createTestDepartment(t *testing.T, ctx context.Context, pool *pgxpool.Pool, code, path string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO organization_departments (id, name, code, path)
		VALUES ($1, $2, $3, $4::ltree)`, id, code, code, path)
	require.NoError(t, err)
	return id
}

func createTestRole(t *testing.T, ctx context.Context, pool *pgxpool.Pool, code, dataScope string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO access_roles (id, name, code, description, data_scope, status)
		VALUES ($1, $2, $3, '', $4, 'active')`, id, code, code, dataScope)
	require.NoError(t, err)
	return id
}

func userIDs(users []identitydomain.User) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	return ids
}

func departmentIDs(departments []organizationdomain.Department) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(departments))
	for _, department := range departments {
		ids = append(ids, department.ID)
	}
	return ids
}

func createIdentityUserInDepartment(
	t *testing.T,
	ctx context.Context,
	repo *identitypostgres.Repository,
	suffix, passwordHash string,
	departmentID uuid.UUID,
	roleIDs []uuid.UUID,
) identitydomain.User {
	t.Helper()
	user, err := repo.CreateUser(ctx, identitydomain.User{
		ID:           uuid.New(),
		Username:     "identity_" + suffix + "_" + uuid.NewString()[:8],
		PasswordHash: passwordHash,
		DisplayName:  "Identity Scope Target",
		UserType:     "admin",
		Status:       "active",
	}, roleIDs, []uuid.UUID{departmentID}, nil, &departmentID)
	require.NoError(t, err)
	if len(roleIDs) == 0 {
		roleIDs = nil
	}
	return user
}

func loginIdentityTestUser(t *testing.T, ctx context.Context, service *identityapp.Service, username string) identitydomain.LoginResult {
	t.Helper()
	return loginWithPassword(t, ctx, service, username, "identity-test-password-123")
}

func loginWithPassword(t *testing.T, ctx context.Context, service *identityapp.Service, username, password string) identitydomain.LoginResult {
	t.Helper()
	result, err := service.Login(ctx, identitydomain.LoginInput{
		Username: username, Password: password,
		ClientType: "admin", DeviceName: "integration", IPAddress: "127.0.0.1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.AccessToken)
	require.NotEmpty(t, result.RefreshToken)
	return result
}

func assertAccessTokenRejected(t *testing.T, ctx context.Context, service *identityapp.Service, token string) {
	t.Helper()
	_, err := service.Authenticate(ctx, token)
	require.Error(t, err)
	require.Equal(t, 401, apperror.HTTPStatus(err))
}

func countActiveSessions(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) int64 {
	t.Helper()
	var count int64
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT count(*) FROM identity_sessions
		WHERE user_id = $1 AND revoked_at IS NULL`, userID).Scan(&count))
	return count
}

func insertTestCode(t *testing.T, ctx context.Context, repo *identitypostgres.Repository, phone, code string, ttl time.Duration) {
	t.Helper()
	err := repo.InsertOneTimeCode(ctx, "login_sms", phone, security.HashToken(code), time.Now().UTC().Add(ttl))
	require.NoError(t, err)
}
