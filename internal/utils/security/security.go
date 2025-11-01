// Package security provides utility functions for hashing, password
// verification, user context management, and generation of short, deterministic
// project prefixes for API keys or identifiers.
package security

import (
	"crypto/sha1"
	"encoding/hex"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const UserIDKey contextKey = "userID"

// GetUserFromContext retrieves the user ID (UUID) from the given HTTP request context.
//
// It returns the UUID and a boolean indicating whether the user ID was found
// and successfully type-asserted. If no user ID is present, the boolean will be false.
func GetUserFromContext(r *http.Request) (uuid.UUID, bool) {
	userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
	return userID, ok
}

// GetIPAddress returns the host IP address from r.RemoteAddr
func GetIPAddress(r *http.Request) (string, error) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}

	return host, nil
}

// ShortProjectPrefix generates a short, deterministic character string based on
// the given project name.
//
// Example:
//
//	ShortProjectPrefix("FileShare") → "file-9a4f"
func ShortProjectPrefix(projectName string) string {
	name := strings.ToLower(projectName)
	sum := sha1.Sum([]byte(name))
	return name[:min(2, len(name))] + hex.EncodeToString(sum[:2])
}

// HashPassword takes a plaintext password and returns its bcrypt hash.
func HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// VerifyPassword compares a bcrypt-hashed password with a plaintext password.
func VerifyPassword(hashedPassword, providedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(providedPassword))
}
