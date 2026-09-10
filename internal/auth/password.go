package auth

import (
	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultCost is the bcrypt cost factor to use for hashing.
	DefaultCost = 12
)

// HashPassword hashes a plain-text password using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares a bcrypt hashed password with its possible plaintext equivalent.
// Returns true on success, or false on failure.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
