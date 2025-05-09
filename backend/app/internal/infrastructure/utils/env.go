package utils

import (
	"os"
)

// GetEnv is a helper to read an environment variable or return a default value
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GetEnvMode() string {
	return GetEnv("ENV_MODE", "dev")
}