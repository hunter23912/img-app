package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultEndpoint = "https://img-cn.65535.space"
	defaultModel    = "gpt-image-2"
)

type healthResponse struct {
	OK         bool `json:"ok"`
	Configured bool `json:"configured"`
}

type generateRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Size    string `json:"size"`
	Quality string `json:"quality"`
}

type appConfig struct {
	Endpoint string
	APIKey   string
	Addr     string
}

type relayGenerateRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	N              int    `json:"n"`
	ResponseFormat string `json:"response_format,omitempty"`
}

type relayImageResponse struct {
	Created int64 `json:"created,omitempty"`
	Data    []struct {
		URL     string `json:"url,omitempty"`
		B64JSON string `json:"b64_json,omitempty"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message,omitempty"`
		Type    string `json:"type,omitempty"`
	} `json:"error,omitempty"`
}

type asyncJobResponse struct {
	JobID     string `json:"job_id,omitempty"`
	Status    string `json:"status,omitempty"`
	StatusURL string `json:"status_url,omitempty"`
}

type imageResponse struct {
	Image string `json:"image"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func main() {
	config := loadConfig()

	// NewServeMux 是 Go 标准库里的路由器，用来把不同 URL 分发给不同处理函数。
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", healthHandler(config))
	mux.HandleFunc("/api/generate", generateHandler(config))
	mux.HandleFunc("/api/edit", editHandler(config))

	log.Printf("backend starting")
	log.Printf("listen addr: %s", config.Addr)
	log.Printf("image endpoint: %s", config.Endpoint)
	log.Printf("api key configured: %t", config.APIKey != "")
	if config.APIKey == "" {
		log.Printf("warning: IMG_API_KEY is empty; image requests will fail until it is set")
	}

	// 开发阶段让后端监听 8080 端口；withCORS 用来允许 Vite 前端访问这个服务。
	if err := http.ListenAndServe(config.Addr, withRequestLog(withCORS(mux))); err != nil {
		log.Fatal(err)
	}
}

func loadConfig() appConfig {
	loadDotEnv()

	endpoint := strings.TrimSpace(os.Getenv("IMG_ENDPOINT"))
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	addr := strings.TrimSpace(os.Getenv("APP_ADDR"))
	if addr == "" {
		addr = "localhost:8080"
	}

	return appConfig{
		Endpoint: endpoint,
		APIKey:   strings.TrimSpace(os.Getenv("IMG_API_KEY")),
		Addr:     addr,
	}
}

func loadDotEnv() {
	for _, path := range []string{".env", filepath.Join("backend", ".env")} {
		if err := applyDotEnvFile(path); err == nil {
			log.Printf("loaded config from %s", path)
			return
		}
	}
}

func applyDotEnvFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)

		if key == "" {
			continue
		}

		// 已经在系统环境变量里显式设置的值优先级更高。
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return nil
}

func healthHandler(config appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 这个接口只接受 GET 请求，后续真实业务接口会使用 POST。
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method not allowed",
			})
			return
		}

		// 健康检查接口用于确认 Go 后端已经正常启动，同时告诉前端密钥是否已配置。
		writeJSON(w, http.StatusOK, healthResponse{
			OK:         true,
			Configured: config.APIKey != "",
		})
	}
}

func generateHandler(config appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 文生图接口只接受 POST，密钥从后端环境变量读取，不再由前端传入。
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
			return
		}

		if config.APIKey == "" {
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
		// 图编辑接口接收 multipart/form-data，然后转发给中转站的 /v1/images/edits。
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

func normalizeImageRequest(input *generateRequest) {
	input.Model = strings.TrimSpace(input.Model)
	input.Prompt = strings.TrimSpace(input.Prompt)
	input.Size = strings.TrimSpace(input.Size)
	input.Quality = strings.TrimSpace(input.Quality)

	if input.Model == "" {
		input.Model = defaultModel
	}
	if input.Quality == "" {
		input.Quality = "auto"
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 允许本地 Vite 开发服务器访问 Go 后端。
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// 浏览器在跨域 POST 前可能会先发 OPTIONS 预检请求，这里直接返回通过。
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(data)
	r.bytes += n
	return n, err
}

func withRequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &responseRecorder{ResponseWriter: w}

		next.ServeHTTP(recorder, r)

		statusCode := recorder.statusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		log.Printf(
			"http request: method=%s path=%s status=%d bytes=%d duration_ms=%d client_ip=%s user_agent=%q",
			r.Method,
			r.URL.RequestURI(),
			statusCode,
			recorder.bytes,
			time.Since(start).Milliseconds(),
			clientIP(r),
			r.UserAgent(),
		)
	})
}

func clientIP(r *http.Request) string {
	if forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwardedFor != "" {
		ip, _, _ := strings.Cut(forwardedFor, ",")
		if ip = strings.TrimSpace(ip); ip != "" {
			return ip
		}
	}

	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}

	host, _, err := strings.Cut(r.RemoteAddr, ":")
	if err {
		return host
	}

	return r.RemoteAddr
}

func buildImagesURL(endpoint string, action string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("endpoint must be a valid http or https url")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("endpoint must use http or https")
	}

	path := strings.TrimRight(parsed.Path, "/")
	expectedSuffix := "/images/" + action

	// 用户可以输入 Base URL、/v1，或直接输入完整的 /v1/images/generations 地址。
	if strings.HasSuffix(path, expectedSuffix) {
		return parsed.String(), nil
	}

	if path == "" {
		parsed.Path = "/v1" + expectedSuffix
	} else {
		parsed.Path = path + expectedSuffix
	}

	return parsed.String(), nil
}

func callRelayGenerate(generateURL string, apiKey string, input generateRequest) (string, error) {
	payload := relayGenerateRequest{
		Model:          input.Model,
		Prompt:         input.Prompt,
		Size:           input.Size,
		Quality:        input.Quality,
		N:              1,
		ResponseFormat: "url",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode relay request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, generateURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create relay request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 330 * time.Second}
	start := time.Now()
	log.Printf("relay generate call: url=%s model=%q size=%q quality=%q", generateURL, input.Model, input.Size, input.Quality)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("relay generate network error: duration_ms=%d error=%v", time.Since(start).Milliseconds(), err)
		return "", fmt.Errorf("call relay: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return "", fmt.Errorf("read relay response: %w", err)
	}
	log.Printf(
		"relay generate response: status=%d duration_ms=%d content_type=%q bytes=%d",
		resp.StatusCode,
		time.Since(start).Milliseconds(),
		resp.Header.Get("Content-Type"),
		len(respBody),
	)

	var relayResp relayImageResponse
	if err := json.Unmarshal(respBody, &relayResp); err != nil {
		return "", nonJSONResponseError(resp, respBody)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", relayError(resp.StatusCode, relayResp)
	}

	if resp.StatusCode == http.StatusAccepted {
		var job asyncJobResponse
		if err := json.Unmarshal(respBody, &job); err == nil && job.JobID != "" {
			return "", fmt.Errorf("image job %s is %s; sync mode was expected", job.JobID, job.Status)
		}
		return "", fmt.Errorf("relay returned async job; sync image data was expected")
	}

	return firstImage(relayResp)
}

func callRelayEdit(
	editURL string,
	apiKey string,
	input generateRequest,
	imageFile multipart.File,
	imageHeader *multipart.FileHeader,
	maskFile multipart.File,
	maskHeader *multipart.FileHeader,
) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("model", input.Model); err != nil {
		return "", fmt.Errorf("write model field: %w", err)
	}
	if err := writer.WriteField("prompt", input.Prompt); err != nil {
		return "", fmt.Errorf("write prompt field: %w", err)
	}
	if err := writer.WriteField("size", input.Size); err != nil {
		return "", fmt.Errorf("write size field: %w", err)
	}
	if err := writer.WriteField("quality", input.Quality); err != nil {
		return "", fmt.Errorf("write quality field: %w", err)
	}
	if err := writer.WriteField("response_format", "url"); err != nil {
		return "", fmt.Errorf("write response_format field: %w", err)
	}

	if err := copyMultipartFile(writer, "image", imageHeader.Filename, imageFile); err != nil {
		return "", err
	}
	if maskFile != nil && maskHeader != nil {
		if err := copyMultipartFile(writer, "mask", maskHeader.Filename, maskFile); err != nil {
			return "", err
		}
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, editURL, &body)
	if err != nil {
		return "", fmt.Errorf("create relay edit request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 330 * time.Second}
	start := time.Now()
	log.Printf("relay edit call: url=%s model=%q size=%q quality=%q image=%q mask=%t", editURL, input.Model, input.Size, input.Quality, imageHeader.Filename, maskFile != nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("relay edit network error: duration_ms=%d error=%v", time.Since(start).Milliseconds(), err)
		return "", fmt.Errorf("call relay edit: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return "", fmt.Errorf("read relay edit response: %w", err)
	}
	log.Printf(
		"relay edit response: status=%d duration_ms=%d content_type=%q bytes=%d",
		resp.StatusCode,
		time.Since(start).Milliseconds(),
		resp.Header.Get("Content-Type"),
		len(respBody),
	)

	var relayResp relayImageResponse
	if err := json.Unmarshal(respBody, &relayResp); err != nil {
		return "", nonJSONResponseError(resp, respBody)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", relayError(resp.StatusCode, relayResp)
	}

	return firstImage(relayResp)
}

func copyMultipartFile(writer *multipart.Writer, field string, filename string, file multipart.File) error {
	contentType := imageContentType(filename)
	if contentType == "" {
		return fmt.Errorf("%s must be a png, jpeg, webp, or gif image", field)
	}

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, field, filename))
	header.Set("Content-Type", contentType)

	part, err := writer.CreatePart(header)
	if err != nil {
		return fmt.Errorf("create %s form file: %w", field, err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("copy %s form file: %w", field, err)
	}

	return nil
}

func imageContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	if contentType := mime.TypeByExtension(ext); strings.HasPrefix(contentType, "image/") {
		return contentType
	}

	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return ""
	}
}

func relayError(statusCode int, relayResp relayImageResponse) error {
	if relayResp.Error != nil && relayResp.Error.Message != "" {
		return errors.New(relayResp.Error.Message)
	}

	switch statusCode {
	case http.StatusForbidden:
		return fmt.Errorf("relay returned 403: API key has no image permission or is invalid")
	case http.StatusTooManyRequests:
		return fmt.Errorf("relay returned 429: rate limit or concurrent task limit exceeded")
	case http.StatusGatewayTimeout:
		return fmt.Errorf("relay returned 504: sync wait timed out; the task may still be running")
	case http.StatusServiceUnavailable:
		return fmt.Errorf("relay returned 503: service unavailable or queue is too long")
	default:
		return fmt.Errorf("relay returned status %d", statusCode)
	}
}

func nonJSONResponseError(resp *http.Response, body []byte) error {
	contentType := resp.Header.Get("Content-Type")
	snippet := strings.TrimSpace(string(body))
	if len(snippet) > 300 {
		snippet = snippet[:300] + "..."
	}
	if snippet == "" {
		snippet = "<empty body>"
	}

	return fmt.Errorf("relay returned non-json response: status %d, content-type %q, body %q", resp.StatusCode, contentType, snippet)
}

func firstImage(relayResp relayImageResponse) (string, error) {
	if len(relayResp.Data) == 0 {
		return "", fmt.Errorf("relay response has no image data")
	}

	first := relayResp.Data[0]
	if first.B64JSON != "" {
		return "data:image/png;base64," + first.B64JSON, nil
	}
	if first.URL != "" {
		return first.URL, nil
	}

	return "", fmt.Errorf("relay response has no image url or b64_json")
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	// 统一用 JSON 响应，前端 fetch 后可以直接 response.json() 解析。
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write json response: %v", err)
	}
}
