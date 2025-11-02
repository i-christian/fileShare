package worker

import (
	"fmt"
	"log/slog"
)

// BackgroundTask executes a function fn in a new goroutine.
// It passes the provided logger to fn and recovers from any panics,
// logging the error using the same logger.
func BackgroundTask(logger *slog.Logger, fn func(l *slog.Logger)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error(
					"panic recovered in background task",
					"error", fmt.Errorf("%v", r),
				)
			}
		}()

		fn(logger)
	}()
}

