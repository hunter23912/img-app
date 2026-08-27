package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

func presetsHandler(config appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.Database == nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "database is not initialized"})
			return
		}
		id := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/api/presets"), "/")
		switch {
		case r.Method == http.MethodGet && id == "":
			presets, err := config.Database.ListPresets()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load presets"})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"presets": presets})
		case r.Method == http.MethodPost && id == "":
			var draft promptPresetDraft
			if !decodeJSONBody(w, r, &draft) {
				return
			}
			preset, err := config.Database.CreatePreset(draft)
			if err != nil {
				writePresetError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, preset)
		case r.Method == http.MethodPost && id == "import":
			var payload struct {
				Presets []promptPresetImport `json:"presets"`
			}
			if !decodeJSONBody(w, r, &payload) {
				return
			}
			count, err := config.Database.ImportPresets(payload.Presets)
			if err != nil {
				writePresetError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"imported": count})
		case r.Method == http.MethodPut && id != "":
			var draft promptPresetDraft
			if !decodeJSONBody(w, r, &draft) {
				return
			}
			preset, err := config.Database.UpdatePreset(id, draft)
			if err != nil {
				writePresetError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, preset)
		case r.Method == http.MethodDelete && id != "":
			if err := config.Database.DeletePreset(id); err != nil {
				writePresetError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		}
	}
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, destination any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(destination); err != nil || ensureJSONBodyEnded(decoder) != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return false
	}
	return true
}

func writePresetError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, errNotFound) {
		status = http.StatusNotFound
	}
	writeJSON(w, status, errorResponse{Error: err.Error()})
}
