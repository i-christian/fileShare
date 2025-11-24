// Package files defines the service and handlers used for file manipulation
package files

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/i-christian/fileShare/internal/database"
	"github.com/i-christian/fileShare/internal/filestore"
	"github.com/i-christian/fileShare/internal/utils"
	"github.com/i-christian/fileShare/internal/worker"
)

type FileService struct {
	db              *database.Queries
	store           filestore.FileStorage
	logger          *slog.Logger
	taskDistributor worker.Distributor
}

func NewFileService(db *database.Queries, store filestore.FileStorage, logger *slog.Logger, taskDist worker.Distributor) *FileService {
	return &FileService{
		db:              db,
		store:           store,
		logger:          logger,
		taskDistributor: taskDist,
	}
}

// UploadFile streams the file to storage while calculating the checksum simultaneously.
func (s *FileService) UploadFile(userID uuid.UUID, fileStream io.Reader, contentType string, fileName string, maxUploadSize int64) (database.CreateFileRow, error) {
	uniqueFilename := uuid.New().String() + filepath.Ext(fileName)
	dirPath := filepath.Join("users", userID.String())
	storageKey := filepath.Join(dirPath, uniqueFilename)

	hasher := sha256.New()
	tee := io.TeeReader(fileStream, hasher)

	fileSize, err := s.store.Save(tee, storageKey)
	if err != nil {
		s.logger.Error("failed to save file to storage", "key", storageKey, "error", err)
		_ = s.store.Delete(storageKey)
		return database.CreateFileRow{}, fmt.Errorf("storage error")
	}
	if fileSize > maxUploadSize {
		_ = s.store.Delete(storageKey)
		return database.CreateFileRow{}, fmt.Errorf("file size is too large")
	}

	hashBytes := hasher.Sum(nil)
	checksum := hex.EncodeToString(hashBytes)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	existingFile, _ := s.db.GetFileByChecksum(ctx, database.GetFileByChecksumParams{
		Checksum: checksum,
		UserID:   userID,
	})

	if existingFile.Count > 0 {
		_ = s.store.Delete(storageKey)
		return database.CreateFileRow{}, utils.ErrDuplicateUpload
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

	if strings.HasPrefix(contentType, "image/") {
		taskPayload := &worker.ThumbnailPayload{
			FileID:     fileRec.FileID,
			StorageKey: storageKey,
		}

		opts := []asynq.Option{
			asynq.MaxRetry(3),
			asynq.Queue("default"),
			asynq.Timeout(20 * time.Second),
		}

		err := s.taskDistributor.DistributeGenerateThumbnail(context.Background(), taskPayload, opts...)
		if err != nil {
			utils.WriteServerError(s.logger, "failed to enqueue thumbnail task", err)
		}
	}

	return fileRec, nil
}

// GenerateThumbnail creates a thumbnail for an image file
func (s *FileService) GenerateThumbnail(fileID uuid.UUID, storageKey string) error {
	originalFile, err := s.store.Get(storageKey)
	if err != nil {
		return err
	}
	defer originalFile.Close()

	img, err := imaging.Decode(originalFile)
	if err != nil {
		return err
	}

	thumbnail := imaging.Resize(img, 300, 0, imaging.Lanczos)

	buf := new(bytes.Buffer)
	err = imaging.Encode(buf, thumbnail, imaging.JPEG)
	if err != nil {
		return err
	}

	thumbKey := "thumbnails/" + uuid.New().String() + ".jpg"
	_, err = s.store.Save(buf, thumbKey)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = s.db.UpdateFileThumbnail(ctx, database.UpdateFileThumbnailParams{
		ThumbnailKey: sql.NullString{String: thumbKey, Valid: true},
		FileID:       fileID,
	})
	if err != nil {
		_ = s.store.Delete(thumbKey)
		return err
	}

	return nil
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
func (s *FileService) DownloadFile(ctx context.Context, fileID, userID uuid.UUID) (reader io.ReadCloser, fileInfo database.GetFileInfoRow, err error) {
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
func (s *FileService) SetFileVisibility(ctx context.Context, fileID uuid.UUID, version int32, visibility database.FileVisibility) (string, error) {
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
func (s *FileService) UpdateFileName(ctx context.Context, fileID uuid.UUID, fileName string, version int32) (newName string, err error) {
	newName, err = s.db.UpdateFileName(ctx, database.UpdateFileNameParams{
		Filename: fileName,
		FileID:   fileID,
		Version:  version,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", utils.ErrRecordNotFound
		}
		return "", err

	}

	return newName, nil
}

// DeleteFile performs a soft delete
func (s *FileService) DeleteFile(ctx context.Context, fileID uuid.UUID, userID uuid.UUID, version int32) error {
	delTime := sql.NullTime{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true}

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

// CleanupExpiredSoftDeleted handles hard deletion of files by a cron job
func (s *FileService) CleanupExpiredSoftDeleted(ctx context.Context, limit int32) (deletedFiles int, err error) {
	files, err := s.db.GetExpiredDeletedFiles(ctx, limit)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch expired files: %w", err)
	}

	if len(files) == 0 {
		return 0, nil
	}

	var fileIDs []uuid.UUID
	var storagePaths []string

	for _, f := range files {
		fileIDs = append(fileIDs, f.FileID)
		storagePaths = append(storagePaths, f.StorageKey)
		if f.ThumbnailKey.Valid {
			storagePaths = append(storagePaths, f.ThumbnailKey.String)
		}
	}

	utils.CleanUpFiles(s.store, s.logger, storagePaths)

	if err := s.db.HardDeleteFiles(ctx, fileIDs); err != nil {
		return 0, fmt.Errorf("failed to hard delete file records: %w", err)
	}

	return len(files), nil
}
