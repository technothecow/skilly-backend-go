package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	kafkalib "github.com/segmentio/kafka-go"
)

// Config holds Kafka connection parameters.
type Config struct {
	Brokers        []string // List of Kafka brokers
	DefaultTopic   string   // A default topic for convenience
	DefaultGroupID string   // A default consumer group ID
}

// Client wraps Kafka configurations and provides convenience methods.
// Unlike MinIO or MongoDB where the client is long-lived stateful object for a bucket or collection,
// kafka-go Writer and Reader are often created per operation or for a specific task.
// This client acts more as a configured factory.
type Client struct {
	config     Config
	logger     *slog.Logger
	brokerList []string
}

// DefaultConfig provides sensible defaults for Kafka.
func DefaultConfig() Config {
	return Config{
		Brokers:        []string{"kafka:9092"},
		DefaultTopic:   "default",
		DefaultGroupID: "default",
	}
}

// LoadConfigFromEnv loads Kafka configuration from environment variables.
func LoadConfigFromEnv() Config {
	cfg := DefaultConfig()

	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		cfg.Brokers = strings.Split(brokers, ",")
	}
	if topic := os.Getenv("KAFKA_DEFAULT_TOPIC"); topic != "" {
		cfg.DefaultTopic = topic
	}
	if groupID := os.Getenv("KAFKA_DEFAULT_GROUP_ID"); groupID != "" {
		cfg.DefaultGroupID = groupID
	}

	return cfg
}

// NewClient creates a new KafkaClient.
// It doesn't establish a persistent connection itself but prepares the configuration
// for creating writers and readers.
func NewClient(ctx context.Context, cfg Config, logger *slog.Logger) (*Client, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers list cannot be empty")
	}
	for i, b := range cfg.Brokers {
		cfg.Brokers[i] = strings.TrimSpace(b)
	}

	client := &Client{
		config:     cfg,
		logger:     logger,
		brokerList: cfg.Brokers,
	}

	attemptsLeft := 3
	for attemptsLeft > 0 {
		err := client.Ping(ctx)
		if err == nil {
			break
		}
		logger.Error("Kafka initial connectivity check failed", slog.Any("error", err), slog.Int("attempts_left", attemptsLeft))
		time.Sleep(time.Second)
		attemptsLeft--
	}

	logger.Info("Kafka client configured", slog.Any("brokers", cfg.Brokers))
	return client, nil
}

// MustNewClient is like NewClient but panics if an error occurs.
func MustNewClient(ctx context.Context, cfg Config, logger *slog.Logger) *Client {
	client, err := NewClient(ctx, cfg, logger)
	if err != nil {
		panic(fmt.Errorf("failed to create Kafka client: %w", err))
	}
	return client
}

// Disconnect for KafkaClient. Since writers/readers are typically managed
// by the caller of Produce/Consume methods, this can be a no-op or clean up
// any shared resources if the client were to manage them (e.g., a connection pool).
func (c *Client) Disconnect() error {
	c.logger.Info("KafkaClient Disconnect called (typically no-op for this design)")
	return nil // No specific resources held by KafkaClient itself in this design
}

// GetWriterConfig returns a base kafka.WriterConfig based on the client's configuration.
func (c *Client) GetWriterConfig() kafkalib.WriterConfig {
	wc := kafkalib.WriterConfig{
		Brokers:  c.brokerList,
		Balancer: &kafkalib.LeastBytes{},
		// Configure other writer options as needed:
		// BatchSize: 100,
		// BatchTimeout: 10 * time.Millisecond,
		// Logger: kafka.LoggerFunc(c.Logger.Debugf), // kafka-go specific logging
		// ErrorLogger: kafka.LoggerFunc(c.Logger.Errorf),
	}
	return wc
}

// GetReaderConfig returns a base kafka.ReaderConfig based on the client's configuration.
func (c *Client) GetReaderConfig(topic, groupID string) kafkalib.ReaderConfig {
	if topic == "" {
		topic = c.config.DefaultTopic
	}
	if groupID == "" {
		groupID = c.config.DefaultGroupID
	}
	rc := kafkalib.ReaderConfig{
		Brokers: c.brokerList,
		Topic:   topic,
		GroupID: groupID,
		// Configure other reader options as needed:
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
		MaxWait:  1 * time.Second,
	}
	return rc
}

