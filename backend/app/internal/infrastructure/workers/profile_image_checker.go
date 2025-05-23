package workers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	kafkalib "github.com/segmentio/kafka-go"

	"skilly/internal/infrastructure/dependencies"
)

const (
	profileImageCheckerTopic = "minio-events"
	groupID                  = "profile-image-checker"
)

func ProfileImageChecker(ctx context.Context, deps *dependencies.Dependencies) {
	consumer, err := deps.Kafka.NewConsumer(profileImageCheckerTopic, groupID)
	if err != nil {
		deps.Logger.Error("failed to create consumer", slog.Any("error", err))
		return
	}
	defer consumer.Close()

	for {
		msg, err := consumer.FetchMessage(ctx)
		deps.Logger.Info("starting profile image checker")
		if err != nil {
			if errors.Is(err, context.Canceled) {
				deps.Logger.Info("context canceled")
				return
			} else {
				deps.Logger.Error("failed to read message", slog.Any("error", err))
				continue
			}
		}
		handleMessage(msg, deps)
		consumer.CommitMessages(ctx, msg)
	}
}

func handleMessage(msg kafkalib.Message, deps *dependencies.Dependencies) {
	parsedMsg := map[string]interface{}{}
	err := json.Unmarshal(msg.Value, &parsedMsg)
	if err != nil {
		deps.Logger.Error("failed to unmarshal message", slog.Any("error", err))
		return
	}
	if parsedMsg["EventName"].(string) != "s3:ObjectCreated:Put" {
		deps.Logger.Debug("message not for this consumer", slog.String("event_name", parsedMsg["EventName"].(string)))
		return
	}

	path, _ := strings.CutPrefix(parsedMsg["Key"].(string), deps.S3.GetBucketName()+"/")
	obj, err := deps.S3.GetObject(context.Background(), path)
	if err != nil {
		deps.Logger.Error("failed to get object", slog.Any("error", err))
		return
	}
	defer obj.Close()

	objData := make([]byte, 512)
	_, err = obj.Read(objData)
	if err != nil {
		deps.Logger.Error("failed to read object", slog.Any("error", err))
		return
	}

	deps.Logger.Info("checking object", slog.Any("image", objData))
	contentType := http.DetectContentType(objData)
	if strings.HasPrefix(contentType, "image/") {
		deps.Logger.Info("image confirmed", slog.String("path", path), slog.String("content_type", contentType))
		return
	}

	deps.Logger.Info("non-image confirmed, deleting", slog.String("path", path), slog.String("content_type", contentType))
	err = deps.S3.RemoveObject(context.Background(), path)
	if err != nil {
		deps.Logger.Error("failed to remove non-image", slog.Any("error", err))
	}
}
