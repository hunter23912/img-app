package httpapi

import "net/http"

// NewHandler builds the application's HTTP handler. Dependency construction
// stays in the executable package; this package owns API routes and middleware.
func NewHandler(config ServerConfig) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", healthHandler(config))
	mux.HandleFunc("/api/settings/image", imageSettingsHandler(config))
	mux.HandleFunc("/api/image-profiles", imageProfilesHandler(config))
	mux.HandleFunc("/api/image-profiles/", imageProfilesHandler(config))
	mux.HandleFunc("/api/models", modelsHandler(config))
	mux.HandleFunc("/api/models/", modelsHandler(config))
	mux.HandleFunc("/api/history", historyHandler(config))
	mux.HandleFunc("/api/history/", historyHandler(config))
	mux.HandleFunc("/api/presets", presetsHandler(config))
	mux.HandleFunc("/api/presets/", presetsHandler(config))
	mux.HandleFunc("/api/generate", generateHandler(config))
	mux.HandleFunc("/api/edit", editHandler(config))
	mux.HandleFunc("/api/download/image", downloadImageHandler(config))
	return withRequestLog(withCORS(mux))
}
