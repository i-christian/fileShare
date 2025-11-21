// Package files defines the service and handlers used for file manipulation
package files

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/i-christian/fileShare/internal/database"
	"github.com/i-christian/fileShare/internal/filestore"
	"github.com/i-christian/fileShare/internal/utils"
	"github.com/i-christian/fileShare/internal/utils/security"
)

type FileService struct {
	db     *database.Queries
	store  filestore.FileStorage
	logger *slog.Logger
}

func NewFileService(db *database.Queries, store filestore.FileStorage, logger *slog.Logger) *FileService {
	return &FileService{
		db:     db,
		store:  store,
		logger: logger,
	}
}

// UploadFile handles the logic of saving the binary data and the metadata
func (s *FileService) UploadFile(ctx context.Context, userID uuid.UUID, file multipart.File, fileSize int64, contentType string, fileName string) (database.CreateFileRow, error) {
	checksum, err := security.CalculateChecksum(file)
	if err != nil {
		return database.CreateFileRow{}, err
	}

	existingFile, _ := s.db.GetFileByChecksum(ctx, database.GetFileByChecksumParams{
		Checksum: checksum,
		UserID:   userID,
	})

	if existingFile.Count > 0 {
		return database.CreateFileRow{}, utils.ErrDuplicateUpload
	} else {
		uniqueFilename := uuid.New().String() + filepath.Ext(fileName)
		dirPath := filepath.Join("users", userID.String())
		storageKey := filepath.Join(dirPath, uniqueFilename)

		if err := s.store.Save(file, storageKey); err != nil {
			s.logger.Error("failed to save file to storage", "key", storageKey, "error", err)
			return database.CreateFileRow{}, fmt.Errorf("storage error")
		}

		params := database.CreateFileParams{
			UserID:     userID,
			Filename:   fileName,
			StorageKey: storageKey,
			MimeType:   contentType,
			SizeBytes:  fileSize,
			Checksum:   checksum,
		}

		fileRec, err := s.db.CreateFile(ctx, params)
		if err != nil {
			_ = s.store.Delete(storageKey)
			return database.CreateFileRow{}, fmt.Errorf("database error: %w", err)
		}

		return fileRec, nil
	}
}

// GetFileMetadata retrieves file info ensuring the user owns it
func (s *FileService) GetFileMetadata(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) (database.GetFileInfoRow, error) {
	file, err := s.db.GetFileInfo(ctx, fileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.GetFileInfoRow{}, utils.ErrRecordNotFound
		}
		return database.GetFileInfoRow{}, err
	}

	if file.OwnerID != userID {
		return database.GetFileInfoRow{}, utils.ErrAuthRequired
	}

	return file, nil
}

// DownloadFile returns the file stream
func (s *FileService) DownloadFile(ctx context.Context, fileID, userID uuid.UUID) (reader io.Reader, fileInfo database.GetFileInfoRow, err error) {
	fileInfo, err = s.db.GetFileInfo(ctx, fileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, database.GetFileInfoRow{}, utils.ErrRecordNotFound
		}
		return nil, database.GetFileInfoRow{}, err
	}

	isOwner := userID != uuid.Nil && fileInfo.OwnerID == userID
	if !isOwner && fileInfo.Visibility == database.FileVisibilityPrivate {
		return nil, database.GetFileInfoRow{}, utils.ErrNotPermitted
	}

	stream, err := s.store.Get(fileInfo.StorageKey)
	if err != nil {
		utils.WriteServerError(s.logger, fmt.Sprintf("file found in database but missing in storage key=%s", fileInfo.StorageKey), err)

		return nil, database.GetFileInfoRow{}, errors.New("file content missing")
	}

	return stream, fileInfo, nil
}

// ListUserFiles returns a list of files for the user
func (s *FileService) ListUserFiles(ctx context.Context, userID uuid.UUID, filters utils.Filters) ([]database.ListUserFilesRow, utils.Metadata, error) {
	limit := filters.PageSize
	offset := (filters.Page - 1) * filters.PageSize

	count, err := s.db.CountUserFiles(ctx, userID)
	if err != nil {
		return []database.ListUserFilesRow{}, utils.Metadata{}, err
	}

	files, err := s.db.ListUserFiles(ctx, database.ListUserFilesParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []database.ListUserFilesRow{}, utils.Metadata{}, utils.ErrRecordNotFound
		}
		return []database.ListUserFilesRow{}, utils.Metadata{}, err
	}

	meta := utils.CalculateMetadata(int(count), filters.Page, filters.PageSize)

	fmt.Println(files)
	return files, meta, nil
}

// ListPublicFiles returns a list of files
func (s *FileService) ListPublicFiles(ctx context.Context, filters utils.Filters) ([]database.ListPublicFilesRow, utils.Metadata, error) {
	limit := filters.PageSize
	offset := (filters.Page - 1) * filters.PageSize

	count, err := s.db.CountPublicFiles(ctx)
	if err != nil {
		return []database.ListPublicFilesRow{}, utils.Metadata{}, err
	}
	files, err := s.db.ListPublicFiles(ctx, database.ListPublicFilesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []database.ListPublicFilesRow{}, utils.Metadata{}, utils.ErrRecordNotFound
		}
		return []database.ListPublicFilesRow{}, utils.Metadata{}, err
	}

	meta := utils.CalculateMetadata(int(count), filters.Page, filters.PageSize)

	return files, meta, nil
}

// SetFileVisibility toggles file visibility status by file owner
func (s *FileService) SetFileVisibility(ctx context.Context, userID, fileID uuid.UUID, version int32, visibility database.FileVisibility) (string, error) {
	newVisibility, err := s.db.SetFileVisibility(ctx, database.SetFileVisibilityParams{
		Visibility: visibility,
		FileID:     fileID,
		Version:    version,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", utils.ErrRecordNotFound
		}
		return "", err
	}

	return string(newVisibility), nil
}

// UpdateFileName method updates a file name
func (s *FileService) UpdateFileName(ctx context.Context, fileID uuid.UUID, fileName string, version int32) error {
	err := s.db.UpdateFileName(ctx, database.UpdateFileNameParams{
		Filename: fileName,
		FileID:   fileID,
		Version:  version,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.ErrRecordNotFound
		}
		return err

	}

	return nil
}

// DeleteFile performs a soft delete
func (s *FileService) DeleteFile(ctx context.Context, fileID uuid.UUID, userID uuid.UUID, version int32) error {
	delTime := sql.NullTime{Time: time.Now().Add(24 * 30 * time.Hour), Valid: true}

	err := s.db.DeleteFile(ctx, database.DeleteFileParams{
		DeletedAt: delTime,
		FileID:    fileID,
		Version:   version,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.ErrRecordNotFound
		}
		return err

	}

	return nil
}
