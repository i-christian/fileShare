package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/i-christian/fileShare/internal/database"
	"github.com/i-christian/fileShare/internal/utils"
	"github.com/i-christian/fileShare/internal/utils/security"
)

type AuthService struct {
	queries        *database.Queries
	Logger         *slog.Logger
	jwtSecret      []byte
	accessTokenTTL time.Duration
}

func NewAuthService(queries *database.Queries, jwtSecret string, accessTokenTTL time.Duration, logger *slog.Logger) *AuthService {
	return &AuthService{
		queries:        queries,
		jwtSecret:      []byte(jwtSecret),
		accessTokenTTL: accessTokenTTL,
		Logger:         logger,
	}
}

func (s *AuthService) Register(ctx context.Context, email, firstName, lastName, password string) (ApiUser, error) {
	hashedPassword, err := security.HashPassword(password)
	if err != nil {
		return ApiUser{}, err
	}

	user, err := s.queries.CreateUser(ctx, database.CreateUserParams{
		FirstName:    firstName,
		LastName:     lastName,
		Email:        email,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return ApiUser{}, utils.ErrRecordExists
		} else {
			return ApiUser{}, err
		}
	}

	return ApiUser{
		UserID:     user.UserID,
		LastName:   user.LastName,
		FirstName:  user.FirstName,
		Email:      user.Email,
		IsVerified: user.IsVerified,
		Role:       string(user.Role),
		UpdatedAt:  user.UpdatedAt,
		LastLogin:  user.LastLogin,
	}, nil
}

func (s *AuthService) generateAccessToken(email, firstName, lastName string, userID uuid.UUID) (string, error) {
	expirationTime := time.Now().Add(s.accessTokenTTL)

	claims := jwt.MapClaims{
		"sub":        userID.String(),
		"first_name": firstName,
		"last_name":  lastName,
		"email":      email,
		"exp":        expirationTime.Unix(),
		"iat":        time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *AuthService) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}

		return s.jwtSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}

		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (s *AuthService) LoginWithRefresh(ctx context.Context, email, password string, refreshTokenTTL time.Duration) (accessToken string, refreshToken string, err error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	if err := security.VerifyPassword(user.PasswordHash, password); err != nil {
		return "", "", ErrInvalidCredentials
	}

	accessToken, err = s.generateAccessToken(user.Email, user.FirstName, user.LastName, user.UserID)
	if err != nil {
		return "", "", err
	}

	token, err := s.queries.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{
		UserID:    user.UserID,
		Token:     uuid.NewString(),
		ExpiresAt: time.Now().Add(refreshTokenTTL),
		CreatedAt: time.Now(),
		Revoked:   false,
	})

	return accessToken, token.Token, nil
}

func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshTokenString string) (string, error) {
	token, err := s.queries.GetRefreshToken(ctx, refreshTokenString)
	if err != nil {
		return "", ErrInvalidToken
	}

	if token.Revoked {
		return "", ErrInvalidToken
	}

	if time.Now().After(token.ExpiresAt) {
		return "", ErrExpiredToken
	}

	user, err := s.queries.GetUserByID(ctx, token.UserID)
	if err != nil {
		return "", errors.Join(utils.ErrUnexpectedError, err)
	}

	accessToken, err := s.generateAccessToken(user.Email, user.FirstName, user.LastName, user.UserID)

	return accessToken, nil
}
