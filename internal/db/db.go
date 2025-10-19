package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/i-christian/fileShare/internal/utils"
	"github.com/pressly/goose/v3"

	_ "github.com/lib/pq"
)

//go:embed schema/*.sql
var embedMigrations embed.FS

// InitialiseDB opens and configures a database connections for the application.
func InitialiseDB(driverName string) (*sql.DB, error) {
	dbHost := utils.GetEnvOrFile("DB_HOST")
	dbUser := utils.GetEnvOrFile("DB_USERNAME")
	dbName := utils.GetEnvOrFile("DB_NAME")
	dbPort := utils.GetEnvOrFile("DB_PORT")
	dbSchema := utils.GetEnvOrFile("DB_SCHEMA")

	parsedPassword := utils.ToggleEnvOrSecret(os.Getenv("DB_PASSWORD_FILE"), os.Getenv("DB_PASSWORD"))

	var dataSourceName string
	if utils.GetEnvOrFile("ENV") == "testing" {
		dataSourceName = utils.GetEnvOrFile("DB_URL")
	} else {
		dataSourceName = fmt.Sprintf("%s://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s", driverName, dbUser, parsedPassword, dbHost, dbPort, dbName, dbSchema)
	}

	conn, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(25)
	conn.SetConnMaxLifetime(30 * time.Minute)

	return conn, nil
}

// SetUpMigration Setup database migrations and closes database connection afterwards
func SetUpMigration(db *sql.DB) error {
	goose.SetBaseFS(embedMigrations)

	err := goose.SetDialect("postgres")
	if err != nil {
		return err
	}

	if err := goose.Up(db, os.Getenv("GOOSE_MIGRATION_DIR")); err != nil {
		return err
	}

	return nil
}

// Health checks the health of the database connection by pinging the database.
// It returns a map with keys indicating various health statistics.
func Health(db *sql.DB) map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	// Ping the database
	err := db.PingContext(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Fatalf("db down: %v", err) // Log the error and terminate the program
		return stats
	}

	// Database is up, add more statistics
	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := db.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	// Evaluate stats to provide a health message
	if dbStats.OpenConnections > 25 {
		stats["message"] = "The database is experiencing heavy load."
	}

	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

// Close closes the database connection.
// It logs a message indicating the disconnection from the specific database.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func Close(conn *sql.DB) error {
	slog.Info("Disconnected from database")
	return conn.Close()
}
