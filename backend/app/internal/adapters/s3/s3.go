package s3

import (
	"context"
	"log/slog"
	"net/url"
	"time"

	"skilly/internal/adapters/minio"
	"skilly/internal/infrastructure/s3"
)

// Must be thread safe
type Client interface {
	GenerateDownloadUrl(ctx context.Context, key string, expires time.Duration) (url.URL, error)
	GenerateUploadUrl(ctx context.Context, key string, expires time.Duration) (url.URL, error)
	GetObject(ctx context.Context, key string) (s3.Object, error)
	RemoveObject(ctx context.Context, key string) error
	GetBucketName() string
	Disconnect() error
}

func MustConnect(ctx context.Context, logger *slog.Logger) Client {
	cfg := minio.LoadConfigFromEnv()
	client := minio.MustConnect(ctx, cfg, logger)
	return client
}
