package validator

import (
	"bytes"
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

// ValidateAndPrepareStream checks the file extension and MIME type.
func ValidateAndPrepareStream(filename string, stream io.Reader) (fileStream io.Reader, contentType string, err error) {
	ext := strings.ToLower(filepath.Ext(filename))
	blockedExtensions := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".bat": true, ".cmd": true,
		".sh": true, ".php": true, ".pl": true, ".cgi": true, ".jar": true,
		".vbs": true, ".powershell": true, ".js": true,
	}
	if blockedExtensions[ext] {
		return nil, "", fmt.Errorf("file extension '%s' is not allowed", ext)
	}

	header := make([]byte, 512)
	n, err := io.ReadFull(stream, header)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, "", fmt.Errorf("failed to read file header: %w", err)
	}

	contentType = http.DetectContentType(header[:n])
	blockedMimes := map[string]bool{
		"application/x-dosexec":   true,
		"application/x-sh":        true,
		"application/x-httpd-php": true,
		"application/javascript":  true,
	}
	if blockedMimes[contentType] {
		return nil, "", fmt.Errorf("detected blocked content type: %s", contentType)
	}

	fileStream = io.MultiReader(bytes.NewReader(header[:n]), stream)

	return fileStream, contentType, nil
}
