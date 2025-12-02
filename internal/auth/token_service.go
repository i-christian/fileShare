package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/i-christian/fileShare/internal/database"
	"github.com/i-christian/fileShare/internal/utils"
	"github.com/i-christian/fileShare/internal/utils/security"
	"github.com/i-christian/fileShare/internal/validator"
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

	plainText, hashByte := security.GenerateStringAndHash()

	err = s.queries.CreateActionToken(ctx, database.CreateActionTokenParams{
		UserID:    user.UserID,
		Purpose:   database.TokenPurposeEmailVerification,
		TokenHash: hashByte,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		return ApiUser{}, err
	}

	return ApiUser{
		UserID:          user.UserID,
		LastName:        user.LastName,
		FirstName:       user.FirstName,
		Email:           user.Email,
		IsVerified:      user.IsVerified,
		ActivationToken: plainText,
		Role:            string(user.Role),
		LastLogin:       user.LastLogin,
	}, nil
}

func (s *AuthService) generateAccessToken(email, firstName, lastName string, userID uuid.UUID, role string, isVerified bool) (string, error) {
	expirationTime := time.Now().Add(s.accessTokenTTL)

	claims := jwt.MapClaims{
		"sub":         userID.String(),
		"first_name":  firstName,
		"last_name":   lastName,
		"email":       email,
		"role":        role,
		"is_verified": strconv.FormatBool(isVerified),
		"exp":         expirationTime.Unix(),
		"iat":         time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken parses a JWT string, validates it, and returns a ContextUser.
func (s *AuthService) ValidateToken(tokenString string) (*security.ContextUser, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
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
		return newUserFromClaims(claims)
	}

	return nil, ErrInvalidToken
}

// newUserFromClaims is a helper to parse claims into a ContextUser.
func newUserFromClaims(claims jwt.MapClaims) (*security.ContextUser, error) {
	getStringClaim := func(key string) (string, error) {
		val, ok := claims[key].(string)
		if !ok || val == "" {
			return "", ErrInvalidClaims
		}
		return val, nil
	}

	userIDStr, err := getStringClaim("sub")
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, ErrInvalidClaims
	}

	firstName, err := getStringClaim("first_name")
	if err != nil {
		return nil, err
	}

	lastName, err := getStringClaim("last_name")
	if err != nil {
		return nil, err
	}

	email, err := getStringClaim("email")
	if err != nil {
		return nil, err
	}

	role, err := getStringClaim("role")
	if err != nil {
		return nil, err
	}

	verifiedStr, err := getStringClaim("is_verified")
	if err != nil {
		return nil, err
	}

	verifiedBool, err := strconv.ParseBool(verifiedStr)
	if err != nil {
		return nil, ErrInvalidClaims
	}

	user := &security.ContextUser{
		FirstName:   firstName,
		LastName:    lastName,
		Email:       email,
		Role:        role,
		UserID:      userID,
		IsActivated: verifiedBool,
	}

	return user, nil
}

func (s *AuthService) LoginWithRefresh(ctx context.Context, email, password string, refreshTokenTTL time.Duration) (accessToken string, refreshToken string, err error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	if err := security.VerifyPassword(user.PasswordHash, password); err != nil {
		return "", "", ErrInvalidCredentials
	}

	accessToken, err = s.generateAccessToken(user.Email, user.FirstName, user.LastName, user.UserID, string(user.Role), user.IsVerified)
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
	if err != nil {
		return "", "", utils.ErrUnexpectedError
	}

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

	accessToken, err := s.generateAccessToken(user.Email, user.FirstName, user.LastName, user.UserID, string(user.Role), user.IsVerified)
	if err != nil {
		return "", err
	}

	return accessToken, nil
}

func (s *AuthService) SendPasswordResetLink(ctx context.Context, userID uuid.UUID) (resetToken string, err error) {
	resetToken, hashByte := security.GenerateStringAndHash()

	err = s.queries.CreateActionToken(ctx, database.CreateActionTokenParams{
		UserID:    userID,
		Purpose:   database.TokenPurposePasswordReset,
		TokenHash: hashByte,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	})
	if err != nil {
		return "", err
	}

	return resetToken, nil
}

func (s *AuthService) VerifyPasswordReset(ctx context.Context, userID uuid.UUID, newPassword string, resetToken string, v *validator.Validator) (status bool, err error) {
	tokenHash := sha256.Sum256([]byte(resetToken))

	user, err := s.queries.GetActionTokenForUser(ctx, database.GetActionTokenForUserParams{
		TokenHash: tokenHash[:],
		Purpose:   database.TokenPurposePasswordReset,
		UserID:    userID,
		ExpiresAt: time.Now(),
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			v.AddError("token", "invalid or expired reset token")
			return false, utils.ErrRecordNotFound
		default:
			return false, err
		}
	}

	passwordHash, err := security.HashPassword(newPassword)
	if err != nil {
		return false, err
	}

	err = s.queries.ChangePassword(ctx, database.ChangePasswordParams{
		PasswordHash: passwordHash,
		Version:      user.Version,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, utils.ErrEditConflict
		}
		return false, err
	}

	return true, nil
}
