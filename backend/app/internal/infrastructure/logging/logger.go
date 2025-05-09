package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"skilly/internal/infrastructure/utils"
)

// Config holds configuration for the logger.
type config struct {
	Level     string // e.g., "debug", "info", "warn", "error"
	Format    string // e.g., "text" or "json"
	AddSource bool   // Whether to include source file and line
	Output    io.Writer
}

// NewLogger creates and configures a new slog.Logger.
func newLogger(cfg config) (*slog.Logger, error) {
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		slog.Warn("Unknown log level, defaulting to INFO", "configuredLevel", cfg.Level)
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		AddSource: cfg.AddSource,
		Level:     level,
	}

	var handler slog.Handler
	output := cfg.Output
	if output == nil {
		output = os.Stdout // Default to stdout if not specified
	}

	switch strings.ToLower(cfg.Format) {
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	case "text":
		fallthrough
	default:
		handler = slog.NewTextHandler(output, opts)
	}

	logger := slog.New(handler)
	return logger, nil
}

// MustNewLogger is like NewLogger but panics on error.
// Useful for initialization code in main.
func mustNewLogger(cfg config) *slog.Logger {
	logger, err := newLogger(cfg)
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	return logger
}

// NewProductionLoggerConfig provides a sensible default configuration for production.
func newProductionLoggerConfig() config {
	return config{
		Level:     "info",     // Typically INFO in prod, DEBUG in dev
		Format:    "json",     // JSON is better for log aggregators
		AddSource: false,      // Can be noisy and add overhead in prod; enable for debugging
		Output:    os.Stdout,
	}
}

// NewDevelopmentLoggerConfig provides a sensible default configuration for development.
func newDevelopmentLoggerConfig() config {
	return config{
		Level:     "debug",
		Format:    "text", // Text is more human-readable for local dev
		AddSource: true,
		Output:    os.Stdout,
	}
}

func SetupLogger() *slog.Logger {
	var loggerCfg config
	switch utils.GetEnvMode() {
	case "dev":
		loggerCfg = newDevelopmentLoggerConfig()
	case "prod":
		loggerCfg = newProductionLoggerConfig()
	}

	appLogger := mustNewLogger(loggerCfg)
	slog.SetDefault(appLogger)

	return appLogger
}
