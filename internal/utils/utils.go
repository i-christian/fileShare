// Package utils provides global helper functions for this web service
package utils

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// ValidateEnvVars checks if required env vars are all set during server startup
func ValidateEnvVars(logger *slog.Logger) {
	requiredVars := []string{"DB_HOST", "DB_PORT", "DB_NAME", "DB_USERNAME", "PORT", "DOMAIN", "JWT_SECRET", "PROJECT_NAME", "GOOSE_DRIVER", "GOOSE_MIGRATION_DIR", "SUPERUSER_EMAIL", "ENV", "UPLOADS_DIR", "PROJECT_NAME", "MAX_UPLOAD_SIZE"}
	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			logger.Error(fmt.Sprintf("Environment variable %s is required", v))
			os.Exit(1)
		}
	}
}

// ToggleEnvOrSecret chooses between secretfile or environment variable
func ToggleEnvOrSecret(fileEnv, envVar string) string {
	var value string
	if filePath := fileEnv; filePath != "" {
		value = GetEnvOrFile(filePath)
	} else if envVal := envVar; envVal != "" {
		value = envVal
	}

	return value
}

// GetEnvOrFile returns the value of the environment variable `key`.
// If `key_FILE` exists, it will read the file at that path and return its contents.
func GetEnvOrFile(key string) string {
	var filePath string
	if strings.Contains(key, "secrets") {
		filePath = key
	}

	if filePath != "" {
		data, err := os.ReadFile(strings.TrimSpace(filePath))
		if err != nil {
			panic("failed to read secret file for " + key + ": " + err.Error())
		}
		return strings.TrimSpace(string(data))
	}

	return strings.TrimSpace(os.Getenv(key))
}

// ReadInt is a helper for query params
func ReadInt(qs map[string][]string, key string, defaultValue int) int {
	s := qs[key]
	if len(s) == 0 {
		return defaultValue
	}

	i, err := strconv.Atoi(s[0])
	if err != nil {
		return defaultValue
	}
	return i
}
