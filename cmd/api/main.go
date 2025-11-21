package main

import (
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/i-christian/fileShare/internal/db"
	"github.com/i-christian/fileShare/internal/utils"
	"github.com/i-christian/fileShare/internal/utils/security"
	"github.com/i-christian/fileShare/internal/vcs"
	_ "github.com/joho/godotenv/autoload"
)

// config holds all configuration for the application
type config struct {
	port            int
	env             string
	domain          string
	version         string
	maxUploadSize   uint64
	jwtSecret       string
	apiKeyPrefix    string
	jwtTTL          time.Duration
	refreshTokenTTL time.Duration
	limiter         struct {
		rps     float64
		burst   int
		enabled bool
	}
	mail struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
}

// application holds the dependencies for the HTTP handlers, helpers, and middleware.
type application struct {
	config config
	logger *slog.Logger
	wg     sync.WaitGroup
}

func main() {
	var logger *slog.Logger
	if utils.GetEnvOrFile("ENV") == "testing" {
		logger = slog.New(slog.DiscardHandler)
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true}))
	}

	cfg, err := parseConfig()
	if err != nil {
		logger.Error("failed to parse config", "error", err)
		os.Exit(1)
	}

	utils.ValidateEnvVars(logger)

	dbConn, err := db.InitialiseDB(utils.GetEnvOrFile("GOOSE_DRIVER"))
	if err != nil {
		logger.Error("failed to initialise database", "error", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	if err := runMigrations(dbConn, logger); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	app := &application{
		config: cfg,
		logger: logger,
	}

	if err := app.serve(dbConn); err != nil {
		logger.Error("server terminated", "error", err)
		os.Exit(1)
	}
}

func parseConfig() (config, error) {
	var cfg config

	port, _ := strconv.Atoi(utils.GetEnvOrFile("PORT"))
	jwtSecret, err := hex.DecodeString(utils.GetEnvOrFile("JWT_SECRET"))
	if err != nil {
		return cfg, fmt.Errorf("invalid JWT secret: %w", err)
	}

	mailPort, _ := strconv.Atoi(utils.GetEnvOrFile("MAILTRAP_SMTP_PORT"))

	var parsedValue uint64
	parsedValue, err = strconv.ParseUint(utils.GetEnvOrFile("MAX_UPLOAD_SIZE"), 10, 64)
	if err != nil || parsedValue == 0 {
		// Set maximum default size to 200MB
		parsedValue = 200 << 20
	}

	cfg.maxUploadSize = parsedValue
	cfg.port = port
	cfg.env = utils.GetEnvOrFile("ENV")
	cfg.domain = utils.GetEnvOrFile("DOMAIN")
	cfg.version = vcs.Version()
	cfg.jwtSecret = string(jwtSecret)
	cfg.apiKeyPrefix = security.ShortProjectPrefix(utils.GetEnvOrFile("PROJECT_NAME"))
	cfg.jwtTTL = 15 * time.Minute
	cfg.refreshTokenTTL = 7 * 24 * time.Hour

	cfg.mail.host = utils.GetEnvOrFile("MAILTRAP_SMTP_HOST")
	cfg.mail.port = mailPort
	cfg.mail.username = utils.GetEnvOrFile("MAILTRAP_USER")
	cfg.mail.password = utils.GetEnvOrFile("MAILTRAP_PASSWORD")
	cfg.mail.sender = utils.GetEnvOrFile("MAILTRAP_SENDER_EMAIL")

	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", cfg.version)
		os.Exit(0)
	}

	return cfg, nil
}

func runMigrations(conn *sql.DB, logger *slog.Logger) error {
	var err error
	for i := range 10 {
		logger.Info("running database migration", "attempt", i+1)
		err = db.SetUpMigration(conn)
		if err == nil {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("migration failed after retries: %w", err)
}
