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
		if imageID, ok := parseHistoryImagePath(r.URL.Path); ok {
			serveStoredHistoryImage(w, database, imageID)
			return
		}
		if pathID != "" {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "history image not found"})
			return
		}
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

func serveStoredHistoryImage(w http.ResponseWriter, database *appDatabase, id string) {
	image, err := database.historyImageData(id)
	if err != nil {
		if err == errNotFound {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "history image not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "history image read failed"})
		return
	}

	imageBytes, err := decodeImageDataURL(image)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "history image data is invalid"})
		return
	}
	contentType, _, err := detectImageType(imageBytes)
	if err != nil {
		writeJSON(w, http.StatusUnsupportedMediaType, errorResponse{Error: err.Error()})
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	w.Header().Set("Content-Length", strconv.Itoa(len(imageBytes)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(imageBytes)
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
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "image must be a valid HTTPS URL or image data URL"})
		return
	}

	history.Remove(image)
	w.WriteHeader(http.StatusNoContent)
}
