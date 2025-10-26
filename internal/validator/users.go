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
}

func ValidateBasicLogin(v *Validator, user *User) {
	v.Check(VerifyEmail(user.Email), "email", "a valid value must provided")
	v.Check(user.Password != "", "password", "must be provided")
}
