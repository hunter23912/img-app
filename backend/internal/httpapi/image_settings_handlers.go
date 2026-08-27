package httpapi

import (
	"net/http"
	"strings"

	"img-app/backend/internal/store"
)

func imageSettingsHandler(config appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.Database == nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "database is not initialized"})
			return
		}

		switch r.Method {
		case http.MethodGet:
			settings, err := config.effectiveImageSettings()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load image settings"})
				return
			}
			writeJSON(w, http.StatusOK, imageSettingsResponse{
				Endpoint: settings.Endpoint,
				APIKey:   settings.APIKey,
			})
		case http.MethodPut:
			var input imageSettingsRequest
			if !decodeJSONBody(w, r, &input) {
				return
			}
			input.Endpoint = strings.TrimSpace(input.Endpoint)
			input.APIKey = strings.TrimSpace(input.APIKey)
			if input.Endpoint != "" {
				if err := store.ValidateImageEndpoint(input.Endpoint); err != nil {
					writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
					return
				}
			}

			if _, err := config.Database.SaveImageSettings(input.Endpoint, input.APIKey); err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to save image settings"})
				return
			}
			settings, err := config.effectiveImageSettings()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load image settings"})
				return
			}
			writeJSON(w, http.StatusOK, imageSettingsResponse{
				Endpoint: settings.Endpoint,
				APIKey:   settings.APIKey,
			})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		}
	}
}
