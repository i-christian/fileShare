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

// Upload handles multipart/form-data file uploads
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	user, ok := security.GetUserFromContext(r)
	if !ok && !user.IsAnonymous() {
		utils.UnauthorisedResponse(w, utils.ErrAuthRequired.Error())
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, int64(h.maxUploadSize))

	if err := r.ParseMultipartForm(int64(h.maxUploadSize)); err != nil {
		utils.BadRequestResponse(w, errors.New("file too big or malformed body"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.BadRequestResponse(w, errors.New("missing 'file' field"))
		return
	}
	defer file.Close()

	contentType, err := validator.IsValidFileType(file, header.Filename)
	if err != nil {
		v := validator.New()
		v.AddError("file", err.Error())
		utils.FailedValidationResponse(w, v.Errors)
		return
	}

	uploadPayload := &validator.FileUpload{
		Filename:      header.Filename,
		MaxUploadSize: int64(h.maxUploadSize),
		UploadSize:    header.Size,
	}

	v := validator.New()
	if validator.ValidateFileUpload(v, uploadPayload); !v.Valid() {
		utils.FailedValidationResponse(w, v.Errors)
		return
	}

	uploadedFile, err := h.service.UploadFile(r.Context(), user.UserID, file, header.Size, contentType, header.Filename)
	if err != nil {
		utils.WriteServerError(h.logger, "failed to upload file", err)
		if errors.Is(err, utils.ErrDuplicateUpload) {
			utils.ServerErrorResponse(w, err.Error())
			return
		}
		utils.ServerErrorResponse(w, "failed to process upload")
		return
	}

	err = utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"message": "File uploaded successfully",
		"file":    uploadedFile,
	}, nil)
	if err != nil {
		utils.WriteServerError(h.logger, "failed to write response", err)
		utils.ServerErrorResponse(w, "server error")
	}
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
		utils.ServerErrorResponse(w, "server error")
	}
}

// ListMyFiles retrieves user files with pagination validation
func (h *FileHandler) ListMyFiles(w http.ResponseWriter, r *http.Request) {
	user, ok := security.GetUserFromContext(r)
	if !ok && !user.IsAnonymous() {
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
	if !ok && !user.IsAnonymous() {
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

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileInfo.Filename))
	w.Header().Set("Content-Type", fileInfo.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.SizeBytes, 10))

	if _, err := io.Copy(w, stream); err != nil {
		h.logger.Error("connection dropped during download", "error", err)
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
	if !ok && !user.IsAnonymous() {
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
	if validator.ValidateDeleteFile(v, input.Version); !v.Valid() {
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
		utils.ServerErrorResponse(w, "server error")
	}
}
