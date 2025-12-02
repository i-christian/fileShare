package files

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/i-christian/fileShare/internal/database"
	"github.com/i-christian/fileShare/internal/utils"
	"github.com/i-christian/fileShare/internal/utils/security"
	"github.com/i-christian/fileShare/internal/validator"
)

type FileHandler struct {
	service       *FileService
	logger        *slog.Logger
	maxUploadSize uint64
}

func NewFileHandler(maxUploadSize uint64, service *FileService, logger *slog.Logger) *FileHandler {
	return &FileHandler{
		service:       service,
		logger:        logger,
		maxUploadSize: maxUploadSize,
	}
}

// Upload handles streaming file uploads
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	user, ok := security.GetUserFromContext(r)
	if !ok || user.IsAnonymous() {
		utils.UnauthorisedResponse(w, utils.ErrAuthRequired.Error())
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, int64(h.maxUploadSize))

	reader, err := r.MultipartReader()
	if err != nil {
		utils.BadRequestResponse(w, errors.New("malformed multipart request"))
		return
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			utils.BadRequestResponse(w, errors.New("error reading multipart body"))
			return
		}

		if part.FormName() == "file" {
			defer part.Close()

			filename := part.FileName()
			if filename == "" {
				utils.BadRequestResponse(w, errors.New("filename is missing"))
				return
			}

			fileStream, contentType, err := validator.ValidateAndPrepareStream(filename, part)
			if err != nil {
				utils.FailedValidationResponse(w, map[string]string{"error": err.Error()})
				utils.WriteServerError(h.logger, "failed to detect file type", err)
				return
			}

			uploadedFile, err := h.service.UploadFile(
				user.UserID,
				fileStream,
				contentType,
				filename,
				int64(h.maxUploadSize),
			)
			if err != nil {
				utils.WriteServerError(h.logger, "failed to upload file", err)
				if errors.Is(err, utils.ErrDuplicateUpload) {
					utils.ServerErrorResponse(w, err.Error())
					return
				}
				utils.ServerErrorResponse(w, "failed to process upload")
				return
			}

			utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
				"message": "File uploaded successfully",
				"file":    uploadedFile,
			}, nil)
			return
		}
	}

	utils.BadRequestResponse(w, errors.New("missing 'file' field in form data"))
}

// ListPublicFiles retrieves public files with pagination validation
func (h *FileHandler) ListPublicFiles(w http.ResponseWriter, r *http.Request) {
	input := validator.Filters{
		Page:     utils.ReadInt(r.URL.Query(), "page", 1),
		PageSize: utils.ReadInt(r.URL.Query(), "page_size", 20),
	}

	v := validator.New()
	if validator.ValidateFilters(v, input); !v.Valid() {
		utils.FailedValidationResponse(w, v.Errors)
		return
	}

	filters := utils.Filters{Page: input.Page, PageSize: input.PageSize}

	files, metadata, err := h.service.ListPublicFiles(r.Context(), filters)
	if err != nil {
		utils.WriteServerError(h.logger, "failed to fetch public files", err)
		utils.ServerErrorResponse(w, "failed to fetch files")
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"metadata": metadata,
		"files":    files,
	}, nil)
	if err != nil {
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
	}
}

// ListMyFiles retrieves user files with pagination validation
func (h *FileHandler) ListMyFiles(w http.ResponseWriter, r *http.Request) {
	user, ok := security.GetUserFromContext(r)
	if !ok || user.IsAnonymous() {
		utils.UnauthorisedResponse(w, utils.ErrAuthRequired.Error())
		return
	}

	input := validator.Filters{
		Page:     utils.ReadInt(r.URL.Query(), "page", 1),
		PageSize: utils.ReadInt(r.URL.Query(), "page_size", 20),
	}

	v := validator.New()
	if validator.ValidateFilters(v, input); !v.Valid() {
		utils.FailedValidationResponse(w, v.Errors)
		return
	}

	filters := utils.Filters{Page: input.Page, PageSize: input.PageSize}

	files, metadata, err := h.service.ListUserFiles(r.Context(), user.UserID, filters)
	if err != nil {
		utils.WriteServerError(h.logger, "failed to fetch user files", err)
		utils.ServerErrorResponse(w, "failed to fetch files")
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"metadata": metadata,
		"files":    files,
	}, nil)
	if err != nil {
		utils.ServerErrorResponse(w, "server error")
	}
}

// GetMetadata retrieves details about a specific file
func (h *FileHandler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	fileID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		utils.BadRequestResponse(w, errors.New("invalid file ID parameter"))
		return
	}

	user, ok := security.GetUserFromContext(r)
	if !ok || user.IsAnonymous() {
		utils.UnauthorisedResponse(w, utils.ErrAuthRequired.Error())
		return
	}

	meta, err := h.service.GetFileMetadata(r.Context(), fileID, user.UserID)
	if err != nil {
		utils.WriteServerError(h.logger, "failed to get file metadata", err)
		if errors.Is(err, utils.ErrRecordNotFound) {
			utils.NotFoundResponse(w)
			return
		}
		utils.ServerErrorResponse(w, "failed to retrieve file")
		return
	}

	isOwner := meta.OwnerID == user.UserID
	if meta.Visibility == database.FileVisibilityPrivate && !isOwner {
		utils.NotPermittedResponse(w)
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, utils.Envelope{"file": meta}, nil)
	if err != nil {
		utils.ServerErrorResponse(w, "server error")
	}
}

