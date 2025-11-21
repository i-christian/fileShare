package utils

import (
	"errors"
	"log/slog"
	"net/http"
)

var (
	ErrUnexpectedError = errors.New("a server error occured while processing the request. Our team has been notified")
	ErrRecordExists    = errors.New("the record already exist in the database")
	ErrRecordNotFound  = errors.New("the record does not exist")
	ErrEditConflict    = errors.New("edit confict")
	ErrAuthRequired    = errors.New("authentication is required to access this resource")
	ErrDuplicateUpload = errors.New("file already exists")
	ErrNotPermitted    = errors.New("you do not have the permission to access this resource")
)

// WriteErrorJSON returns an error in json format to the client
func WriteErrorJSON(w http.ResponseWriter, status int, msg any) {
	WriteJSON(w, status, Envelope{"error": msg}, nil)
}

// BadRequestResponse is a wrapper on WriteErrorJSON function which handles bad request errors to client
func BadRequestResponse(w http.ResponseWriter, err error) {
	WriteErrorJSON(w, http.StatusBadRequest, err.Error())
}

// ServerErrorResponse is a helper function to send internal server error responses to client
func ServerErrorResponse(w http.ResponseWriter, message string) {
	WriteErrorJSON(w, http.StatusInternalServerError, message)
}

// UnauthorisedResponse is a helper to send unauthorised response to client
func UnauthorisedResponse(w http.ResponseWriter, msg string) {
	WriteErrorJSON(w, http.StatusUnauthorized, msg)
}

// InactivateAccountResponse is a helper to send an error response to client if they are using an unverified account to access a protected route.
func InactivateAccountResponse(w http.ResponseWriter) {
	msg := "your user account must be activated to access this resource"
	WriteErrorJSON(w, http.StatusForbidden, msg)
}

// NotPermittedResponse sends a 403 error if client has no permission to access a particular resource.
func NotPermittedResponse(w http.ResponseWriter) {
	message := "your user account doesn't have the necessary permissions to access this resource"
	WriteErrorJSON(w, http.StatusForbidden, message)
}

// NotFoundResponse send a 404 error to client if no record is found
func NotFoundResponse(w http.ResponseWriter) {
	WriteErrorJSON(w, http.StatusNotFound, ErrRecordNotFound.Error())
}

// FailedValidationResponse returns an error if request body is invalid
func FailedValidationResponse(w http.ResponseWriter, msg map[string]string) {
	WriteErrorJSON(w, http.StatusUnprocessableEntity, msg)
}

// RateLimitExcededResponse returns an error if too many requests are made from a single IP
func RateLimitExcededResponse(w http.ResponseWriter) {
	WriteErrorJSON(w, http.StatusTooManyRequests, "rate limit exceded")
}

func EditConflictResponse(w http.ResponseWriter) {
	message := "unable to update the record due to an edit conflict, please try again"
	WriteErrorJSON(w, http.StatusConflict, message)
}

// WriteServerLog logs server errors to the configured slog.logger for the application
func WriteServerLog(logger *slog.Logger, logLevel slog.Level, msg string, err error) {
	switch logLevel {
	case slog.LevelDebug:
		logger.Debug(msg, "details", err.Error())
	case slog.LevelError:
		logger.Error(msg, "details", err.Error())
	case slog.LevelWarn:
		logger.Error(msg, "details", err.Error())
	default:
		logger.Info(msg, "details", err.Error())
	}
}

// WriteServerError is a wraps WriteServerLog function to log errors
func WriteServerError(logger *slog.Logger, msg string, err error) {
	WriteServerLog(logger, slog.LevelError, msg, err)
}
