package http

import (
	"encoding/json"
	"reflect"
	"testing"

	identitydomain "mars-admin-go/internal/modules/identity/domain"

	"github.com/stretchr/testify/require"
)

func TestLegacyLoginInputMapsCaptchaAliases(t *testing.T) {
	var input legacyLoginInput
	require.NoError(t, json.Unmarshal([]byte(`{
		"username": "legacy-user",
		"password": "secret",
		"captchaUuid": "019b0000-0000-7000-8000-000000000001",
		"captcha": "1234",
		"rememberMe": true
	}`), &input))
	mapped := input.toDomain()
	require.Equal(t, "019b0000-0000-7000-8000-000000000001", mapped.CaptchaID)
	require.Equal(t, "1234", mapped.CaptchaCode)
	require.True(t, mapped.RememberMe)
}

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

func TestLoginResultJSONUsesAbsoluteExpiry(t *testing.T) {
	resultType := reflect.TypeOf(identitydomain.LoginResult{})
	expiresAt, ok := resultType.FieldByName("ExpiresAt")
	require.True(t, ok)
	require.Equal(t, "expiresAt", expiresAt.Tag.Get("json"))
	_, hasExpiresIn := resultType.FieldByName("ExpiresIn")
	require.False(t, hasExpiresIn)
}
