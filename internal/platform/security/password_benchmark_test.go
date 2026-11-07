package security

import "testing"

func BenchmarkVerifyPassword(b *testing.B) {
	hash, err := HashPassword("benchmark-password-123")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valid, err := VerifyPassword("benchmark-password-123", hash)
		if err != nil || !valid {
			b.Fatalf("verify failed: %v", err)
		}
	}
}
