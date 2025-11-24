package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/i-christian/fileShare/internal/files"
	"github.com/i-christian/fileShare/internal/jobs"
	"github.com/i-christian/fileShare/internal/mailer"
	"github.com/i-christian/fileShare/internal/worker"
)

// Processor defines the interface for running tasks
type Processor interface {
	Start() error
	Shutdown()
}

type RedisTaskProcessor struct {
	server      *asynq.Server
	fileService *files.FileService
	mailer      *mailer.Mailer
	conn        *sql.DB
	logger      *slog.Logger
}

func NewRedisTaskProcessor(redisOpt asynq.RedisClientOpt, fileService *files.FileService, conn *sql.DB, logger *slog.Logger, mailer *mailer.Mailer) *RedisTaskProcessor {
	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				logger.Error("process task failed",
					"type", task.Type(),
					"payload", string(task.Payload()),
					"error", err,
				)
			}),
		},
	)

	return &RedisTaskProcessor{
		server:      server,
		fileService: fileService,
		logger:      logger,
		mailer:      mailer,
		conn:        conn,
	}
}

func (p *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()

	mux.HandleFunc(worker.TaskGenerateThumbnail, p.ProcessTaskGenerateThumbnail)
	mux.HandleFunc(worker.TaskSendEmail, p.ProcessTaskSendEmail)
	mux.HandleFunc(worker.TaskCleanupSystem, p.ProcessTaskCleanupSystem)

	return p.server.Run(mux)
}

func (p *RedisTaskProcessor) Shutdown() {
	p.server.Shutdown()
}

func (p *RedisTaskProcessor) ProcessTaskGenerateThumbnail(ctx context.Context, task *asynq.Task) error {
	var payload worker.ThumbnailPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", asynq.SkipRetry)
	}

	p.logger.Info("processing thumbnail task", "file_id", payload.FileID)

	err := p.fileService.GenerateThumbnail(payload.FileID, payload.StorageKey)
	if err != nil {
		return fmt.Errorf("failed to generate thumbnail: %w", err)
	}

	p.logger.Info("processed thumbnail task successfully", "file_id", payload.FileID)
	return nil
}

func (p *RedisTaskProcessor) ProcessTaskSendEmail(ctx context.Context, task *asynq.Task) error {
	var payload worker.EmailPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", asynq.SkipRetry)
	}

	err := p.mailer.Send(payload.Recipient, payload.TemplateFile, payload.Data)
	if err != nil {
		p.logger.Error("failed to send email", "recipient", payload.UserID, "error", err)
		return err
	}

	p.logger.Info("processed email task successfully", "recipient", payload.UserID)
	return nil
}

func (p *RedisTaskProcessor) ProcessTaskCleanupSystem(ctx context.Context, task *asynq.Task) error {
	p.logger.Info("starting system cleanup task")

	var limit int32 = 100
	deletedFileCount, err := p.fileService.CleanupExpiredSoftDeleted(ctx, limit)
	if err != nil {
		p.logger.Error("failed to cleanup files", "error", err)
	}

	expiredCounts, err := jobs.CleanUpExpired(ctx, p.conn)
	if err != nil {
		p.logger.Error("failed to cleanup tokens", "error", err)
	}

	p.logger.Info("system cleanup task finished", "apiKeys", expiredCounts.APIKeysDeleted, "actionTokens", expiredCounts.ActionTokensDeleted, "refreshTokens", expiredCounts.RefreshTokensDeleted, "deleted files", deletedFileCount)
	return nil
}
