package main

import (
	"context"
	"database/sql"
	"encoding/hex"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/i-christian/fileShare/internal/db"
	"github.com/i-christian/fileShare/internal/utils"
	_ "github.com/joho/godotenv/autoload"
)

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
	fileStore := utils.SetUpFileStorage(logger)

	SecretKey, err := hex.DecodeString(utils.GetEnvOrFile("RANDOM_HEX"))
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	log.Printf("The server is starting on: http://%s:%s\n", os.Getenv("DOMAIN"), os.Getenv("PORT"))

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	err := httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("http server error: %s", err))
	}

	// Wait for the graceful shutdown to complete
	<-done
	slog.Info("Graceful shutdown complete.")
}
