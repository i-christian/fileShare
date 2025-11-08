package main

import (
	"context"
	"database/sql"
	"encoding/hex"
	"expvar"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/i-christian/fileShare/internal/auth"
	"github.com/i-christian/fileShare/internal/database"
	"github.com/i-christian/fileShare/internal/db"
	"github.com/i-christian/fileShare/internal/mailer"
	"github.com/i-christian/fileShare/internal/public"
	"github.com/i-christian/fileShare/internal/router"
	"github.com/i-christian/fileShare/internal/user"
	"github.com/i-christian/fileShare/internal/utils"
	"github.com/i-christian/fileShare/internal/utils/security"
	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	logger          *slog.Logger
	domain          string
	jwtSecret       string
	apiKeyPrefix    string
	environment     string
	version         string
	port            int
	jwtTTL          time.Duration
	refreshTokenTTL time.Duration
}

func main() {
	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)
	var wg sync.WaitGroup

	var logger *slog.Logger
	if utils.GetEnvOrFile("ENV") == "testing" {
		logger = slog.New(slog.DiscardHandler)
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true}))
	}

	utils.ValidateEnvVars(logger)
	_ = utils.SetUpFileStorage(logger)

	jwtSecret, err := hex.DecodeString(utils.GetEnvOrFile("JWT_SECRET"))
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	conn, err := db.InitialiseDB(utils.GetEnvOrFile("GOOSE_DRIVER"))
	if err != nil {
		logger.Error("failed to initialise database", "error message", err.Error())
		os.Exit(1)
	}

	// Set up database in-app database migrations
	func() {
		var err error
		for i := 0; i < 10; i++ {
			log.Printf("Trying to run database migration: %d\n", i)
			err = db.SetUpMigration(conn)
			if err == nil {
				break
			}
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			logger.Error("failed to run migrations", "error message", err.Error())
			os.Exit(1)
		}
	}()

	port, _ := strconv.Atoi(utils.GetEnvOrFile("PORT"))
	apiKeyPrefix := security.ShortProjectPrefix(utils.GetEnvOrFile("PROJECT_NAME"))
	domain := utils.GetEnvOrFile("DOMAIN")
	env := utils.GetEnvOrFile("ENV")
	version := utils.GetEnvOrFile("VERSION")
	config := &Config{
		port:            port,
		domain:          domain,
		jwtSecret:       string(jwtSecret),
		environment:     env,
		version:         version,
		jwtTTL:          15 * time.Minute,
		apiKeyPrefix:    apiKeyPrefix,
		refreshTokenTTL: 7 * 24 * time.Hour,
		logger:          logger,
	}

	routeConfig := &router.RoutesConfig{
		Domain: config.domain,
	}
	flag.Float64Var(&routeConfig.Rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&routeConfig.Burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&routeConfig.LimiterEnabled, "limiter-enabled", true, "Enable rate limiter")

	flag.Parse()

	mailHost := utils.GetEnvOrFile("MAILTRAP_SMTP_HOST")
	mailPort, _ := strconv.Atoi(utils.GetEnvOrFile("MAILTRAP_SMTP_PORT"))
	mailUserName := utils.GetEnvOrFile("MAILTRAP_USER")
	mailPassword := utils.GetEnvOrFile("MAILTRAP_PASSWORD")
	mailSender := utils.GetEnvOrFile("MAILTRAP_SENDER_EMAIL")

	mailService, err := mailer.New(mailHost, mailPort, mailUserName, mailPassword, mailSender)
	if err != nil {
		utils.WriteServerError(logger, "failed to setup mail service", err)
		os.Exit(1)
	}

	psqlService := database.New(conn)
	publicHandler := public.NewPublicHandler(config.environment, config.version, logger)

	authService := auth.NewAuthService(psqlService, config.jwtSecret, config.jwtTTL, config.logger)
	apiKeyService := auth.NewApiKeyService(8, config.apiKeyPrefix, psqlService, config.logger, &wg)
	authHandler := auth.NewAuthHandler(authService, apiKeyService, config.refreshTokenTTL, config.logger, mailService, &wg)

	userService := user.NewUserService(psqlService, config.logger)
	userHandler := user.NewUserHandler(userService)

	router := router.RegisterRoutes(routeConfig, authHandler, authService, apiKeyService, userHandler, publicHandler)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.port),
		Handler:      router,
		IdleTimeout:  time.Minute,
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 40 * time.Second,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	expvar.NewString("version").Set(version)
	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))
	expvar.Publish("databaase", expvar.Func(func() any {
		return db.Health(conn)
	}))
	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))

	go gracefulShutdown(conn, httpServer, done, &wg, logger)

	log.Printf("The server is starting on: http://%s:%d in %s environment\n", config.domain, config.port, config.environment)
	err = httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		logger.Error(fmt.Sprintf("http server error: %s", err))
		os.Exit(1)
	}

	// Wait for the graceful shutdown to complete
	<-done
	logger.Info("Graceful shutdown complete.")
}

func gracefulShutdown(
	conn *sql.DB,
	httpServer *http.Server,
	done chan bool,
	wg *sync.WaitGroup,
	logger *slog.Logger,
) {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	slog.Info("shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has 10 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Info("Server forced to shutdown with error, ", "Message", err.Error())
	}

	logger.Info("completing background tasks")
	wg.Wait()

	if err := conn.Close(); err != nil {
		logger.Error("failed to close database connection", "error", err)
	} else {
		logger.Info("Database connection disconnected")
	}
	// Notify the main goroutine that the shutdown is complete
	done <- true
}
