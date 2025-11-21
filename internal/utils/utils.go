// Package utils provides global helper functions for this web service
package utils

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/i-christian/fileShare/internal/filestore"
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

	return os.Getenv(key)
}

// SetUpFileStorage setups file storage for the application
func SetUpFileStorage(logger *slog.Logger) filestore.FileStorage {
	uploadsDir := GetEnvOrFile("UPLOADS_DIR")
	if uploadsDir == "" {
		logger.Error("UPLOADS_DIR environment variable is not set")
		os.Exit(1)
	}

	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		logger.Error("Failed to create base uploads directory", "path", uploadsDir, "error", err)
		os.Exit(1)
	}

	subDirsToCreate := []string{
		"users",
	}

	for _, subDir := range subDirsToCreate {
		fullPath := filepath.Join(uploadsDir, subDir)
		if err := os.MkdirAll(fullPath, 0o755); err != nil {
			logger.Error("Failed to create subdirectory within uploads", "path", fullPath, "error", err)
			os.Exit(1)
		}
	}

	fileStore, err := filestore.NewDiskStorage(uploadsDir)
	if err != nil {
		logger.Error("Failed to initialize disk storage", "error", err)
		os.Exit(1)
	}

	return fileStore
}

// CleanUpUserFiles deletes a list of files from storage.
func CleanUpUserFiles(fileStore filestore.FileStorage, logger *slog.Logger, paths []string) {
	if len(paths) == 0 {
		logger.Info("User has no files on the system")
	}

	for _, deletePath := range paths {
		if err := fileStore.Delete(deletePath); err != nil {
			logger.Error("failed to delete disk file", "path", deletePath, "error", err)
			continue
		} else {
			logger.Info("file path deleted", "path", deletePath)
		}
	}
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
