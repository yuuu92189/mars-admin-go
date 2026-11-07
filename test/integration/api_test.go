//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"mars-admin-go/internal/bootstrap"
	notificationriver "mars-admin-go/internal/modules/notification/adapter/river"
	"mars-admin-go/internal/platform/config"

	"github.com/coder/websocket"
	redisclient "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestAPIAgainstRealPostgresAndRedis(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	postgresContainer, err := postgres.Run(ctx, "postgres:18.6-bookworm",
		postgres.WithDatabase("mars_admin"),
		postgres.WithUsername("mars"),
		postgres.WithPassword("mars_test_password"),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, postgresContainer)
	databaseURL, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	redisContainer, err := redis.Run(ctx, "redis:8.10.1-alpine")
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, redisContainer)
	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	tempDir := t.TempDir()
	wechatServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"openid":"integration-openid","session_key":"test-session"}`))
	}))
	t.Cleanup(wechatServer.Close)
	cfg := config.Config{
		App: config.AppConfig{
			Env: "test", Name: "integration", Version: "test", Port: 8080,
			PublicBaseURL: "http://example.test", LogLevel: "error", ShutdownTimeout: 10 * time.Second,
			CORSOrigins: []string{"http://localhost:3000"}, AutoMigrate: true,
		},
		Database: config.DatabaseConfig{
			URL: databaseURL, MaxConnections: 5, MinConnections: 1, ConnectionTimeout: 10 * time.Second,
		},
		Redis: config.RedisConfig{URL: "redis://" + redisEndpoint + "/0", PoolSize: 5},
		Auth: config.AuthConfig{
			SessionTTL: 30 * time.Minute, RefreshTTL: 24 * time.Hour,
			CookieSameSite: "lax", LoginMaxAttempts: 5, LoginLockDuration: 10 * time.Minute,
		},
		Storage: config.StorageConfig{
			Provider: "local", LocalDir: filepath.Join(tempDir, "uploads"),
			PublicBaseURL: "http://example.test/files", MaxUploadBytes: 1 << 20,
		},
		Integrations: config.IntegrationConfig{
			SMSProvider: "console",
			WeChatMini: config.WeChatMiniConfig{
				AppID: "integration-app", AppSecret: "integration-secret", Endpoint: wechatServer.URL,
			},
		},
		Bootstrap: config.BootstrapConfig{
			AdminUsername: "admin", AdminPassword: "integration-password-123", AdminEmail: "admin@example.invalid",
		},
	}
	app, err := bootstrap.New(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer closeCancel()
		_ = app.Close(closeCtx)
	})
	smsRecorder := &recordingSMS{}
	notificationWorker, err := notificationriver.NewBackend(ctx, app.Pool, app.Logger, notificationriver.Options{
		SMS: smsRecorder, MaxJobs: 2,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer closeCancel()
		_ = notificationWorker.Stop(closeCtx)
	})

	server := httptest.NewServer(app.Router)
	t.Cleanup(server.Close)

	ready, err := http.Get(server.URL + "/health/ready")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, ready.StatusCode)
	_ = ready.Body.Close()

	body, _ := json.Marshal(map[string]any{
		"username": "admin", "password": "integration-password-123", "clientType": "admin",
	})
	response, err := http.Post(server.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	var login struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refreshToken"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&login))
	_ = response.Body.Close()
	require.NotEmpty(t, login.Token)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/message?token=" + login.Token
	connection, _, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	defer connection.CloseNow()
	_, data, err := connection.Read(ctx)
	require.NoError(t, err)
	require.Contains(t, string(data), `"type":"connected"`)
	require.NoError(t, connection.Write(ctx, websocket.MessageText, []byte(`{"type":"ping"}`)))
	_, data, err = connection.Read(ctx)
	require.NoError(t, err)
	require.Contains(t, string(data), `"type":"pong"`)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/v1/auth/me", nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+login.Token)
	response, err = http.DefaultClient.Do(request)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	data, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	require.Contains(t, string(data), `"username":"admin"`)

	refreshBody, _ := json.Marshal(map[string]string{"refreshToken": login.RefreshToken})
	response, err = http.Post(server.URL+"/api/v1/auth/refresh", "application/json", bytes.NewReader(refreshBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	var refreshed struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&refreshed))
	_ = response.Body.Close()
	require.NotEmpty(t, refreshed.Token)
	login.Token = refreshed.Token
	response, err = http.Post(server.URL+"/api/v1/auth/refresh", "application/json", bytes.NewReader(refreshBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
	_ = response.Body.Close()

	createBody, _ := json.Marshal(map[string]any{
		"username": "integration-user", "password": "integration-user-password",
		"displayName": "Integration User", "userType": "admin", "status": "active",
		"phone": "13800139999",
	})
	response = authenticatedRequest(t, http.MethodPost, server.URL+"/api/v1/admin/users", login.Token, createBody)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	var createdUser struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&createdUser))
	_ = response.Body.Close()
	require.NotEmpty(t, createdUser.ID)

	messageBody, _ := json.Marshal(map[string]any{
		"receiverId": createdUser.ID, "clientMessageId": "integration-message-1",
		"messageType": "text", "content": "hello",
	})
	response = authenticatedRequest(t, http.MethodPost, server.URL+"/api/v1/messages", login.Token, messageBody)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	_ = response.Body.Close()
	response = authenticatedRequest(t, http.MethodPost, server.URL+"/api/v1/messages", login.Token, messageBody)
	require.Equal(t, http.StatusConflict, response.StatusCode)
	_ = response.Body.Close()

	var upload bytes.Buffer
	multipartWriter := multipart.NewWriter(&upload)
	part, err := multipartWriter.CreateFormFile("file", "integration.txt")
	require.NoError(t, err)
	_, _ = io.Copy(part, strings.NewReader("integration file"))
	require.NoError(t, multipartWriter.Close())
	request, err = http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/api/v1/files", &upload)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+login.Token)
	request.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	response, err = http.DefaultClient.Do(request)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	var uploadResult struct {
		File struct {
			ID string `json:"id"`
		} `json:"file"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&uploadResult))
	_ = response.Body.Close()
	require.NotEmpty(t, uploadResult.File.ID)

	response = authenticatedRequest(t, http.MethodGet, server.URL+"/api/v1/admin/audit/events", login.Token, nil)
	require.Equal(t, http.StatusOK, response.StatusCode)
	auditData, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	require.Contains(t, string(auditData), `"action":"post"`)

	redisCache := redisclient.NewClient(&redisclient.Options{Addr: redisEndpoint})
	t.Cleanup(func() { _ = redisCache.Close() })
	dictionaryTypeBody, _ := json.Marshal(map[string]any{
		"code": "integration_dict", "name": "Integration Dictionary", "status": "active",
	})
	response = authenticatedRequest(t, http.MethodPost, server.URL+"/api/v1/admin/dictionaries", login.Token, dictionaryTypeBody)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	var dictionaryType struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&dictionaryType))
	_ = response.Body.Close()
	response = authenticatedRequest(t, http.MethodGet, server.URL+"/api/v1/admin/dictionaries", login.Token, nil)
	require.Equal(t, http.StatusOK, response.StatusCode)
	_ = response.Body.Close()
	cacheKeysAfterRead, err := redisCache.Keys(ctx, "mars_admin:dictionary:*").Result()
	require.NoError(t, err)
	require.NotEmpty(t, cacheKeysAfterRead)
	updateDictionaryBody, _ := json.Marshal(map[string]any{
		"code": "integration_dict", "name": "Integration Dictionary Updated", "status": "active",
	})
	response = authenticatedRequest(t, http.MethodPut, server.URL+"/api/v1/admin/dictionaries/"+dictionaryType.ID, login.Token, updateDictionaryBody)
	require.Equal(t, http.StatusOK, response.StatusCode)
	_ = response.Body.Close()
	cacheKeysAfterUpdate, err := redisCache.Keys(ctx, "mars_admin:dictionary:*").Result()
	require.NoError(t, err)
	require.Empty(t, cacheKeysAfterUpdate)
	accessCacheKeys, err := redisCache.Keys(ctx, "mars_admin:identity:access:*").Result()
	require.NoError(t, err)
	require.NotEmpty(t, accessCacheKeys)
	roleBody, _ := json.Marshal(map[string]any{
		"name": "Integration Role", "code": "integration_role", "dataScope": "self",
	})
	response = authenticatedRequest(t, http.MethodPost, server.URL+"/api/v1/admin/access/roles", login.Token, roleBody)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	_ = response.Body.Close()
	accessCacheKeys, err = redisCache.Keys(ctx, "mars_admin:identity:access:*").Result()
	require.NoError(t, err)
	require.Empty(t, accessCacheKeys)

	socialBody, _ := json.Marshal(map[string]any{
		"loginType": "MINIPROGRAM", "wxCode": "integration-login-code",
	})
	response, err = http.Post(server.URL+"/api/app/auth/login", "application/json", bytes.NewReader(socialBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	var legacySocial struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
			User  struct {
				ID       string `json:"id"`
				UserType string `json:"userType"`
			} `json:"user"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&legacySocial))
	_ = response.Body.Close()
	require.Equal(t, 200, legacySocial.Code)
	require.NotEmpty(t, legacySocial.Data.Token)
	require.Equal(t, "app", legacySocial.Data.User.UserType)

	noticeBody, _ := json.Marshal(map[string]any{
		"title": "Integration SMS notice", "content": "Delivered asynchronously",
		"noticeType": 1, "channels": []string{"station", "sms"}, "targetType": "users",
		"targetIds": []string{createdUser.ID},
	})
	response = authenticatedRequest(t, http.MethodPost, server.URL+"/api/v1/notices", login.Token, noticeBody)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	var notice struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&notice))
	_ = response.Body.Close()
	response = authenticatedRequest(t, http.MethodPost, server.URL+"/api/v1/notices/"+notice.ID+"/publish", login.Token, nil)
	require.Equal(t, http.StatusNoContent, response.StatusCode)
	_ = response.Body.Close()
	require.Eventually(t, func() bool { return smsRecorder.Count() == 1 }, 10*time.Second, 100*time.Millisecond)
	var sendStatus string
	var successCount int
	require.NoError(t, app.Pool.QueryRow(ctx, `
		SELECT status, success_count FROM notification_send_logs
		WHERE notice_id = $1 AND channel = 'sms'
		ORDER BY created_at DESC LIMIT 1`, notice.ID).Scan(&sendStatus, &successCount))
	require.Equal(t, "success", sendStatus)
	require.Equal(t, 1, successCount)

	response = authenticatedRequest(t, http.MethodGet, server.URL+"/api/auth/info", login.Token, nil)
	require.Equal(t, http.StatusOK, response.StatusCode)
	var legacyInfo struct {
		Code int `json:"code"`
		Data struct {
			Menus []map[string]any `json:"menus"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&legacyInfo))
	_ = response.Body.Close()
	require.Equal(t, http.StatusOK, legacyInfo.Code)
	require.NotEmpty(t, legacyInfo.Data.Menus)
	require.Contains(t, legacyInfo.Data.Menus[0], "visible")
	require.Equal(t, float64(1), legacyInfo.Data.Menus[0]["visible"])

	response = authenticatedRequest(t, http.MethodGet, server.URL+"/api/sys/user/page?page=1&pageSize=10", login.Token, nil)
	require.Equal(t, http.StatusOK, response.StatusCode)
	var legacyUsers struct {
		Code int `json:"code"`
		Data struct {
			List []map[string]any `json:"list"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&legacyUsers))
	_ = response.Body.Close()
	require.Equal(t, http.StatusOK, legacyUsers.Code)
	require.NotEmpty(t, legacyUsers.Data.List)
	require.Contains(t, legacyUsers.Data.List[0], "nickname")

	response = authenticatedRequest(t, http.MethodGet, server.URL+"/api/dashboard/stats", login.Token, nil)
	require.Equal(t, http.StatusOK, response.StatusCode)
	var legacyDashboard struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&legacyDashboard))
	_ = response.Body.Close()
	require.Contains(t, legacyDashboard.Data, "userCount")
	require.Contains(t, legacyDashboard.Data, "permissionCount")

	legacyMessageBody, _ := json.Marshal(map[string]any{
		"receiverId": createdUser.ID, "msgType": 1, "content": "legacy compatibility message",
	})
	response = authenticatedRequest(t, http.MethodPost, server.URL+"/api/sys/chat/send", login.Token, legacyMessageBody)
	require.Equal(t, http.StatusOK, response.StatusCode)
	var legacyMessage struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&legacyMessage))
	_ = response.Body.Close()
	require.Equal(t, http.StatusOK, legacyMessage.Code)
	require.Equal(t, float64(1), legacyMessage.Data["msgType"])

	codegenImport, _ := json.Marshal([]string{"access_menus"})
	response = authenticatedRequest(t, http.MethodPost, server.URL+"/api/tool/gen/import", login.Token, codegenImport)
	require.Equal(t, http.StatusOK, response.StatusCode)
	_ = response.Body.Close()
	response = authenticatedRequest(t, http.MethodGet, server.URL+"/api/tool/gen/page?page=1&pageSize=10", login.Token, nil)
	require.Equal(t, http.StatusOK, response.StatusCode)
	var codegenPage struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
			List  []struct {
				ID string `json:"id"`
			} `json:"list"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&codegenPage))
	_ = response.Body.Close()
	require.Equal(t, int64(1), codegenPage.Data.Total)
	require.NotEmpty(t, codegenPage.Data.List)
	response = authenticatedRequest(t, http.MethodGet, server.URL+"/api/tool/gen/preview/"+codegenPage.Data.List[0].ID, login.Token, nil)
	require.Equal(t, http.StatusOK, response.StatusCode)
	_ = response.Body.Close()

}

type recordingSMS struct {
	mu      sync.Mutex
	notices int
}

func (s *recordingSMS) SendCode(context.Context, string, string) error { return nil }

func (s *recordingSMS) SendNotice(context.Context, string, string, map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notices++
	return nil
}

func (s *recordingSMS) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.notices
}

func authenticatedRequest(t *testing.T, method, url, token string, body []byte) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	request, err := http.NewRequestWithContext(context.Background(), method, url, reader)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	if response.StatusCode >= 400 {
		data, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		t.Logf("%s %s returned %d: %s", method, url, response.StatusCode, data)
		response.Body = io.NopCloser(bytes.NewReader(data))
	}
	return response
}
