package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	if err := security.VerifyPassword(user.PasswordHash, password); err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := s.generateAccessToken(&user)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthService) generateAccessToken(user *database.GetUserByEmailRow) (string, error) {
	expirationTime := time.Now().Add(s.accessTokenTTL)

	claims := jwt.MapClaims{
		"sub":        user.UserID.String(),
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"email":      user.Email,
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

	accessToken, err = s.generateAccessToken(&user)
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
		return "", err
	}

	returnedUser := database.GetUserByEmailRow{
		UserID:       user.UserID,
		Email:        user.Email,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
		LastLogin:    user.LastLogin,
	}

	accessToken, err := s.generateAccessToken(&returnedUser)

	return accessToken, nil
}
