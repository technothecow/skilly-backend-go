package tests

import (
	"context"
	"net/http"
	"testing"
)

func TestPing(t *testing.T) {
	ctx := context.Background()
	close, err := TestUsingDockerCompose(ctx, t)
	if err != nil {
		t.Fatal(err)
		return
	}
	defer close()

	httpClient := &http.Client{}
	resp, err := httpClient.Get("http://localhost:8000/ping")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 200, got %d", resp.StatusCode)
	}
}
