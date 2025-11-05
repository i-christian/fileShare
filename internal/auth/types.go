package auth

import (
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("token has expired")
	ErrEmailInUse         = errors.New("email already in use")
	ErrInvalidClaims      = errors.New("token contains invalid claims")
)

type ApiUser struct {
	LastLogin       sql.NullTime `json:"last_login"`
	LastName        string       `json:"last_name"`
	FirstName       string       `json:"first_name"`
	Email           string       `json:"email"`
	Role            string       `json:"role"`
	ActivationToken string       `json:"-"`
	UserID          uuid.UUID    `json:"user_id"`
	IsVerified      bool         `json:"is_verified"`
}
