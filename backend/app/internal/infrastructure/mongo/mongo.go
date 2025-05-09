package mongo

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Config holds MongoDB connection parameters.
type Config struct {
	URI                  string
	DatabaseName         string
	ConnectTimeout       time.Duration
	PingTimeout          time.Duration
	MinPoolSize          uint64
	MaxPoolSize          uint64
	MaxConnIdleTime      time.Duration
	RetryConnectAttempts int
	RetryConnectDelay    time.Duration
}

// Client is a wrapper around the official mongo.Client and its Config.
type Client struct {
	*mongo.Client
	Database *mongo.Database
	logger   *slog.Logger
	config   Config
}

// DefaultConfig returns a Config struct with sensible default values.
// These can be overridden by environment variables.
func DefaultConfig() Config {
	return Config{
		URI:                  "mongodb://mongo:27017",
		DatabaseName:         "db",
		ConnectTimeout:       10 * time.Second,
		PingTimeout:          5 * time.Second,  // Should be less than ConnectTimeout
		MinPoolSize:          5,                // Sensible default, adjust based on load
		MaxPoolSize:          100,              // Default in driver
		MaxConnIdleTime:      10 * time.Minute, // Default in driver
		RetryConnectAttempts: 3,
		RetryConnectDelay:    2 * time.Second,
	}
}

// LoadConfigFromEnv loads configuration from environment variables, falling back to defaults.
func LoadConfigFromEnv() Config {
	cfg := DefaultConfig()

	if uri := os.Getenv("MONGODB_URI"); uri != "" {
		cfg.URI = uri
	}
	if dbName := os.Getenv("MONGODB_DATABASE_NAME"); dbName != "" {
		cfg.DatabaseName = dbName
	}
	if timeoutStr := os.Getenv("MONGODB_CONNECT_TIMEOUT_SECONDS"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil {
			cfg.ConnectTimeout = time.Duration(timeout) * time.Second
		}
	}
	if timeoutStr := os.Getenv("MONGODB_PING_TIMEOUT_SECONDS"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil {
			cfg.PingTimeout = time.Duration(timeout) * time.Second
		}
	}
	if minPool := os.Getenv("MONGODB_MIN_POOL_SIZE"); minPool != "" {
		if val, err := strconv.ParseUint(minPool, 10, 64); err == nil {
			cfg.MinPoolSize = val
		}
	}
	if maxPool := os.Getenv("MONGODB_MAX_POOL_SIZE"); maxPool != "" {
		if val, err := strconv.ParseUint(maxPool, 10, 64); err == nil {
			cfg.MaxPoolSize = val
		}
	}
	if idleTime := os.Getenv("MONGODB_MAX_CONN_IDLE_TIME_MINUTES"); idleTime != "" {
		if val, err := strconv.ParseUint(idleTime, 10, 64); err == nil {
			cfg.MaxConnIdleTime = time.Duration(val) * time.Minute
		}
	}
	if retryAttempts := os.Getenv("MONGODB_RETRY_CONNECT_ATTEMPTS"); retryAttempts != "" {
		if val, err := strconv.Atoi(retryAttempts); err == nil {
			cfg.RetryConnectAttempts = val
		}
	}
	if retryDelay := os.Getenv("MONGODB_RETRY_CONNECT_DELAY_SECONDS"); retryDelay != "" {
		if val, err := strconv.Atoi(retryDelay); err == nil {
			cfg.RetryConnectDelay = time.Duration(val) * time.Second
		}
	}
	return cfg
}

