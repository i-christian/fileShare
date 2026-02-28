// Package filestore provides a unified abstraction for file storage operations.
package filestore

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/i-christian/fileShare/internal/utils"
)

// StorageType defines storage types supported by application
type StorageType string

const (
	StorageS3   StorageType = "cloud"
	StorageDisk StorageType = "local"
)

// FileStorage defines the interface for file storage operations.
// Implementations of this interface are responsible for saving, retrieving,
// and deleting files from a persistent storage medium i.e disk or amazon S3 buckets.
type FileStorage interface {
	// Save writes the content of the provided io.Reader to the specified path.
	Save(ctx context.Context, file io.Reader, path string) (size int64, err error)

	// Get retrieves the content of the file at the specified path.
	// It returns an io.Reader from which the file's content can be read.
	Get(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete removes files from storage.
	Delete(ctx context.Context, paths []string) (successCount, failureCount int, err error)
}

// SetUpFileStorage initializes the storage provider based on env config
func SetUpFileStorage(logger *slog.Logger) FileStorage {
	storageType := StorageType(utils.GetEnvOrFile("STORAGE_TYPE"))
	var store FileStorage
	var err error

	switch storageType {
	case StorageS3:
		accessKey := utils.GetEnvOrFile("S3_ACCESS_KEY")
		secretKey := utils.GetEnvOrFile("S3_SECRET_KEY")
		endpoint := utils.GetEnvOrFile("S3_ENDPOINT")
		region := utils.GetEnvOrFile("S3_REGION")
		bucket := utils.GetEnvOrFile("S3_BUCKET")

		store, err = NewS3Storage(accessKey, secretKey, endpoint, region, bucket)
		if err != nil {
			utils.WriteServerError(logger, "failed to initilise S3 storage", err)
			os.Exit(1)
		}

		logger.Info("Initialised S3 File storage", "bucket", bucket)

	case StorageDisk:
		uploadsDir := utils.GetEnvOrFile("UPLOADS_DIR")
		if uploadsDir == "" {
			uploadsDir = "./data/uploads" // Dev
		}

		if err = os.MkdirAll(uploadsDir, 0o755); err != nil {
			utils.WriteServerError(logger, "failed to create base uploads directory", err)
			os.Exit(1)
		} else {
			logger.Info("Initialised disk upload directory", "path", uploadsDir)
		}

		subDirsToCreate := []string{
			"users",
		}

		for _, subDir := range subDirsToCreate {
			fullPath := filepath.Join(uploadsDir, subDir)
			if err := os.MkdirAll(fullPath, 0o755); err != nil {
				utils.WriteServerError(logger, "failed to create subdirectory within uploads", err)
				os.Exit(1)
			}
		}

		store, err = NewDiskStorage(uploadsDir, logger)
		if err != nil {
			utils.WriteServerError(logger, "failed to initialise disk storage", err)
			os.Exit(1)
		} else {
			logger.Info("Initialised disk storage", "parent directory", uploadsDir)
		}

	}

	return store
}
