package main

import (
	"encoding/json"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strings"
)

func healthHandler(config appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method not allowed",
			})
			return
		}

		writeJSON(w, http.StatusOK, healthResponse{
			OK:         true,
			Configured: config.APIKey != "",
		})
	}
}

func generateHandler(config appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
			return
		}

		if config.APIKey == "" {
			// API key 只存在后端环境变量中，前端不会也不应该直接持有密钥。
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "IMG_API_KEY is not configured"})
			return
		}

		var input generateRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
			return
		}

		normalizeImageRequest(&input)
		if input.Prompt == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "prompt is required"})
			return
		}
		generateURL, err := buildImagesURL(config.Endpoint, "generations")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		slog.Info("image generate requested", "model", input.Model, "size", input.Size, "quality", input.Quality, "prompt_chars", len(input.Prompt))
		image, err := callRelayGenerate(generateURL, config.APIKey, input)
		if err != nil {
			slog.Error("image generate failed", "error", err)
			writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
			return
		}
		config.ImageSourceRegistry.Trust(image)
		config.ImageHistory.Add(image)

		slog.Info("image generate succeeded", "model", input.Model, "size", input.Size)
		writeJSON(w, http.StatusOK, imageResponse{Image: image})
	}
}

func editHandler(config appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
			return
		}

		if config.APIKey == "" {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "IMG_API_KEY is not configured"})
			return
		}

		if err := r.ParseMultipartForm(128 * 1024 * 1024); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid multipart form"})
			return
		}

		input := generateRequest{
			Model:   strings.TrimSpace(r.FormValue("model")),
			Prompt:  strings.TrimSpace(r.FormValue("prompt")),
			Size:    strings.TrimSpace(r.FormValue("size")),
			Quality: strings.TrimSpace(r.FormValue("quality")),
		}

		normalizeImageRequest(&input)
		if input.Prompt == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "prompt is required"})
			return
		}

		imageFile, imageHeader, err := r.FormFile("image")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "image file is required"})
			return
		}
		defer imageFile.Close()

		var maskFile multipart.File
		var maskHeader *multipart.FileHeader
		maskFile, maskHeader, _ = r.FormFile("mask")
		if maskFile != nil {
			defer maskFile.Close()
		}

		editURL, err := buildImagesURL(config.Endpoint, "edits")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		slog.Info("image edit requested", "model", input.Model, "size", input.Size, "quality", input.Quality, "prompt_chars", len(input.Prompt), "image", imageHeader.Filename, "has_mask", maskFile != nil)
		image, err := callRelayEdit(editURL, config.APIKey, input, imageFile, imageHeader, maskFile, maskHeader)
		if err != nil {
			slog.Error("image edit failed", "error", err)
			writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
			return
		}
		config.ImageSourceRegistry.Trust(image)
		config.ImageHistory.Add(image)

		slog.Info("image edit succeeded", "model", input.Model, "size", input.Size)
		writeJSON(w, http.StatusOK, imageResponse{Image: image})
	}
}
