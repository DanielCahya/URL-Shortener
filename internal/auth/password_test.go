package auth_test

import (
	"testing"

	"github.com/DanielCahya/url-shortener/internal/auth"
)

func TestPasswordHashing(t *testing.T) {
	password := "my_secure_password"

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if hash == password {
		t.Fatalf("hash should not equal plain text password")
	}

	// Verify correct password
	if !auth.CheckPasswordHash(password, hash) {
		t.Fatalf("expected CheckPasswordHash to return true for correct password")
	}

	// Verify incorrect password
	if auth.CheckPasswordHash("wrong_password", hash) {
		t.Fatalf("expected CheckPasswordHash to return false for incorrect password")
	}
}
