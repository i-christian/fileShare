package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/i-christian/fileShare/internal/database"
	"github.com/i-christian/fileShare/internal/utils/security"
)

type ApiKeyService struct {
	apiKeySecretLen uint8
	apiKeyPrefixLen uint8
	apiKeyPrefix    string
	queries         *database.Queries
}

func NewApiKeyService(apiKeySecretLen, apiKeyPrefixLen uint8, apiKeyPrefix string, queries *database.Queries) *ApiKeyService {
	return &ApiKeyService{
		apiKeySecretLen: apiKeySecretLen,
		apiKeyPrefixLen: apiKeyPrefixLen,
		apiKeyPrefix:    apiKeyPrefix,
		queries:         queries,
	}
}

var alphabet = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// generateSecureString creates a cryptographically secure random string.
func generateSecureString(length uint8) (string, error) {
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

// GenerateAPIKey creates a new API key for a user, stores its hash,
// and returns the full, unhashed key one time.
func (s *ApiKeyService) GenerateAPIKey(ctx context.Context, userID uuid.UUID, name string, scope []database.ApiScope) (string, error) {
	var prefix string
	var err error
	for i := 0; i < 5; i++ {
		randomPart, err := generateSecureString(s.apiKeyPrefixLen)
		if err != nil {
			return "", err
		}
		prefix = s.apiKeyPrefix + randomPart

		count, err := s.queries.CheckIfAPIKeyExists(ctx, prefix)
		if err != nil {
			return "", err
		}
		if count == 0 {
			break
		}

		if i == 4 {
			return "", errors.New("failed to generate unique api key prefix")
		}
	}

	secret, err := generateSecureString(s.apiKeySecretLen)
	if err != nil {
		return "", err
	}

	keyHash, err := security.HashPassword(secret)
	if err != nil {
		return "", err
	}

	newKey, err := s.queries.CreateApiKey(ctx, database.CreateApiKeyParams{
		UserID:  userID,
		Name:    name,
		KeyHash: keyHash,
		Prefix:  prefix,
		Scope:   scope,
	})
	if err != nil {
		return "", err
	}

	fullKey := newKey.Prefix + "_" + secret

	return fullKey, nil
}

// ValidateAPIKey checks a full API key string...
func (s *ApiKeyService) ValidateAPIKey(ctx context.Context, keyString string) (uuid.UUID, error) {
	parts := strings.SplitN(keyString, "_", 3)
	if len(parts) != 3 || parts[0]+"_" != s.apiKeyPrefix {
		return uuid.Nil, errors.New("invalid api key format")
	}

	prefix := parts[0] + "_" + parts[1]
	secret := parts[2]

	dBKey, err := s.queries.GetApiKeyByPrefix(ctx, prefix)
	if err != nil {
		if err == sql.ErrNoRows {
			return uuid.Nil, errors.New("invalid api key")
		}
		return uuid.Nil, err
	}

	err = security.VerifyPassword(dBKey.KeyHash, secret)
	if err != nil {
		return uuid.Nil, errors.New("invalid api key")
	}

	_ = s.queries.UpdateApiKeyLastUsed(ctx, database.UpdateApiKeyLastUsedParams{
		ApiKeyID:   dBKey.ApiKeyID,
		LastUsedAt: sql.NullTime{Time: time.Now(), Valid: true},
	})

	return dBKey.UserID, nil
}
