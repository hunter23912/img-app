package main

import (
	"log/slog"
	"net/http"
	"os"

	"img-app/backend/internal/config"
	"img-app/backend/internal/history"
	"img-app/backend/internal/httpapi"
	"img-app/backend/internal/imageops"
	"img-app/backend/internal/logging"
	"img-app/backend/internal/store"
)

func main() {
	logging.Init()

	runtimeConfig, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "error", err)
		os.Exit(1)
	}
	database, err := store.OpenDatabase(config.DatabasePath(runtimeConfig))
	if err != nil {
		slog.Error("open database failed", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	imageSources := imageops.NewSourceRegistry()
	if sources, err := database.ListImageSources(); err != nil {
		slog.Error("restore image sources failed", "error", err)
		os.Exit(1)
	} else {
		for _, source := range sources {
			imageSources.Trust(source)
		}
	}

	handlerConfig := httpapi.ServerConfig{
		Endpoint:            runtimeConfig.Endpoint,
		APIKey:              runtimeConfig.APIKey,
		Addr:                runtimeConfig.Addr,
		ImageSourceRegistry: imageSources,
		ImageHistory:        history.New(),
		Database:            database,
	}

	settings, settingsErr := handlerConfig.EffectiveImageSettings()
	if settingsErr != nil {
		slog.Error("load image settings failed", "error", settingsErr)
		os.Exit(1)
	}
	slog.Info("backend starting",
		"addr", runtimeConfig.Addr,
		"image_endpoint", settings.Endpoint,
		"api_key_configured", settings.APIKey != "",
	)
	if settings.APIKey == "" {
		slog.Warn("image API key is not configured", "env", "IMG_API_KEY")
	}

	if err := http.ListenAndServe(runtimeConfig.Addr, httpapi.NewHandler(handlerConfig)); err != nil {
		slog.Error("backend stopped", "error", err)
		os.Exit(1)
	}
}
