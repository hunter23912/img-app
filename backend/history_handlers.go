package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func historyHandler(config appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.Database != nil {
			databaseHistoryHandler(w, r, config.Database)
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{"images": config.ImageHistory.List()})
		case http.MethodDelete:
			deleteHistoryImage(w, r, config.ImageHistory)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		}
	}
}

func databaseHistoryHandler(w http.ResponseWriter, r *http.Request, database *appDatabase) {
	pathID := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/api/history"), "/")
	switch r.Method {
	case http.MethodGet:
		limit := 5
		if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
			parsed, err := strconv.Atoi(rawLimit)
			if err != nil || parsed < 1 || parsed > 5 {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: "limit must be between 1 and 5"})
				return
			}
			limit = parsed
		}
		page, err := database.listTasks(limit, strings.TrimSpace(r.URL.Query().Get("cursor")))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, page)
	case http.MethodDelete:
		if pathID == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "task id is required"})
			return
		}
		if err := database.deleteTask(pathID); err != nil {
			if err == errNotFound {
				writeJSON(w, http.StatusNotFound, errorResponse{Error: "history task not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "history deletion failed"})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
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
