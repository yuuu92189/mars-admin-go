package http

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	identitydomain "mars-admin-go/internal/modules/identity/domain"
	"mars-admin-go/internal/platform/config"

	"github.com/stretchr/testify/require"
)

func TestCanonicalLoginInputUsesCanonicalCaptchaFields(t *testing.T) {
	var input identitydomain.LoginInput
	require.NoError(t, json.Unmarshal([]byte(`{
		"username": "canonical-user",
		"password": "secret",
		"captchaId": "019b0000-0000-7000-8000-000000000002",
		"captchaCode": "5678"
	}`), &input))
	require.Equal(t, "019b0000-0000-7000-8000-000000000002", input.CaptchaID)
	require.Equal(t, "5678", input.CaptchaCode)
}

func TestCanonicalRegisterInputUsesCanonicalCaptchaFields(t *testing.T) {
	var input identitydomain.RegisterInput
	require.NoError(t, json.Unmarshal([]byte(`{
		"username": "register-user",
		"password": "correct-horse-battery-staple",
		"displayName": "Register User",
		"captchaId": "019b0000-0000-7000-8000-000000000003",
		"captchaCode": "9012"
	}`), &input))
	require.Equal(t, "019b0000-0000-7000-8000-000000000003", input.CaptchaID)
	require.Equal(t, "9012", input.CaptchaCode)
}

func TestLoginResultJSONUsesAbsoluteExpiry(t *testing.T) {
	resultType := reflect.TypeOf(identitydomain.LoginResult{})
	expiresAt, ok := resultType.FieldByName("ExpiresAt")
	require.True(t, ok)
	require.Equal(t, "expiresAt", expiresAt.Tag.Get("json"))
	_, hasExpiresIn := resultType.FieldByName("ExpiresIn")
	require.False(t, hasExpiresIn)
}

func TestSessionCookiesKeepCSRFValidForRefreshLifetime(t *testing.T) {
	handler := Handler{config: config.Config{
		Auth: config.AuthConfig{
			RefreshTTL:     30 * 24 * time.Hour,
			CookieSameSite: "lax",
		},
	}}
	recorder := httptest.NewRecorder()
	handler.setSessionCookies(recorder, identitydomain.LoginResult{
		AccessToken:  "access",
		RefreshToken: "refresh",
		CSRFToken:    "csrf",
		ExpiresAt:    time.Now().Add(30 * time.Minute),
	})

	var csrfMaxAge int
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "mars_csrf" {
			csrfMaxAge = cookie.MaxAge
		}
	}
	require.Equal(t, int((30 * 24 * time.Hour).Seconds()), csrfMaxAge)
}