// ProduceMessage sends a single message to the specified topic.
// For high-throughput, consider creating a kafka.Writer once and reusing it.
// This method creates/closes a writer per call for simplicity.
func (c *Client) ProduceMessage(ctx context.Context, topic string, key, value []byte) error {
	if topic == "" {
		topic = c.config.DefaultTopic
	}

	writerConfig := c.GetWriterConfig()
	writerConfig.Topic = topic

	writer := kafkalib.NewWriter(writerConfig)
	defer func() {
		if err := writer.Close(); err != nil {
			c.logger.Error("Failed to close Kafka writer", slog.Any("error", err), slog.String("topic", topic))
		}
	}()

	msg := kafkalib.Message{
		Key:   key,
		Value: value,
		Time:  time.Now(), // Optional: Kafka will timestamp if not provided
	}

	err := writer.WriteMessages(ctx, msg)
	if err != nil {
		c.logger.Error("Failed to write Kafka message", slog.Any("error", err), slog.String("topic", topic))
		return fmt.Errorf("failed to write message to topic %s: %w", topic, err)
	}
	c.logger.Debug("Message produced successfully", slog.String("topic", topic), slog.String("key", string(key)))
	return nil
}

// NewConsumer creates and returns a new kafka.Reader for the given topic and groupID.
// The caller is responsible for calling Close() on the returned reader.
func (c *Client) NewConsumer(topic, groupID string) (*kafkalib.Reader, error) {
	if topic == "" && c.config.DefaultTopic == "" {
		return nil, fmt.Errorf("topic must be specified or a default topic configured")
	}
	if groupID == "" && c.config.DefaultGroupID == "" {
		return nil, fmt.Errorf("groupID must be specified or a default group ID configured")
	}

	readerConfig := c.GetReaderConfig(topic, groupID)
	reader := kafkalib.NewReader(readerConfig)

	c.logger.Info("Kafka consumer created", slog.String("topic", readerConfig.Topic), slog.String("group_id", readerConfig.GroupID))
	return reader, nil
}

// Ping (Example - kafka-go doesn't have a direct ping)
// A more robust ping might involve trying to get cluster metadata or list topics.
// This is a very basic check using a temporary writer.
func (c *Client) Ping(ctx context.Context) error {
	c.logger.Debug("Attempting Kafka ping...")
	// For a simple "ping", we can try to create a writer and immediately close it.
	// This isn't a perfect health check but can catch basic connectivity/auth issues.
	// Alternatively, try to fetch metadata if kafka-go offers a simple way.
	// For now, we'll test creating a writer for a dummy topic.
	cfg := c.GetWriterConfig()
	cfg.Topic = "ping-topic-test" // A dummy topic
	cfg.Async = true              // Don't block for ping

	// Reduce timeouts for a faster ping
	if cfg.Dialer != nil {
		originalTimeout := cfg.Dialer.Timeout
		cfg.Dialer.Timeout = 3 * time.Second                    // Shorter timeout for ping
		defer func() { cfg.Dialer.Timeout = originalTimeout }() // Restore
	}

	// kafka.DialContext doesn't really test connectivity in a way that exposes issues easily before Write.
	// We can try to list topics, which requires a kafka.Client, not directly a Writer/Reader.
	// Let's use a temporary writer creation as a proxy.

	conn, err := kafkalib.DialContext(ctx, "tcp", c.brokerList[0]) // Try connecting to the first broker
	if err != nil {
		c.logger.Error("Kafka ping: dial failed", slog.Any("error", err), slog.String("broker", c.brokerList[0]))
		return fmt.Errorf("kafka ping: dial failed for %s: %w", c.brokerList[0], err)
	}
	defer conn.Close()

	// Fetch metadata as a more robust check
	_, err = conn.ReadPartitions() // You can pick any topic if you know one exists
	if err != nil {
		c.logger.Error("Kafka ping: failed to read partitions (metadata)", slog.Any("error", err))
		return fmt.Errorf("kafka ping: could not read partitions: %w", err)
	}

	c.logger.Info("Kafka ping successful")
	return nil
}
