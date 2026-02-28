package filestore

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/i-christian/fileShare/internal/utils"
)

// DiskStorage is an implementation of the FileStorage interface that stores files on the local disk.
type DiskStorage struct {
	root   *os.Root
	logger *slog.Logger
}

// NewDiskStorage is a constructor for DiskStorage.
// It initializes a new DiskStorage instance, creating an os.Root rooted
// at the specified `baseDir`. All subsequent file operations performed
// through this DiskStorage instance will be confined within `baseDir`.
func NewDiskStorage(baseDir string, logger *slog.Logger) (*DiskStorage, error) {
	root, err := os.OpenRoot(baseDir)
	if err != nil {
		return nil, err
	}
	return &DiskStorage{root: root, logger: logger}, nil
}

// It ensures that the necessary parent directories for the `path` exist in root directory before creating file.
func (s *DiskStorage) ensureParent(inputPath string) error {
	rootPath := s.getRootPath()
	path := filepath.Join(rootPath, inputPath)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return nil
}

// Save writes the content of the provided io.Reader (`file`) to the
// specified `path` within the DiskStorage's root directory
func (s *DiskStorage) Save(ctx context.Context, file io.Reader, inputPath string) (size int64, err error) {
	err = s.ensureParent(inputPath)
	if err != nil {
		return 0, err
	}

	out, err := s.root.Create(inputPath)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	size, err = io.Copy(out, file)
	return size, err
}

// Get retrieves the content of the file at the specified `path` from the DiskStorage's root directory.
func (s *DiskStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	return s.root.Open(path)
}

// getRootPath retrieves the root path to files directory
func (s *DiskStorage) getRootPath() string {
	return s.root.Name()
}

// Delete removes the file at the specified `path` from the DiskStorage's root directory.
func (s *DiskStorage) Delete(ctx context.Context, paths []string) (successCount, failureCount int, err error) {
	if len(paths) == 0 {
		return 0, 0, utils.ErrFilesNotFound
	}

	for idx := range paths {
		err := s.root.RemoveAll(paths[idx])
		if err != nil {
			failureCount += 1

			utils.WriteServerError(s.logger, "failed to delete disk file", err)
			continue

		}

		successCount += 1
	}

	return successCount, failureCount, nil
}
