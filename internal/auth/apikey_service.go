package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/i-christian/fileShare/internal/database"
	"github.com/i-christian/fileShare/internal/utils"
	"github.com/i-christian/fileShare/internal/utils/security"
)

type ApiKeyService struct {
	queries         *database.Queries
	apiKeyPrefix    string
	apiKeySecretLen uint8
	apiKeyPrefixLen uint8
}

func NewApiKeyService(apiKeySecretLen, apiKeyPrefixLen uint8, apiKeyPrefix string, queries *database.Queries) *ApiKeyService {
	return &ApiKeyService{
		apiKeySecretLen: apiKeySecretLen,
		apiKeyPrefixLen: apiKeyPrefixLen,
		apiKeyPrefix:    apiKeyPrefix,
		queries:         queries,
	}
}

// GenerateAPIKey creates a new API key for a user, stores its hash,
// and returns the full, unhashed key one time.
func (s *ApiKeyService) GenerateAPIKey(ctx context.Context, userID uuid.UUID, name string, expires time.Time, scope []database.ApiScope) (string, error) {
	var prefix string
	var err error
	for i := 0; i < 5; i++ {
		randomPart, err := security.GenerateSecureString(s.apiKeyPrefixLen)
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

	secret, err := security.GenerateSecureString(s.apiKeySecretLen)
	if err != nil {
		return "", err
	}

	keyHash, err := security.HashPassword(secret)
	if err != nil {
		return "", err
	}

	newKey, err := s.queries.CreateApiKey(ctx, database.CreateApiKeyParams{
		UserID:    userID,
		Name:      name,
		KeyHash:   keyHash,
		Prefix:    prefix,
		Scope:     scope,
		ExpiresAt: expires,
	})
	if err != nil {
		return "", err
	}

	fullKey := newKey.Prefix + "_" + secret

	return fullKey, nil
}

// ValidateAPIKey checks a full API key string
func (s *ApiKeyService) ValidateAPIKey(ctx context.Context, keyString string) (uuid.UUID, error) {
	parts := strings.SplitN(keyString, "_", 2)
	if len(parts) != 2 {
		return uuid.Nil, errors.New("invalid api key format")
	}

	prefix := parts[0]
	secret := parts[1]

	dBKey, err := s.queries.GetApiKeyByPrefix(ctx, prefix)
	if err != nil {
		if err == sql.ErrNoRows {
			return uuid.Nil, errors.Join(errors.New("invalid api key prefix"), utils.ErrUnexpectedError)
		}
		return uuid.Nil, errors.Join(err, utils.ErrUnexpectedError)
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
