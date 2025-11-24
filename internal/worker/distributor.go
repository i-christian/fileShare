package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const (
	TaskGenerateThumbnail = "task:image:generate_thumbnail"
	TaskSendEmail         = "task:email:send"
	TaskCleanupSystem     = "task:system:cleaup_expired"
)

type ThumbnailPayload struct {
	FileID     uuid.UUID `json:"file_id"`
	StorageKey string    `json:"storage_key"`
}

type EmailPayload struct {
	TemplateFile string `json:"template_file"`
	UserID       uuid.UUID
	Recipient    string         `json:"recipient"`
	Data         map[string]any `json:"data"`
}

type CleanupPayload struct{}

// Distributor defines how to send tasks to the queue
type Distributor interface {
	DistributeGenerateThumbnail(ctx context.Context, payload *ThumbnailPayload, opts ...asynq.Option) error
	DistributeSendEmail(ctx context.Context, payload *EmailPayload, opts ...asynq.Option) error
}

// RedisTaskDistributor implements Distributor
type RedisTaskDistributor struct {
	client *asynq.Client
}

// NewRedisTaskDistributor creates a new task sender
func NewRedisTaskDistributor(redisOpt asynq.RedisClientOpt) *RedisTaskDistributor {
	client := asynq.NewClient(redisOpt)
	return &RedisTaskDistributor{
		client: client,
	}
}

func (d *RedisTaskDistributor) DistributeGenerateThumbnail(ctx context.Context, payload *ThumbnailPayload, opts ...asynq.Option) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal task payload: %w", err)
	}

	task := asynq.NewTask(TaskGenerateThumbnail, jsonPayload, opts...)

	info, err := d.client.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	slog.Info("enqueued task",
		"type", task.Type(),
		"queue", info.Queue,
		"max_retry", info.MaxRetry,
	)
	return nil
}

func (d *RedisTaskDistributor) DistributeSendEmail(ctx context.Context, payload *EmailPayload, opts ...asynq.Option) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	if len(opts) == 0 {
		opts = []asynq.Option{
			asynq.MaxRetry(5),
			asynq.Timeout(10 * time.Second),
		}
	}

	task := asynq.NewTask(TaskSendEmail, jsonPayload, opts...)

	info, err := d.client.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to enqueue email task: %w", err)
	}

	slog.Info("enqueued email task", "queue", info.Queue, "recipient", payload.UserID)
	return nil
}
