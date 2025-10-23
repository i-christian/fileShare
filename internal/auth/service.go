package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/i-christian/fileShare/internal/database"
	"github.com/i-christian/fileShare/internal/utils/security"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("token has expired")
	ErrEmailInUse         = errors.New("email already in use")
)

type AuthService struct {
	queries        *database.Queries
	jwtSecret      []byte
	accessTokenTTL time.Duration
}

func NewAuthService(queries *database.Queries, jwtSecret string, accessTokenTTL time.Duration) *AuthService {
	return &AuthService{
		queries:        queries,
		jwtSecret:      []byte(jwtSecret),
		accessTokenTTL: accessTokenTTL,
	}
}

func (s *AuthService) Register(ctx context.Context, email, firstName, lastName, password string) (database.User, error) {
	hashedPassword, err := security.HashPassword(password)
	if err != nil {
		return database.User{}, err
	}

	user, err := s.queries.CreateUser(ctx, database.CreateUserParams{
		FirstName:    firstName,
		LastName:     lastName,
		Email:        email,
		PasswordHash: hashedPassword,
	})

	return user, nil
}
