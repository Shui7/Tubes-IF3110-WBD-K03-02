package password

import (
	"strings"
	"testing"
)

func TestHashPasswordAndVerifyPasswordHash(t *testing.T) {
	const plainText = "correct horse battery staple"

	hash, err := HashPassword(plainText)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "" || hash == plainText {
		t.Fatalf("HashPassword() = %q, want a non-empty hashed value", hash)
	}
	if !VerifyPasswordHash(plainText, hash) {
		t.Fatal("VerifyPasswordHash() rejected the matching password")
	}
	if VerifyPasswordHash("wrong password", hash) {
		t.Fatal("VerifyPasswordHash() accepted a different password")
	}
}

func TestVerifyPasswordHashRejectsMalformedHash(t *testing.T) {
	for _, hash := range []string{"", "not-a-bcrypt-hash", "$2a$invalid"} {
		t.Run(hash, func(t *testing.T) {
			if VerifyPasswordHash("password", hash) {
				t.Errorf("VerifyPasswordHash(%q) = true, want false", hash)
			}
		})
	}
}

func TestHashPasswordRejectsPasswordLongerThanBcryptLimit(t *testing.T) {
	_, err := HashPassword(strings.Repeat("x", 73))
	if err == nil {
		t.Fatal("HashPassword() error = nil, want error for password longer than bcrypt limit")
	}
}
