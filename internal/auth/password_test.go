package auth

import (
	"strings"
	"testing"
)

func TestHashPassword_ProducesVerifiableHash(t *testing.T) {
	hash := HashPassword("rahasia-sekali-123")

	if !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("unexpected hash format: %s", hash)
	}

	ok, err := VerifyPassword("rahasia-sekali-123", hash)
	if err != nil || !ok {
		t.Fatalf("VerifyPassword(correct) = %v, %v; want true, nil", ok, err)
	}
}

func TestHashPassword_UsesRandomSalt(t *testing.T) {
	if HashPassword("sama") == HashPassword("sama") {
		t.Fatal("two hashes of the same password must differ")
	}
}

func TestVerifyPassword_RejectsWrongPassword(t *testing.T) {
	hash := HashPassword("rahasia-sekali-123")

	ok, err := VerifyPassword("salah", hash)
	if err != nil || ok {
		t.Fatalf("VerifyPassword(wrong) = %v, %v; want false, nil", ok, err)
	}
}

func TestVerifyPassword_RejectsMalformedHash(t *testing.T) {
	tests := []string{
		"",
		"plain-text-password",
		"$bcrypt$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=18$m=19456,t=2,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=abc,t=2,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=19456,t=2,p=1$!!!$aGFzaA",
	}

	for _, encoded := range tests {
		if _, err := VerifyPassword("x", encoded); err == nil {
			t.Errorf("VerifyPassword(%q) expected error", encoded)
		}
	}
}
