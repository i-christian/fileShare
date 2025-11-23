package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/i-christian/fileShare/internal/database"
	"github.com/i-christian/fileShare/internal/utils"
	"github.com/i-christian/fileShare/internal/utils/security"
	"github.com/i-christian/fileShare/internal/worker"
)

type APIKeyService struct {
	queries         *database.Queries
	logger          *slog.Logger
	wg              *sync.WaitGroup
	apiKeyPrefix    string
	apiKeyPrefixLen uint8
}

func NewAPIKeyService(apiKeyPrefixLen uint8, apiKeyPrefix string, queries *database.Queries, logger *slog.Logger, wg *sync.WaitGroup) *APIKeyService {
	return &APIKeyService{
		apiKeyPrefixLen: apiKeyPrefixLen,
		apiKeyPrefix:    apiKeyPrefix,
		queries:         queries,
		logger:          logger,
		wg:              wg,
	}
}

// GenerateAPIKey creates a new API key for a user, stores its hash,
// and returns the full, unhashed key one time.
func (s *APIKeyService) GenerateAPIKey(ctx context.Context, userID uuid.UUID, name string, expires time.Time, scope []database.ApiScope) (string, error) {
	var prefix string
	var err error
	for i := 0; i < 5; i++ {
		plainText, _ := security.GenerateStringAndHash()

		prefix = s.apiKeyPrefix + plainText[:s.apiKeyPrefixLen]

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

	secret, _ := security.GenerateStringAndHash()
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
func (s *APIKeyService) ValidateAPIKey(ctx context.Context, keyString string) (*security.ContextUser, error) {
	parts := strings.SplitN(keyString, "_", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid api key format")
	}

	prefix := parts[0]
	secret := parts[1]

	dBKey, err := s.queries.GetApiKeyByPrefix(ctx, prefix)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("invalid api key prefix")
		}
		return nil, errors.Join(err, utils.ErrUnexpectedError)
	}

	if !dBKey.ExpiresAt.IsZero() && time.Now().After(dBKey.ExpiresAt) {
		return nil, errors.New("api key has expired")
	}

	err = security.VerifyPassword(dBKey.KeyHash, secret)
	if err != nil {
		return nil, errors.New("invalid api key")
	}

	worker.BackgroundTask(s.wg, s.logger, func(l *slog.Logger) {
		params := database.UpdateApiKeyLastUsedParams{
			ApiKeyID:   dBKey.ApiKeyID,
			LastUsedAt: sql.NullTime{Time: time.Now(), Valid: true},
		}

		err := s.queries.UpdateApiKeyLastUsed(context.Background(), params)
		if err != nil {
			l.Error(
				"failed to update api key last used time",
				"error", err,
				"api_key_id", dBKey.ApiKeyID,
			)
		}
	})

	return &security.ContextUser{
		FirstName:   dBKey.FirstName,
		LastName:    dBKey.LastName,
		Email:       dBKey.Email,
		Role:        string(dBKey.Role),
		UserID:      dBKey.UserID,
		IsActivated: dBKey.IsVerified,
	}, nil
}
