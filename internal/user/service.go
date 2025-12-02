package user

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/i-christian/fileShare/internal/database"
	"github.com/i-christian/fileShare/internal/utils"
	"github.com/i-christian/fileShare/internal/validator"
)

type UserService struct {
	queries *database.Queries
	logger  *slog.Logger
}

func NewUserService(queries *database.Queries, logger *slog.Logger) *UserService {
	return &UserService{
		queries: queries,
		logger:  logger,
	}
}

func (s *UserService) GetUserInfo(ctx context.Context, userID uuid.UUID) (database.GetUserByIDRow, error) {
	return s.queries.GetUserByID(ctx, userID)
}

func (s *UserService) ActivateUser(ctx context.Context, userID uuid.UUID, tokenPlain string, v *validator.Validator) (email string, activated bool, err error) {
	tokenHash := sha256.Sum256([]byte(tokenPlain))

	user, err := s.queries.GetActionTokenForUser(ctx, database.GetActionTokenForUserParams{
		TokenHash: tokenHash[:],
		Purpose:   database.TokenPurposeEmailVerification,
		UserID:    userID,
		ExpiresAt: time.Now(),
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			v.AddError("token", "invalid or expired activation token")
			return "", false, utils.ErrRecordNotFound
		default:
			return "", false, err
		}
	}

	activated, err = s.queries.ActivateUserEmail(ctx, database.ActivateUserEmailParams{
		UserID:  user.UserID,
		Version: user.Version,
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return "", false, utils.ErrEditConflict
		default:
			return "", false, err
		}
	}

	err = s.queries.DeleteActionToken(ctx, database.DeleteActionTokenParams{
		TokenHash: tokenHash[:],
		UserID:    userID,
	})
	if err != nil {
		return "", false, utils.ErrUnexpectedError
	}

	return user.Email, activated, nil
}
