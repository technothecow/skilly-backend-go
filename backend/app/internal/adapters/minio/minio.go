package minio

import (
	"context"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"time"

	miniolib "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"skilly/internal/infrastructure/s3"
)

type Config struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
	UseSSL     bool
}

type Client struct {
	*miniolib.Client
	BucketName string
}

func DefaultConfig() Config {
	return Config{
		Endpoint:   "http://minio:9000",
		AccessKey:  "minio",
		SecretKey:  "minio123",
		BucketName: "skilly",
		UseSSL:     false,
	}
}

func LoadConfigFromEnv() Config {
	cfg := DefaultConfig()

	if endpoint := os.Getenv("MINIO_ENDPOINT"); endpoint != "" {
		cfg.Endpoint = endpoint
	}
	if accessKey := os.Getenv("MINIO_ACCESS_KEY"); accessKey != "" {
		cfg.AccessKey = accessKey
	}
	if secretKey := os.Getenv("MINIO_SECRET_KEY"); secretKey != "" {
		cfg.SecretKey = secretKey
	}
	if bucketName := os.Getenv("MINIO_BUCKET_NAME"); bucketName != "" {
		cfg.BucketName = bucketName
	}
	if useSSL := os.Getenv("MINIO_USE_SSL"); useSSL != "" {
		if val, err := strconv.ParseBool(useSSL); err == nil {
			cfg.UseSSL = val
		}
	}
	return cfg
}

func (m *Client) ensureBucketExists(ctx context.Context, logger *slog.Logger) error {
	exists, err := m.BucketExists(ctx, m.BucketName)
	if err != nil {
		return err
	}

	if !exists {
		logger.Info("Creating bucket", slog.String("bucket_name", m.BucketName))
		if err := m.MakeBucket(ctx, m.BucketName, miniolib.MakeBucketOptions{}); err != nil {
			return err
		}
		logger.Info("Bucket created", slog.String("bucket_name", m.BucketName))
	}

	return nil
}

func Connect(ctx context.Context, cfg Config, logger *slog.Logger) (*Client, error) {
	minioClient, err := miniolib.New(cfg.Endpoint, &miniolib.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		logger.Error("Failed to create Minio client", slog.Any("error", err))
		return nil, err
	}

	client := &Client{
		Client:     minioClient,
		BucketName: cfg.BucketName,
	}

	if err := client.ensureBucketExists(ctx, logger); err != nil {
		logger.Error("Failed to ensure bucket exists", slog.Any("error", err))
		return nil, err
	}

	return client, nil
}

func MustConnect(ctx context.Context, cfg Config, logger *slog.Logger) *Client {
	client, err := Connect(ctx, cfg, logger)
	if err != nil {
		panic(err)
	}
	return client
}

func (c *Client) Disconnect() error {
	return nil
}

func (c *Client) GenerateUploadUrl(ctx context.Context, key string, expires time.Duration) (url.URL, error) {
	uploadUrl, err := c.PresignedPutObject(ctx, c.BucketName, key, expires)
	if err != nil {
		return url.URL{}, err
	}

	return *uploadUrl, nil
}

func (c *Client) GenerateDownloadUrl(ctx context.Context, key string, expires time.Duration) (url.URL, error) {
	downloadUrl, err := c.PresignedGetObject(ctx, c.BucketName, key, expires, nil)
	if err != nil {
		return url.URL{}, err
	}

	return *downloadUrl, nil
}

func (c *Client) GetObject(ctx context.Context, key string) (s3.Object, error) {
	return c.Client.GetObject(ctx, c.BucketName, key, miniolib.GetObjectOptions{})
}

func (c *Client) RemoveObject(ctx context.Context, key string) error {
	return c.Client.RemoveObject(ctx, c.BucketName, key, miniolib.RemoveObjectOptions{})
}

func (c *Client) GetBucketName() string {
	return c.BucketName
}
