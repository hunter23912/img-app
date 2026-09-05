package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"img-app/backend/internal/history"
	"img-app/backend/internal/provider"
	"img-app/backend/internal/store"
)

const (
	maxInputImageBytes    = 50 * 1024 * 1024
	maxEditMultipartBytes = maxInputImageBytes + 2*1024*1024
	maxEditImages         = 4
)

var imageReferencePattern = regexp.MustCompile(`@([0-9]+)`)

func healthHandler(config appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
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

		settings, input, ok := resolveGenerateInput(w, r, config)
		if !ok {
			return
		}
		generateURL, err := provider.BuildImagesURL(settings.Endpoint, "generations")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		stream := wantsImageStream(r)
		w, stopHeartbeat := imageStreamHeartbeat(w, stream)
		defer stopHeartbeat()
		if stream {
			writeImageEvent(w, "image_generate.started", "", "")
		}
		onEvent := imageEventCallback(w, stream, "image_generate")
		slog.Info("image generate requested", "model", input.Model, "size", input.Size, "quality", input.Quality, "prompt_chars", len(input.Prompt))
		image, err := provider.CallGenerate(r.Context(), generateURL, settings.APIKey, provider.ImageRequest{
			Model:        input.Model,
			Prompt:       input.Prompt,
			Size:         input.Size,
			Quality:      input.Quality,
			Moderation:   input.Moderation,
			Background:   input.Background,
			OutputFormat: input.OutputFormat,
			N:            input.N,
		}, onEvent)
		if err != nil {
			slog.Error("image generate failed", "error", err)
			if stream {
				writeImageEvent(w, "image_generate.failed", "", err.Error())
				return
			}
			writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
			return
		}

		displayImage := persistImageResult(config, "generate", input, image)
		slog.Info("image generate succeeded", "model", input.Model, "size", input.Size)
		if stream {
			writeImageEvent(w, "image_generate.completed", displayImage, "")
			return
		}
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

		r.Body = http.MaxBytesReader(w, r.Body, maxEditMultipartBytes)
		if err := r.ParseMultipartForm(maxEditMultipartBytes); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid multipart form"})
			return
		}
		defer r.MultipartForm.RemoveAll()
		input := generateRequest{
			Model:        strings.TrimSpace(r.FormValue("model")),
			Prompt:       strings.TrimSpace(r.FormValue("prompt")),
			Size:         strings.TrimSpace(r.FormValue("size")),
			Quality:      strings.TrimSpace(r.FormValue("quality")),
			Moderation:   strings.TrimSpace(r.FormValue("moderation")),
			Background:   strings.TrimSpace(r.FormValue("background")),
			OutputFormat: strings.TrimSpace(r.FormValue("output_format")),
			N:            parsePositiveInt(r.FormValue("n")),
		}
		normalizeImageRequest(&input)
		if !validateImageInput(w, config, input) {
			return
		}
		imageHeaders := r.MultipartForm.File["image"]
		if len(imageHeaders) == 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "image file is required"})
			return
		}
		if len(imageHeaders) > maxEditImages {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "最多上传 " + strconv.Itoa(maxEditImages) + " 张图片"})
			return
		}
		if referenceError := validateImageReferences(input.Prompt, len(imageHeaders)); referenceError != "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: referenceError})
			return
		}
		images := make([]provider.ImageFile, 0, len(imageHeaders))
		for _, imageHeader := range imageHeaders {
			imageFile, openErr := imageHeader.Open()
			if openErr != nil {
				for _, opened := range images {
					_ = opened.File.Close()
				}
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: "无法读取图片文件"})
				return
			}
			images = append(images, provider.ImageFile{File: imageFile, Header: imageHeader})
		}
		defer func() {
			for _, image := range images {
				_ = image.File.Close()
			}
		}()
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
		stream := wantsImageStream(r)
		w, stopHeartbeat := imageStreamHeartbeat(w, stream)
		defer stopHeartbeat()
		if stream {
			writeImageEvent(w, "image_edit.started", "", "")
		}
		onEvent := imageEventCallback(w, stream, "image_edit")
		slog.Info("image edit requested", "model", input.Model, "size", input.Size, "quality", input.Quality, "prompt_chars", len(input.Prompt), "images", len(images), "has_mask", maskFile != nil)
		image, err := provider.CallEditImages(r.Context(), editURL, settings.APIKey, provider.ImageRequest{
			Model:        input.Model,
			Prompt:       input.Prompt,
			Size:         input.Size,
			Quality:      input.Quality,
			Moderation:   input.Moderation,
			Background:   input.Background,
			OutputFormat: input.OutputFormat,
			N:            input.N,
		}, images, maskFile, maskHeader, onEvent)
		if err != nil {
			slog.Error("image edit failed", "error", err)
			if stream {
				writeImageEvent(w, "image_edit.failed", "", err.Error())
				return
			}
			writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
			return
		}

		displayImage := persistImageResult(config, "edit", input, image)
		slog.Info("image edit succeeded", "model", input.Model, "size", input.Size)
		if stream {
			writeImageEvent(w, "image_edit.completed", displayImage, "")
			return
		}
		writeJSON(w, http.StatusOK, imageResponse{Image: displayImage})
	}
}

