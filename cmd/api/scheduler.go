package main

import (
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/i-christian/fileShare/internal/worker"
)

// Scheduler defines the interface for scheduling periodic tasks
type Scheduler interface {
	Start() error
	Shutdown()
}

type RedisTaskScheduler struct {
	scheduler *asynq.Scheduler
	logger    *slog.Logger
}

func NewRedisTaskScheduler(redisOpt asynq.RedisClientOpt, logger *slog.Logger) *RedisTaskScheduler {
	scheduler := asynq.NewScheduler(
		redisOpt,
		&asynq.SchedulerOpts{
			EnqueueErrorHandler: func(task *asynq.Task, opts []asynq.Option, err error) {
				logger.Error("failed to enqueue periodic task", "task", task.Type(), "error", err)
			},
		},
	)

	return &RedisTaskScheduler{
		scheduler: scheduler,
		logger:    logger,
	}
}

func (s *RedisTaskScheduler) Start() error {
	if _, err := s.scheduler.Register("0 3 * * *", asynq.NewTask(worker.TaskCleanupSystem, nil)); err != nil {
		return fmt.Errorf("failed to register cleanup task: %w", err)
	}

	s.logger.Info("scheduler started and cron jobs registered")
	return s.scheduler.Run()
}

func (s *RedisTaskScheduler) Shutdown() {
	s.scheduler.Shutdown()
}
