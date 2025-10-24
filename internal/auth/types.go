package auth

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("token has expired")
	ErrEmailInUse         = errors.New("email already in use")
)

type ApiUser struct {
	UserID     uuid.UUID    `json:"user_id"`
	LastName   string       `json:"last_name"`
	FirstName  string       `json:"first_name"`
	Email      string       `json:"email"`
	IsVerified bool         `json:"is_verified"`
	Role       string       `json:"role"`
	UpdatedAt  time.Time    `json:"updated_at"`
	LastLogin  sql.NullTime `json:"last_login"`
}
