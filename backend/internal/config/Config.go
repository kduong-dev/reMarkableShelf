package config

import (
	"os"

	"github.com/kqvd/reMarkableShelf/backend/internal/fatal"
)

func EnvString(key string, dflt string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return dflt
}

func EnvStringOrFatal(key string) string {
	value := os.Getenv(key)
	if value == "" {
		fatal.LogErrorf("The following environment variable must be set: %s", key)
	}
	return value
}
