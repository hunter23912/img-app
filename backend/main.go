package main

import (
	"log/slog"
	"net/http"
	"os"
)

func main() {
	initLogger()

	config, err := loadConfig()
	if err != nil {
		slog.Error("load config failed", "error", err)
		os.Exit(1)
	}
	config.ImageSourceRegistry = newImageSourceRegistry()
	config.ImageHistory = newImageHistory()
	database, err := openDatabase(databasePathFromEnvironment())
	if err != nil {
		slog.Error("open database failed", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	config.Database = database
	if sources, err := database.listImageSources(); err != nil {
		slog.Error("restore image sources failed", "error", err)
		os.Exit(1)
	} else {
		for _, source := range sources {
			config.ImageSourceRegistry.Trust(source)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", healthHandler(config))
	mux.HandleFunc("/api/history", historyHandler(config))
	mux.HandleFunc("/api/history/", historyHandler(config))
	mux.HandleFunc("/api/presets", presetsHandler(config))
	mux.HandleFunc("/api/presets/", presetsHandler(config))
	mux.HandleFunc("/api/generate", generateHandler(config))
	mux.HandleFunc("/api/edit", editHandler(config))
	mux.HandleFunc("/api/download/image", downloadImageHandler(config))

	slog.Info("backend starting",
		"addr", config.Addr,
		"image_endpoint", config.Endpoint,
		"api_key_configured", config.APIKey != "",
	)
	if config.APIKey == "" {
		slog.Warn("image API key is not configured", "env", "IMG_API_KEY")
	}

	if err := http.ListenAndServe(config.Addr, withRequestLog(withCORS(mux))); err != nil {
		slog.Error("backend stopped", "error", err)
		os.Exit(1)
	}
}
