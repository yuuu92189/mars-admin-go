package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func GenerateToken(bytes int) (string, error) {
	if bytes < 32 {
		return "", fmt.Errorf("token entropy must be at least 32 bytes")
	}
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func GenerateNumericCode(length int) (string, error) {
	if length < 4 || length > 10 {
		return "", fmt.Errorf("numeric code length must be between 4 and 10")
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate numeric code: %w", err)
	}
	out := make([]byte, length)
	for i := range buf {
		out[i] = '0' + buf[i]%10
	}
	return string(out), nil
}
