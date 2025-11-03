package security

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const UserIDKey contextKey = "userID"

var alphabet = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// GenerateSecureString creates a cryptographically secure random string.
func GenerateSecureString(length uint8) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	runes := make([]rune, length)
	for i, v := range b {
		runes[i] = alphabet[int(v)%len(alphabet)]
	}
	return string(runes), nil
}

// GenerateTokenHash creates a cryptographically secure 32 byte token hash, and a 26 byte plainText string
func GenerateTokenHash() (tokenHash []byte, plainText string) {
	plainText = rand.Text()
	hash := sha256.Sum256([]byte(plainText))

	tokenHash = hash[:]

	return tokenHash, plainText
}

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
