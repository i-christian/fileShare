package filestore

import (
	"io"
	"os"
	"path/filepath"
)

// DiskStorage is an implementation of the FileStorage interface that stores files on the local disk.
type DiskStorage struct {
	root *os.Root
}

// NewDiskStorage is a constructor for DiskStorage.
// It initializes a new DiskStorage instance, creating an os.Root rooted
// at the specified `baseDir`. All subsequent file operations performed
// through this DiskStorage instance will be confined within `baseDir`.
func NewDiskStorage(baseDir string) (*DiskStorage, error) {
	root, err := os.OpenRoot(baseDir)
	if err != nil {
		return nil, err
	}
	return &DiskStorage{root: root}, nil
}

// It ensures that the necessary parent directories for the `path` exist in root directory before creating file.
func (s *DiskStorage) ensureParent(inputPath string) error {
	rootPath := s.GetRootPath()
	path := filepath.Join(rootPath, inputPath)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return nil
}

// Save writes the content of the provided io.Reader (`file`) to the
// specified `path` within the DiskStorage's root directory
func (s *DiskStorage) Save(file io.Reader, inputPath string) error {
	err := s.ensureParent(inputPath)
	if err != nil {
		return err
	}

	out, err := s.root.Create(inputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Copy the content from the input reader to the newly created file.
	_, err = io.Copy(out, file)
	return err
}

// Get retrieves the content of the file at the specified `path` from the DiskStorage's root directory.
func (s *DiskStorage) Get(path string) (io.Reader, error) {
	return s.root.Open(path)
}

// GetRootPath retrieves the root path to files directory
func (s *DiskStorage) GetRootPath() string {
	return s.root.Name()
}

// Delete removes the file at the specified `path` from the DiskStorage's root directory.
func (s *DiskStorage) Delete(inputPath string) error {
	rootPath := s.GetRootPath()

	deletePath := filepath.Join(rootPath, inputPath)

	return os.RemoveAll(deletePath)
}
