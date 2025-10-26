package validator

import (
	"net/mail"
	"slices"
	"time"
)

// VerifyEmail checks if the email is valid
func VerifyEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

//IsValidTimeFormat checks if the given timeString has the format we expect
func IsValidTimeFormat(layout string, timeString string) bool {
	_, err := time.Parse(layout, timeString)
	return err == nil
}

// Validator contains a map of validation errors
type Validator struct {
	Errors map[string]string
}

// New is a helper which creates a new validator instance with an empty errors map
func New() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

// Valid returns true if the errors maps does not contain any entries
func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// AddError adds an error message to the map (so long as no entry already exists for the given key)
func (v *Validator) AddError(key, message string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

// Check adds an error message to the map only if a validation check is not 'ok'
func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(key, message)
	}
}

// PermittedValue is a generic function which returns true if a specific value is in a list of permitted values
func PermittedValue[T comparable](value T, permittedValues ...T) bool {
	return slices.Contains(permittedValues, value)
}

// Unique generic function returns true if all values in a slice are unique
func Unique[T comparable](values []T) bool {
	uniqueValues := make(map[T]bool)

	for _, value := range values {
		uniqueValues[value] = true
	}

	return len(values) == len(uniqueValues)
}
