package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func historyHandler(config appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, imageHistoryResponse{Images: config.ImageHistory.List()})
		case http.MethodDelete:
			deleteHistoryImage(w, r, config.ImageHistory)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		}
	}
}

func deleteHistoryImage(w http.ResponseWriter, r *http.Request, history *imageHistory) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var input historyDeleteRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}
	if err := ensureJSONBodyEnded(decoder); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}

	image := strings.TrimSpace(input.Image)
	if _, ok := normalizeHistoryImageURL(image); !ok {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "image must be a valid HTTPS URL"})
		return
	}

	history.Remove(image)
	w.WriteHeader(http.StatusNoContent)
}
