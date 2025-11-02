package utils

import (
	"errors"
	"log/slog"
	"net/http"
)

var (
	ErrUnexpectedError = errors.New("A server error occured while processing the request. Our team has been notified")
	ErrRecordExists    = errors.New("The record already exist in the database")
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

// UnauthorisedResponse is a helper to send uauthorised response to client
func UnauthorisedResponse(w http.ResponseWriter, msg string) {
	WriteErrorJSON(w, http.StatusUnauthorized, msg)
}

// FailedValidationResponse returns an error if request body is invalid
func FailedValidationResponse(w http.ResponseWriter, msg map[string]string) {
	WriteErrorJSON(w, http.StatusUnprocessableEntity, msg)
}

func RateLimitExcededResponse(w http.ResponseWriter) {
	WriteErrorJSON(w, http.StatusTooManyRequests, "rate limit exceded")
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
