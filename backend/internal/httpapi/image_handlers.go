package httpapi

import (
	"encoding/json"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strings"

	"img-app/backend/internal/history"
	"img-app/backend/internal/provider"
	"img-app/backend/internal/store"
)

func healthHandler(config appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method not allowed",
			})
			return
		}

		settings, err := config.effectiveImageSettings()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load image settings"})
			return
		}
		writeJSON(w, http.StatusOK, healthResponse{OK: true, Configured: settings.APIKey != ""})
	}
}

func generateHandler(config appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
			return
		}

		settings, err := config.effectiveImageSettings()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load image settings"})
			return
		}
		if settings.APIKey == "" {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "IMG_API_KEY is not configured"})
			return
		}

		var input generateRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
			return
		}

		normalizeImageRequest(&input)
		available, availabilityErr := config.imageModelAvailable(input.Model)
		if availabilityErr != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to validate image model"})
			return
		}
		if !available {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "模型不可用，请重新选择模型"})
			return
		}
		if input.Prompt == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "prompt is required"})
			return
		}
		generateURL, err := provider.BuildImagesURL(settings.Endpoint, "generations")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		slog.Info("image generate requested", "model", input.Model, "size", input.Size, "quality", input.Quality, "prompt_chars", len(input.Prompt))
		image, err := provider.CallGenerate(generateURL, settings.APIKey, provider.ImageRequest(input))
		if err != nil {
			slog.Error("image generate failed", "error", err)
			writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
			return
		}
		displayImage := image
		if config.Database != nil {
			taskID, createErr := config.Database.CreateTask("generate", store.ImageTaskInput(input))
			if createErr != nil {
				slog.Error("create image task failed", "error", createErr)
			} else if completeErr := config.Database.CompleteTask(taskID, image); completeErr != nil {
				slog.Error("mark image task succeeded", "task_id", taskID, "error", completeErr)
			} else {
				displayImage = history.ImageReference(taskID, image)
			}
		}
		config.ImageSourceRegistry.Trust(displayImage)
		if config.Database == nil {
			config.ImageHistory.Add(image)
		}

		slog.Info("image generate succeeded", "model", input.Model, "size", input.Size)
		writeJSON(w, http.StatusOK, imageResponse{Image: displayImage})
	}
}

func editHandler(config appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
			return
		}

		settings, err := config.effectiveImageSettings()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load image settings"})
			return
		}
		if settings.APIKey == "" {
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
		available, availabilityErr := config.imageModelAvailable(input.Model)
		if availabilityErr != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to validate image model"})
			return
		}
		if !available {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "模型不可用，请重新选择模型"})
			return
		}
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

		editURL, err := provider.BuildImagesURL(settings.Endpoint, "edits")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		slog.Info("image edit requested", "model", input.Model, "size", input.Size, "quality", input.Quality, "prompt_chars", len(input.Prompt), "image", imageHeader.Filename, "has_mask", maskFile != nil)
		image, err := provider.CallEdit(editURL, settings.APIKey, provider.ImageRequest(input), imageFile, imageHeader, maskFile, maskHeader)
		if err != nil {
			slog.Error("image edit failed", "error", err)
			writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
			return
		}
		displayImage := image
		if config.Database != nil {
			taskID, createErr := config.Database.CreateTask("edit", store.ImageTaskInput(input))
			if createErr != nil {
				slog.Error("create image task failed", "error", createErr)
			} else if completeErr := config.Database.CompleteTask(taskID, image); completeErr != nil {
				slog.Error("mark image task succeeded", "task_id", taskID, "error", completeErr)
			} else {
				displayImage = history.ImageReference(taskID, image)
			}
		}
		config.ImageSourceRegistry.Trust(displayImage)
		if config.Database == nil {
			config.ImageHistory.Add(image)
		}

		slog.Info("image edit succeeded", "model", input.Model, "size", input.Size)
		writeJSON(w, http.StatusOK, imageResponse{Image: displayImage})
	}
}
