package validator

import (
	"github.com/google/uuid"
)

type User struct {
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Password  string    `json:"password"`
	UserID    uuid.UUID `json:"user_id"`
}

func ValidateUser(v *Validator, user *User) {
	v.Check(VerifyEmail(user.Email), "email", "a valid value must provided")

	v.Check(len(user.FirstName) > 2 && len(user.FirstName) <= 30, "first_name", "must be between 3 to 30 characters long")
	v.Check(len(user.LastName) > 2 && len(user.LastName) <= 30, "last_name", "must be between 3 to 30 characters long")

	v.Check(user.Password != "", "password", "must be provided")
	v.Check(len(user.Password) >= 8, "password", "must be atleast 8 bytes long")
	v.Check(len(user.Password) <= 72, "password", "must not be more than 72 bytes long")
}

func ValidateBasicLogin(v *Validator, user *User) {
	v.Check(VerifyEmail(user.Email), "email", "a valid value must provided")
	v.Check(user.Password != "", "password", "must be provided")
}

func ValidateTokenPlainText(v *Validator, tokenPlainText string) {
	v.Check(tokenPlainText != "", "token", "must be provided")
	v.Check(len(tokenPlainText) == 26, "token", "must be 26 bytes long")
}

func ValidateEmail(v *Validator, email string) {
	v.Check(VerifyEmail(email), "email", "a valid value must be provided")
}

func ValidateResetPassword(v *Validator, token, password string) {
	v.Check(token != "", "token", "must be provided")
	v.Check(len(token) == 26, "token", "must be provided")
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) > 8, "password", "must be atleast 8 bytes long")
}
