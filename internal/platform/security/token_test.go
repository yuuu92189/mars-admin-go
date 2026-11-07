package security

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHashAndVerify(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword("correct horse battery staple", hash)
	if err != nil || !ok {
		t.Fatalf("expected valid password, ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword("wrong password", hash)
	if err != nil || ok {
		t.Fatalf("expected invalid password, ok=%v err=%v", ok, err)
	}
}

func TestLegacyBcryptVerificationAndRehashDetection(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("legacy-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	valid, err := VerifyPassword("legacy-password", string(hash))
	if err != nil || !valid {
		t.Fatalf("legacy bcrypt should verify, valid=%v err=%v", valid, err)
	}
	if !NeedsRehash(string(hash)) {
		t.Fatal("legacy bcrypt hash should require rehash")
	}
}

func TestSecretBoxRoundTrip(t *testing.T) {
	key := "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY"
	box, err := NewSecretBox(key)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := box.Encrypt([]byte("provider-secret"))
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := box.Decrypt(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if string(plaintext) != "provider-secret" {
		t.Fatalf("unexpected plaintext %q", plaintext)
	}
}
