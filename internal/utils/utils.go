// Package utils provides global helper functions for this web service
package utils

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/i-christian/fileShare/internal/filestore"
)

// ValidateEnvVars checks if required env vars are all set during server startup
func ValidateEnvVars(logger *slog.Logger) {
	requiredVars := []string{"DB_HOST", "DB_PORT", "DB_NAME", "DB_USERNAME", "PORT", "DOMAIN", "JWT_SECRET", "PROJECT_NAME", "GOOSE_DRIVER", "GOOSE_MIGRATION_DIR", "SUPERUSER_EMAIL", "ENV", "UPLOADS_DIR", "PROJECT_NAME"}
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

// IsValidFileType checks if the file is safe to upload.
// It blocks executables and scripts but allows general content.
func IsValidFileType(file io.ReadSeeker, filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	blockedExtensions := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".bat": true, ".cmd": true,
		".sh": true, ".php": true, ".pl": true, ".cgi": true, ".jar": true,
		".vbs": true, ".powershell": true, ".js": true,
	}

	if blockedExtensions[ext] {
		return "", fmt.Errorf("file extension '%s' is not allowed for security reasons", ext)
	}

	fileHeader := make([]byte, 512)
	if _, err := file.Read(fileHeader); err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file header: %w", err)
	}

	// Reset file pointer to start
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to reset file pointer: %w", err)
	}

	contentType := http.DetectContentType(fileHeader)

	blockedMimes := map[string]bool{
		"application/x-dosexec":   true,
		"application/x-sh":        true,
		"application/x-httpd-php": true,
		"application/javascript":  true,
	}

	if blockedMimes[contentType] {
		return "", fmt.Errorf("detected blocked content type: %s", contentType)
	}

	return contentType, nil
}
