package security

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type ContextUser struct {
	FirstName   string
	LastName    string
	Email       string
	Role        string
	UserID      uuid.UUID
	IsActivated bool
}

type contextKey string

const UserContextKey = contextKey("user")

// AnonymousUser holds a pointer to a User struct representing an inactivated user with no ID, name, email or password.
var AnonymousUser = &ContextUser{}

// IsAnonymous checks if a user instance is the AnonymousUser
func (u *ContextUser) IsAnonymous() bool {
	return u == AnonymousUser
}

// SetContextUser function returns a new copy of the request with the provided User struct added to the context
func SetContextUser(r *http.Request, user *ContextUser) *http.Request {
	ctx := context.WithValue(r.Context(), UserContextKey, user)

	return r.WithContext(ctx)
}

// GetUserFromContext retrieves the user from the given HTTP request context.
func GetUserFromContext(r *http.Request) (*ContextUser, bool) {
	user, ok := r.Context().Value(UserContextKey).(*ContextUser)
	return user, ok
}

// ShortProjectPrefix generates a short, deterministic character string based on
// the project name.
func ShortProjectPrefix(projectName string) string {
	name := strings.ToLower(projectName)
	sum := sha1.Sum([]byte(name))
	return name[:min(2, len(name))] + hex.EncodeToString(sum[:2])
}

// GenerateStringAndHash creates a cryptographically secure 32 byte token hash, and a 26 byte plainText string
func GenerateStringAndHash() (plainText string, tokenHash []byte) {
	plainText = rand.Text()
	hash := sha256.Sum256([]byte(plainText))

	tokenHash = hash[:]

	return plainText, tokenHash
}

// GetIPAddress returns the host IP address from r.RemoteAddr
func GetIPAddress(r *http.Request) (string, error) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}

	return host, nil
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

// CalculateChecksum generates a SHA256 hex string from a file stream.
// It reads the file fully, calculates the hash, and then rewinds the file
// pointer back to the start so the file can be read again.
func CalculateChecksum(file io.ReadSeeker) (string, error) {
	hasher := sha256.New()

	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	hashBytes := hasher.Sum(nil)
	checksum := hex.EncodeToString(hashBytes)

	if _, err := file.Seek(0, 0); err != nil {
		return "", fmt.Errorf("failed to reset file pointer: %w", err)
	}

	return checksum, nil
}