// Download streams the file to the client
func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	fileID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		utils.BadRequestResponse(w, errors.New("invalid file ID parameter"))
		return
	}

	user, ok := security.GetUserFromContext(r)
	if !ok {
		utils.UnauthorisedResponse(w, utils.ErrAuthRequired.Error())
		return
	}

	stream, fileInfo, err := h.service.DownloadFile(r.Context(), fileID, user.UserID)
	if err != nil {
		utils.WriteServerError(h.logger, "failed to prepare download", err)
		if errors.Is(err, utils.ErrRecordNotFound) {
			utils.NotFoundResponse(w)
			return
		} else if errors.Is(err, utils.ErrNotPermitted) {
			utils.NotPermittedResponse(w)
			return
		}

		utils.ServerErrorResponse(w, "file unavailable")
		return
	}

	defer stream.Close()

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileInfo.Filename))
	w.Header().Set("Content-Type", fileInfo.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.SizeBytes, 10))

	if _, err := io.Copy(w, stream); err != nil {
		h.logger.Error("connection dropped during download", "error", err)
	}
}

// SetFileVisibility toggles file visibility status
func (h *FileHandler) SetFileVisibility(w http.ResponseWriter, r *http.Request) {
	fileID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		utils.BadRequestResponse(w, errors.New("invalid file ID parameter"))
		return
	}

	user, ok := security.GetUserFromContext(r)
	if !ok || user.IsAnonymous() {
		utils.UnauthorisedResponse(w, utils.ErrAuthRequired.Error())
		return
	}

	var input struct {
		Version    int32  `json:"version"`
		Visibility string `json:"visibility"`
	}

	err = utils.ReadJSON(w, r, &input)
	if err != nil {
		utils.BadRequestResponse(w, err)
		return
	}

	f := validator.FileInfo{
		Version:    input.Version,
		Visibility: input.Visibility,
	}
	v := validator.New()
	if validator.ValidateVisibility(v, f); !v.Valid() {
		utils.FailedValidationResponse(w, v.Errors)
		return
	}

	newVis, err := h.service.SetFileVisibility(r.Context(), fileID, input.Version, database.FileVisibility(input.Visibility))
	if err != nil {
		utils.WriteServerError(h.logger, "failed to change file visibility status", err)
		if errors.Is(err, utils.ErrRecordNotFound) {
			utils.NotFoundResponse(w)
			return
		}

		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": fmt.Sprintf("file visibility has been updated to %s", newVis)}, nil)
	if err != nil {
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
	}
}

// UpdateFileName method updates the file name in the database
func (h *FileHandler) UpdateFileName(w http.ResponseWriter, r *http.Request) {
	fileID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		utils.BadRequestResponse(w, errors.New("invalid file ID parameter"))
		return
	}

	user, ok := security.GetUserFromContext(r)
	if !ok || user.IsAnonymous() {
		utils.UnauthorisedResponse(w, utils.ErrAuthRequired.Error())
		return
	}

	var input struct {
		Version  int32  `json:"version"`
		FileName string `json:"filename"`
	}

	err = utils.ReadJSON(w, r, &input)
	if err != nil {
		utils.BadRequestResponse(w, err)
		return
	}

	f := validator.FileInfo{
		Version:  input.Version,
		Filename: input.FileName,
	}
	v := validator.New()
	if validator.ValidateFileNameChange(v, f); !v.Valid() {
		utils.FailedValidationResponse(w, v.Errors)
		return
	}

	newName, err := h.service.UpdateFileName(r.Context(), fileID, input.FileName, input.Version)
	if err != nil {
		utils.WriteServerError(h.logger, "failed to change filename", err)
		if errors.Is(err, utils.ErrRecordNotFound) {
			utils.NotFoundResponse(w)
			return
		}

		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": fmt.Sprintf("filename has been updated to %s", newName)}, nil)
	if err != nil {
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
	}
}

// Delete handles soft deletion
func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	fileID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		utils.BadRequestResponse(w, errors.New("invalid file ID parameter"))
		return
	}

	user, ok := security.GetUserFromContext(r)
	if !ok || user.IsAnonymous() {
		utils.UnauthorisedResponse(w, utils.ErrAuthRequired.Error())
		return
	}

	var input struct {
		Version int32 `json:"version"`
	}

	err = utils.ReadJSON(w, r, &input)
	if err != nil {
		utils.BadRequestResponse(w, err)
		return
	}

	v := validator.New()
	f := validator.FileInfo{
		Version: input.Version,
	}
	if validator.ValidateDeleteFile(v, f); !v.Valid() {
		utils.FailedValidationResponse(w, v.Errors)
		return
	}

	if err := h.service.DeleteFile(r.Context(), fileID, user.UserID, input.Version); err != nil {
		if errors.Is(err, utils.ErrRecordNotFound) {
			utils.NotFoundResponse(w)
			return
		}
		utils.WriteServerError(h.logger, "failed to delete file", err)
		utils.ServerErrorResponse(w, "failed to delete file")
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "file deleted successfully"}, nil)
	if err != nil {
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
	}
}
