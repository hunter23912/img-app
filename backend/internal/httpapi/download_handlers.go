package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"img-app/backend/internal/history"
	"img-app/backend/internal/imageops"
)

const (
	defaultDownloadQuality = 95
	maxDownloadRequestSize = 64 * 1024 * 1024
)

func downloadImageHandler(config appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxDownloadRequestSize)
		var input downloadImageRequest
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
			return
		}
		if err := ensureJSONBodyEnded(decoder); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
			return
		}

		format := strings.ToLower(strings.TrimSpace(input.Format))
		if format != "png" && format != "jpg" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "format must be png or jpg"})
			return
		}

		quality := defaultDownloadQuality
		if input.Quality != nil {
			quality = *input.Quality
		}
		if quality < 1 || quality > 100 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "quality must be between 1 and 100"})
			return
		}

		source := strings.TrimSpace(input.Source)
		if source == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "source is required"})
			return
		}

		imageBytes, err := loadDownloadImageForConfig(source, config)
		if err != nil {
			slog.Warn("image download rejected", "error", err)
			status := http.StatusBadRequest
			if !imageops.IsSourceValidationError(err) {
				status = http.StatusBadGateway
			}
			writeJSON(w, status, errorResponse{Error: err.Error()})
			return
		}

		output := imageBytes
		contentType, extension, err := imageops.DetectType(imageBytes)
		if err != nil {
			writeJSON(w, http.StatusUnsupportedMediaType, errorResponse{Error: err.Error()})
			return
		}

		if format == "jpg" {
			output, err = imageops.EncodeJPEG(imageBytes, quality)
			if err != nil {
				writeJSON(w, http.StatusUnsupportedMediaType, errorResponse{Error: err.Error()})
				return
			}
			contentType = "image/jpeg"
			extension = "jpg"
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="gpt-image.%s"`, extension))
		w.Header().Set("Content-Length", strconv.Itoa(len(output)))
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(output); err != nil {
			return
		}
	}
}

func loadDownloadImageForConfig(source string, config appConfig) ([]byte, error) {
	if id, ok := history.ParseImagePath(source); ok {
		if config.Database == nil {
			return nil, fmt.Errorf("history image storage is unavailable")
		}
		image, err := config.Database.HistoryImageData(id)
		if err != nil {
			if err == errNotFound {
				return nil, fmt.Errorf("history image not found")
			}
			return nil, err
		}
		return imageops.DecodeDataURL(image)
	}
	return imageops.LoadImage(source, config.ImageSourceRegistry)
}

func ensureJSONBodyEnded(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple json values")
		}
		return err
	}
	return nil
}
