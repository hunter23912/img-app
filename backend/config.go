package main

import (
	"fmt"
	"os"
	"strings"
)

const (
	defaultEndpoint = "https://task-api-1-cn.65535.space"
	defaultModel    = "gpt-image-2-lite"
	defaultSize     = "1152x2048"
)

type appConfig struct {
	Endpoint            string
	APIKey              string
	Addr                string
	ImageSourceRegistry *imageSourceRegistry
	ImageHistory        *imageHistory
	Database            *appDatabase
}

func loadConfig() (appConfig, error) {
	endpoint := strings.TrimSpace(os.Getenv("IMG_ENDPOINT"))
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	addr := strings.TrimSpace(os.Getenv("APP_ADDR"))
	if addr == "" {
		addr = "localhost:8080"
	}

	apiKey := strings.TrimSpace(os.Getenv("IMG_API_KEY"))
	if apiKey == "" {
		return appConfig{}, fmt.Errorf("IMG_API_KEY is required")
	}

	return appConfig{
		Endpoint: endpoint,
		APIKey:   apiKey,
		Addr:     addr,
	}, nil
}

func databasePathFromEnvironment() string {
	return strings.TrimSpace(os.Getenv("APP_DB_PATH"))
}
