package validator

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

// FileUpload represents the metadata we validate before saving
type FileUpload struct {
	Filename      string
	UploadSize    int64
	MaxUploadSize int64
}

// Filters represents pagination inputs
type Filters struct {
	Page         int
	PageSize     int
	Sort         string
	SortSafelist []string
}

func ValidateFileUpload(v *Validator, file *FileUpload) {
	v.Check(file.UploadSize <= file.MaxUploadSize, "file", fmt.Sprintf("must not exceed %d", file.MaxUploadSize))
	v.Check(file.Filename != "", "file", "must have a filename")
	v.Check(len(file.Filename) <= 255, "file", "filename must be less than 255 characters")
}

func ValidateFilters(v *Validator, f Filters) {
	v.Check(f.Page > 0, "page", "must be greater than zero")
	v.Check(f.Page <= 10_000, "page", "must be a maximum of 10,000")
	v.Check(f.PageSize > 0, "page_size", "must be greater than zero")
	v.Check(f.PageSize <= 100, "page_size", "must be a maximum of 100")
}

func ValidateDeleteFile(v *Validator, version int32) {
	v.Check(version > 0, "version", "must be greater than zero")
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
