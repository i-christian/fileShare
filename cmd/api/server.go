package main

import (
	"context"
	"database/sql"
	"errors"
	"expvar"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/i-christian/fileShare/internal/auth"
	"github.com/i-christian/fileShare/internal/database"
	"github.com/i-christian/fileShare/internal/db"
	"github.com/i-christian/fileShare/internal/files"
	"github.com/i-christian/fileShare/internal/mailer"
	"github.com/i-christian/fileShare/internal/public"
	"github.com/i-christian/fileShare/internal/router"
	"github.com/i-christian/fileShare/internal/user"
	"github.com/i-christian/fileShare/internal/utils"
	"github.com/i-christian/fileShare/internal/worker"
)

func (app *application) serve(dbConn *sql.DB) error {
	mailService, err := mailer.New(
		app.config.mail.host,
		app.config.mail.port,
		app.config.mail.username,
		app.config.mail.password,
		app.config.mail.sender,
	)
	if err != nil {
		return fmt.Errorf("failed to setup mail service: %w", err)
	}

	fileStorage := utils.SetUpFileStorage(app.logger)
	psqlService := database.New(dbConn)
	redisOpt := asynq.RedisClientOpt{
		Addr: utils.GetEnvOrFile("REDIS_ADDR"),
	}
	taskDistributor := worker.NewRedisTaskDistributor(redisOpt)

	publicHandler := public.NewPublicHandler(app.config.env, app.config.version, app.logger)

	authService := auth.NewAuthService(psqlService, app.config.jwtSecret, app.config.jwtTTL, app.logger)
	apiKeyService := auth.NewAPIKeyService(8, app.config.apiKeyPrefix, psqlService, app.logger, &app.wg)
	authHandler := auth.NewAuthHandler(authService, apiKeyService, app.config.refreshTokenTTL, app.logger, taskDistributor)

	userService := user.NewUserService(psqlService, app.logger)
	userHandler := user.NewUserHandler(userService)

	fileService := files.NewFileService(psqlService, fileStorage, app.logger, taskDistributor)
	fileHandler := files.NewFileHandler(app.config.maxUploadSize, fileService, app.logger)

	routeConfig := &router.RoutesConfig{
		Domain:         app.config.domain,
		Rps:            app.config.limiter.rps,
		Burst:          app.config.limiter.burst,
		LimiterEnabled: app.config.limiter.enabled,
	}
	r := router.RegisterRoutes(routeConfig, authHandler, authService, apiKeyService, userHandler, publicHandler, fileHandler)

	publishMetrics(dbConn, app.config.version)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      r,
		IdleTimeout:  time.Minute,
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 40 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	taskProcessor := NewRedisTaskProcessor(redisOpt, fileService, app.logger, mailService)
	go func() {
		app.logger.Info("starting background worker")
		if err := taskProcessor.Start(); err != nil {
			app.logger.Error("failed to start task processor", "error", err)
		}
	}()

	shutdownError := make(chan error)
	go func() {
		app.logger.Info(fmt.Sprintf("server starting on http://%s:%d", app.config.domain, app.config.port), "env", app.config.env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			shutdownError <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-shutdownError:
		return err
	case <-quit:
		app.logger.Info("Shutting down server")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	app.logger.Info("Completing background tasks...")
	taskProcessor.Shutdown()
	app.wg.Wait()

	app.logger.Info("Graceful shutdown complete")

	return nil
}

func publishMetrics(conn *sql.DB, version string) {
	expvar.NewString("version").Set(version)
	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))
	expvar.Publish("database", expvar.Func(func() any {
		return db.Health(conn)
	}))
	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))
}
