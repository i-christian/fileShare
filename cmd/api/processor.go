package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/i-christian/fileShare/internal/files"
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
	logger      *slog.Logger
}

func NewRedisTaskProcessor(redisOpt asynq.RedisClientOpt, fileService *files.FileService, logger *slog.Logger, mailer *mailer.Mailer) *RedisTaskProcessor {
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
	}
}

func (p *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()

	mux.HandleFunc(worker.TaskGenerateThumbnail, p.ProcessTaskGenerateThumbnail)
	mux.HandleFunc(worker.TaskSendEmail, p.ProcessTaskSendEmail)

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
