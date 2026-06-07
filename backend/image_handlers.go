package main

import (
	"encoding/json"
	"log"
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
		if input.Size == "" {
			input.Size = "1024x1024"
		}

		generateURL, err := buildImagesURL(config.Endpoint, "generations")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		log.Printf("image generate request: model=%q size=%q quality=%q prompt_chars=%d", input.Model, input.Size, input.Quality, len(input.Prompt))
		image, err := callRelayGenerate(generateURL, config.APIKey, input)
		if err != nil {
			log.Printf("image generate failed: %v", err)
			writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
			return
		}

		log.Printf("image generate succeeded: model=%q size=%q", input.Model, input.Size)
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

		log.Printf("image edit request: model=%q size=%q quality=%q prompt_chars=%d image=%q mask=%t", input.Model, input.Size, input.Quality, len(input.Prompt), imageHeader.Filename, maskFile != nil)
		image, err := callRelayEdit(editURL, config.APIKey, input, imageFile, imageHeader, maskFile, maskHeader)
		if err != nil {
			log.Printf("image edit failed: %v", err)
			writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
			return
		}

		log.Printf("image edit succeeded: model=%q size=%q", input.Model, input.Size)
		writeJSON(w, http.StatusOK, imageResponse{Image: image})
	}
}