// Connect establishes a connection to MongoDB based on the provided configuration.
// It also performs a ping check with retries to ensure the connection is live.
func Connect(ctx context.Context, cfg Config, logger *slog.Logger) (*Client, error) {
	if logger == nil {
		logger = slog.Default() // Fallback to default logger if none provided
	}
	log := logger.With(slog.String("mongodb_uri_partial", getPartialURI(cfg.URI))) // Log partial URI for security

	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetMinPoolSize(cfg.MinPoolSize).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMaxConnIdleTime(cfg.MaxConnIdleTime).
		SetConnectTimeout(cfg.ConnectTimeout)
		// Potentially add: .SetRetryWrites(true) for replica sets, .SetReadConcern(), .SetWriteConcern()

	log.Info("Attempting to connect to MongoDB",
		slog.String("database", cfg.DatabaseName),
		slog.Uint64("min_pool_size", cfg.MinPoolSize),
		slog.Uint64("max_pool_size", cfg.MaxPoolSize),
		slog.Duration("max_conn_idle_time", cfg.MaxConnIdleTime),
		slog.Duration("connect_timeout", cfg.ConnectTimeout),
	)

	connectCtx, cancelConnect := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancelConnect()

	mongoC, err := mongo.Connect(connectCtx, clientOptions)
	if err != nil {
		log.Error("Failed to create MongoDB client", slog.Any("error", err))
		return nil, fmt.Errorf("mongo.Connect: %w", err)
	}

	var pingErr error
	for i := 0; i < cfg.RetryConnectAttempts+1; i++ {
		if i > 0 { // Don't sleep on the first attempt
			log.Warn("Retrying MongoDB ping", slog.Int("attempt", i+1), slog.Duration("delay", cfg.RetryConnectDelay))
			time.Sleep(cfg.RetryConnectDelay)
		}

		pingCtx, cancelPing := context.WithTimeout(ctx, cfg.PingTimeout)
		pingErr = mongoC.Ping(pingCtx, readpref.Primary()) // Ping primary to ensure write capabilities
		cancelPing()

		if pingErr == nil {
			log.Info("Successfully connected and pinged MongoDB.")
			db := mongoC.Database(cfg.DatabaseName)
			return &Client{Client: mongoC, Database: db, logger: logger, config: cfg}, nil
		}
		log.Warn("MongoDB ping failed", slog.Int("attempt", i+1), slog.Any("error", pingErr))
	}

	// If all pings failed, attempt to disconnect the partially formed client and return the last error
	log.Error("Failed to ping MongoDB after multiple retries", slog.Any("last_error", pingErr))
	disconnectCtx, cancelDisconnect := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelDisconnect()
	if err := mongoC.Disconnect(disconnectCtx); err != nil {
		log.Error("Error disconnecting client after failed ping", slog.Any("error", err))
	}
	return nil, fmt.Errorf("failed to ping MongoDB after %d retries: %w", cfg.RetryConnectAttempts, pingErr)
}

func MustConnect(ctx context.Context, cfg Config, logger *slog.Logger) *Client {
	client, err := Connect(ctx, cfg, logger)
	if err != nil {
		panic(err)
	}
	return client
}

// Disconnect gracefully closes the MongoDB client connection.
func (c *Client) Disconnect(ctx context.Context) error {
	if c.Client == nil {
		c.logger.Warn("MongoDB client is nil, no disconnection needed.")
		return nil
	}
	c.logger.Info("Disconnecting MongoDB client...", slog.String("mongodb_uri_partial", getPartialURI(c.config.URI)))

	disconnectTimeout := 10 * time.Second // Give some time for connections to close
	if val := os.Getenv("MONGODB_DISCONNECT_TIMEOUT_SECONDS"); val != "" {
		if t, err := strconv.Atoi(val); err == nil {
			disconnectTimeout = time.Duration(t) * time.Second
		}
	}

	disconnectCtx, cancel := context.WithTimeout(ctx, disconnectTimeout)
	defer cancel()

	if err := c.Client.Disconnect(disconnectCtx); err != nil {
		c.logger.Error("Failed to disconnect MongoDB client", slog.Any("error", err))
		return fmt.Errorf("MongoDB Disconnect error: %w", err)
	}
	c.logger.Info("MongoDB client disconnected successfully.")
	return nil
}

// getPartialURI helps log the URI without leaking credentials.
// Examples:
// "mongodb://user:password@host:port/db?options" -> "mongodb://<hidden>@host:port/db?options"
// "mongodb+srv://user:password@cluster.mongodb.net/db" -> "mongodb+srv://<hidden>@cluster.mongodb.net/db"
// "mongodb://host:port/db" -> "mongodb://host:port/db" (no change if no credentials)
func getPartialURI(uri string) string {
	schemeEnd := "://"
	schemeIndex := strings.Index(uri, schemeEnd)
	if schemeIndex == -1 {
		return uri // Not a recognizable scheme, return as is
	}

	// Part after "://"
	restOfURI := uri[schemeIndex+len(schemeEnd):]

	atIndex := strings.Index(restOfURI, "@")
	if atIndex == -1 {
		return uri // No "@" symbol, so likely no credentials part, return as is
	}

	// Credentials are before "@", host/path is after
	// We want to keep the scheme and everything from "@" onwards
	schemePart := uri[:schemeIndex+len(schemeEnd)]
	hostAndBeyondPart := restOfURI[atIndex:] // Includes the "@"

	return schemePart + "<credentials-hidden>" + hostAndBeyondPart
}
