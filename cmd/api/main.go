package main

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/i-christian/fileShare/internal/auth"
	"github.com/i-christian/fileShare/internal/database"
	"github.com/i-christian/fileShare/internal/db"
	"github.com/i-christian/fileShare/internal/router"
	"github.com/i-christian/fileShare/internal/utils"
	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	logger          *slog.Logger
	domain          string
	jwtSecret       string
	port            int
	jwtTTL          time.Duration
	refreshTokenTTL time.Duration
}

func main() {
	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)

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

	psqlService := database.New(conn)
	port, _ := strconv.Atoi(utils.GetEnvOrFile("PORT"))
	domain := utils.GetEnvOrFile("DOMAIN")
	config := &Config{
		port:            port,
		domain:          domain,
		jwtSecret:       string(jwtSecret),
		jwtTTL:          15 * time.Minute,
		refreshTokenTTL: 7 * 24 * time.Hour,
		logger:          logger,
	}

	authService := auth.NewAuthService(psqlService, config.jwtSecret, config.jwtTTL)
	authHandler := auth.NewAuthHandler(authService, config.refreshTokenTTL, config.logger)

	router := router.RegisterRoutes(config.domain, authHandler)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.port),
		Handler:      router,
		IdleTimeout:  time.Minute,
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 40 * time.Second,
	}

	log.Printf("The server is starting on: http://%s:%s\n", domain, strconv.Itoa(port))
	err = httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("http server error: %s", err))
	}

	// Wait for the graceful shutdown to complete
	<-done
	slog.Info("Graceful shutdown complete.")
}

func gracefulShutdown(conn *sql.DB, httpServer *http.Server, done chan bool) {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	slog.Info("shutting down gracefully, press Ctrl+C again to force")

	// Shutting down database connection
	if err := db.Close(conn); err != nil {
		slog.Info("Database connection pool closed successfully")
	}

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Info("Server forced to shutdown with error, ", "Message", err.Error())
	}

	slog.Info("Server exiting...")

	// Notify the main goroutine that the shutdown is complete
	done <- true
}
