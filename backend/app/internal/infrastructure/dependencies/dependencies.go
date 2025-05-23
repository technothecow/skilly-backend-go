package dependencies

import (
	"context"
	"log/slog"

	"skilly/internal/adapters/kafka"
	"skilly/internal/adapters/mongo"
	"skilly/internal/adapters/s3"
	"skilly/internal/infrastructure/logging"
)

type Dependencies struct {
	Logger *slog.Logger
	Mongo  *mongo.Client
	S3     s3.Client // it's interface, so pointer is not required
	Kafka  *kafka.Client
}

func MustNewDependencies() *Dependencies {
	var deps Dependencies

	deps.Logger = logging.SetupLogger()
	if deps.Logger == nil {
		panic("error while dependencies setup: Logger is nil")
	}
	deps.Mongo = mongo.MustConnect(context.Background(), mongo.LoadConfigFromEnv(), deps.Logger)
	if deps.Mongo == nil {
		panic("error while dependencies setup: MongoDB client is nil")
	}
	deps.S3 = s3.MustConnect(context.Background(), deps.Logger)
	if deps.S3 == nil {
		panic("error while dependencies setup: S3 client is nil")
	}
	deps.Kafka = kafka.MustNewClient(context.Background(), kafka.LoadConfigFromEnv(), deps.Logger)
	if deps.Kafka == nil {
		panic("error while dependencies setup: Kafka client is nil")
	}

	return &deps
}

func (d *Dependencies) Close(ctx context.Context) {
	d.Logger.Info("Disconnecting MongoDB client...")
	d.Mongo.Disconnect(ctx)
}
