package dependencies

import (
	"context"
	"log/slog"

	"skilly/internal/infrastructure/logging"
	"skilly/internal/infrastructure/mongo"
)

func SetupDependencies() Dependencies {
	var deps Dependencies

	deps.Logger = logging.SetupLogger()
	if deps.Logger == nil {
		panic("error while dependencies setup: Logger is nil")
	}
	deps.Mongo = mongo.MustConnect(context.Background(), mongo.LoadConfigFromEnv(), deps.Logger)
	if deps.Mongo == nil {
		panic("error while dependencies setup: MongoDB client is nil")
	}

	return deps
}

type Dependencies struct {
	Logger *slog.Logger
	Mongo *mongo.Client
}
