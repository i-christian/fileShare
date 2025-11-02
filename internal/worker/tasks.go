package worker

import (
	"fmt"
	"log/slog"
	"sync"
)

// BackgroundTask executes a function fn in a new goroutine.
// It passes the provided logger to fn and recovers from any panics,
// logging the error using the same logger.
func BackgroundTask(wg *sync.WaitGroup, logger *slog.Logger, fn func(l *slog.Logger)) {
	wg.Add(1)

	go func() {
		defer wg.Done()

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
