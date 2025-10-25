package user

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/i-christian/fileShare/internal/database"
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