func resolveGenerateInput(w http.ResponseWriter, r *http.Request, config appConfig) (resolvedImageSettings, generateRequest, bool) {
	settings, err := config.effectiveImageSettings()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load image settings"})
		return resolvedImageSettings{}, generateRequest{}, false
	}
	if settings.APIKey == "" {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "IMG_API_KEY is not configured"})
		return resolvedImageSettings{}, generateRequest{}, false
	}
	var input generateRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return resolvedImageSettings{}, generateRequest{}, false
	}
	normalizeImageRequest(&input)
	if !validateImageInput(w, config, input) {
		return resolvedImageSettings{}, generateRequest{}, false
	}
	return settings, input, true
}

func validateImageInput(w http.ResponseWriter, config appConfig, input generateRequest) bool {
	available, err := config.imageModelAvailable(input.Model)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to validate image model"})
		return false
	}
	if !available {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "模型不可用，请重新选择模型"})
		return false
	}
	if input.Prompt == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "prompt is required"})
		return false
	}
	return true
}

func validateImageReferences(prompt string, imageCount int) string {
	for _, indexes := range imageReferencePattern.FindAllStringSubmatchIndex(prompt, -1) {
		start, end := indexes[0], indexes[1]
		if start > 0 && isImageReferenceWordCharacter(prompt[start-1]) {
			continue
		}
		if end < len(prompt) && isImageReferenceWordCharacter(prompt[end]) {
			continue
		}
		numberText := prompt[indexes[2]:indexes[3]]
		number, err := strconv.Atoi(numberText)
		if err != nil || number < 1 || number > imageCount {
			return "图片引用 @" + numberText + " 无效，请使用 @1 到 @" + strconv.Itoa(imageCount) + "。"
		}
	}
	return ""
}

func isImageReferenceWordCharacter(value byte) bool {
	return (value >= 'A' && value <= 'Z') || (value >= 'a' && value <= 'z') || (value >= '0' && value <= '9') || value == '_'
}

func persistImageResult(config appConfig, mode string, input generateRequest, image string) string {
	source := image
	displayImage := image
	if config.Database != nil {
		taskID, err := config.Database.CreateTask(mode, store.ImageTaskInput{
			Model: input.Model, Prompt: input.Prompt, Size: input.Size, Quality: input.Quality,
		})
		if err != nil {
			slog.Error("create image task failed", "error", err)
		} else if err := config.Database.CompleteTask(taskID, image); err != nil {
			slog.Error("mark image task succeeded", "task_id", taskID, "error", err)
		} else {
			displayImage = history.ImageReference(taskID, image)
			if strings.HasPrefix(source, "https://") {
				// 远程原图可能很大或传输很慢，不能阻塞生成完成事件；后台缓存成功后历史路径会自动改为本地数据。
				go func() {
					if _, err := loadCachedImage(context.Background(), config.Database, source); err != nil {
						slog.Warn("generated image background cache failed; keeping source URL", "task_id", taskID, "error", err)
						return
					}
					slog.Info("generated image background cache complete", "task_id", taskID)
				}()
			}
		}
	}
	config.ImageSourceRegistry.Trust(displayImage)
	if config.Database == nil {
		config.ImageHistory.Add(image)
	}
	return displayImage
}

func wantsImageStream(r *http.Request) bool {
	return !strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("stream")), "false")
}

type imageStreamWriter struct {
	http.ResponseWriter
	mu sync.Mutex
}

func (w *imageStreamWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.ResponseWriter.Write(data)
}

func (w *imageStreamWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func imageStreamHeartbeat(w http.ResponseWriter, enabled bool) (http.ResponseWriter, func()) {
	if !enabled {
		return w, func() {}
	}
	writer := &imageStreamWriter{ResponseWriter: w}
	// 先发送响应头，再启动心跳，避免并发修改 Header。
	startImageStream(writer)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if _, err := writer.Write([]byte(": waiting\n\n")); err != nil {
					return
				}
				writer.Flush()
			}
		}
	}()
	return writer, func() { close(stop); <-done }
}

func startImageStream(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
}

func writeImageEvent(w http.ResponseWriter, eventType, image, message string) {
	event := map[string]string{"type": eventType}
	if image != "" {
		event["image"] = image
	}
	if message != "" {
		event["error"] = message
	}
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	if _, err := w.Write([]byte("data: " + string(data) + "\n\n")); err != nil {
		return
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func imageEventCallback(w http.ResponseWriter, stream bool, operation string) provider.ImageEventHandler {
	if !stream {
		return nil
	}
	return func(event provider.ImageEvent) {
		if event.Image != "" {
			writeImageEvent(w, operation+".partial_image", event.Image, "")
		}
	}
}
