package tests

import (
	"context"
	"log"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestUsingDockerCompose(ctx context.Context, t *testing.T) (func(), error) {
	composeFilePath := "../docker-compose.tst.yml"

	stack, err := compose.NewDockerComposeWith(
		compose.WithStackFiles(composeFilePath),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = stack.
		WaitForService("app", wait.ForListeningPort("8000/tcp")).
		WaitForService("mongo", wait.ForListeningPort("27017/tcp")).
		Up(ctx, compose.Wait(true))
	if err != nil {
		log.Fatal(err)
	}

	return func() {
		err := stack.Down(
			context.Background(),
			compose.RemoveOrphans(true),
			compose.RemoveVolumes(true),
			compose.RemoveImagesLocal,
		)
		if err != nil {
			log.Fatal(err)
		}
	}, nil
}
