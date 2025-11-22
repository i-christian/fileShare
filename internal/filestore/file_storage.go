package filestore

import "io"

// FileStorage defines the interface for file storage operations.
// Implementations of this interface are responsible for saving, retrieving,
// and deleting files from a persistent storage medium i.e disk or amazon S3 buckets.
type FileStorage interface {
	// Save writes the content of the provided io.Reader to the specified path.
	Save(file io.Reader, path string) (size int64, err error)

	// Get retrieves the content of the file at the specified path.
	// It returns an io.Reader from which the file's content can be read.
	Get(path string) (io.Reader, error)

	// GetRootPath returns the root path of files directory
	GetRootPath() string

	// Delete removes the file at the specified path from storage.
	Delete(path string) error
}
