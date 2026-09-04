package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

const relayResponseBodyLimit = 64 * 1024 * 1024

const (
	maxRelayImageBytes = 48 * 1024 * 1024
	relayImageTimeout  = 45 * time.Second
)

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

	// 支持用户配置 base URL、/v1，或直接配置完整的 /v1/images/<action> 地址。
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
	input = normalizeImageRequest(input)
	payload := relayGenerateRequest{
		Model:        input.Model,
		Prompt:       input.Prompt,
		Size:         input.Size,
		Quality:      input.Quality,
		Moderation:   input.Moderation,
		Background:   input.Background,
		OutputFormat: input.OutputFormat,
		N:            input.N,
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
	slog.Info("relay generate call", "url", generateURL, "model", input.Model, "size", input.Size, "quality", input.Quality)
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("relay generate network error", "duration_ms", time.Since(start).Milliseconds(), "error", err)
		return "", fmt.Errorf("call relay: %w", err)
	}
	defer resp.Body.Close()

	// Decode the JSON value directly. Some compatible relays send a complete
	// JSON response but keep the HTTP connection alive, so waiting for EOF here
	// would make an otherwise completed request hang until Client.Timeout.
	relayResp, responseBytes, err := decodeRelayImageResponse(resp)
	if err != nil {
		return "", err
	}
	slog.Log(context.Background(), relayResponseLevel(resp.StatusCode), "relay generate response",
		"status", resp.StatusCode,
		"duration_ms", time.Since(start).Milliseconds(),
		"content_type", resp.Header.Get("Content-Type"),
		"content_length", resp.ContentLength,
		"transfer_encoding", resp.TransferEncoding,
		"bytes", responseBytes,
	)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", relayError(resp.StatusCode, relayResp)
	}

	if resp.StatusCode == http.StatusAccepted {
		// 当前前端按同步结果设计；如果中转站返回异步任务，先明确报错，后续再接入轮询。
		if relayResp.JobID != "" {
			return "", fmt.Errorf("image job %s is %s; sync mode was expected", relayResp.JobID, relayResp.Status)
		}
		return "", fmt.Errorf("relay returned async job; sync image data was expected")
	}

	return firstImage(relayResp, input.OutputFormat)
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
	input = normalizeImageRequest(input)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// 图编辑接口需要 multipart 转发，后端只做代理，不在这里持久化用户上传的图片。
	if err := writer.WriteField("model", input.Model); err != nil {
		return "", fmt.Errorf("write model field: %w", err)
	}
	if err := writer.WriteField("prompt", input.Prompt); err != nil {
		return "", fmt.Errorf("write prompt field: %w", err)
	}
	if err := writer.WriteField("size", input.Size); err != nil {
		return "", fmt.Errorf("write size field: %w", err)
	}
	if err := writer.WriteField("moderation", input.Moderation); err != nil {
		return "", fmt.Errorf("write moderation field: %w", err)
	}
	if err := writer.WriteField("quality", input.Quality); err != nil {
		return "", fmt.Errorf("write quality field: %w", err)
	}
	if err := writer.WriteField("output_format", input.OutputFormat); err != nil {
		return "", fmt.Errorf("write output_format field: %w", err)
	}
	if input.N > 0 {
		if err := writer.WriteField("n", fmt.Sprint(input.N)); err != nil {
			return "", fmt.Errorf("write n field: %w", err)
		}
	}

	if err := copyMultipartFile(writer, "image[]", imageHeader.Filename, imageFile); err != nil {
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
	slog.Info("relay edit call", "url", editURL, "model", input.Model, "size", input.Size, "quality", input.Quality, "image", imageHeader.Filename, "has_mask", maskFile != nil)
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("relay edit network error", "duration_ms", time.Since(start).Milliseconds(), "error", err)
		return "", fmt.Errorf("call relay edit: %w", err)
	}
	defer resp.Body.Close()

	// Decode the JSON value directly instead of waiting for response EOF. The
	// edit endpoint may leave a keep-alive/chunked response open after sending
	// the complete image JSON.
	relayResp, responseBytes, err := decodeRelayImageResponse(resp)
	if err != nil {
		return "", err
	}
	slog.Log(context.Background(), relayResponseLevel(resp.StatusCode), "relay edit response",
		"status", resp.StatusCode,
		"duration_ms", time.Since(start).Milliseconds(),
		"content_type", resp.Header.Get("Content-Type"),
		"content_length", resp.ContentLength,
		"transfer_encoding", resp.TransferEncoding,
		"bytes", responseBytes,
	)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", relayError(resp.StatusCode, relayResp)
	}

	return firstImage(relayResp, input.OutputFormat)
}

func decodeRelayImageResponse(resp *http.Response) (relayImageResponse, int, error) {
	var relayResp relayImageResponse
	var captured bytes.Buffer
	limitedBody := io.LimitReader(io.TeeReader(resp.Body, &captured), relayResponseBodyLimit)
	var raw json.RawMessage

	if err := json.NewDecoder(limitedBody).Decode(&raw); err != nil {
		// Preserve transport timeout errors so the logs still identify a stalled
		// response body instead of misreporting it as invalid JSON.
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || errors.Is(err, io.ErrUnexpectedEOF) {
			return relayResp, captured.Len(), fmt.Errorf("read relay response: %w", err)
		}
		return relayResp, captured.Len(), nonJSONResponseError(resp, captured.Bytes())
	}

	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &relayResp.Data); err != nil {
			return relayResp, captured.Len(), nonJSONResponseError(resp, captured.Bytes())
		}
	} else if err := json.Unmarshal(trimmed, &relayResp); err != nil {
		return relayResp, captured.Len(), nonJSONResponseError(resp, captured.Bytes())
	}

	return relayResp, captured.Len(), nil
}

func relayResponseLevel(statusCode int) slog.Level {
	if statusCode >= http.StatusInternalServerError {
		return slog.LevelError
	}
	if statusCode >= http.StatusBadRequest {
		return slog.LevelWarn
	}
	return slog.LevelInfo
}

func copyMultipartFile(writer *multipart.Writer, field string, filename string, file multipart.File) error {
	contentType := imageContentType(filename)
	if contentType == "" {
		return fmt.Errorf("%s must be a png, jpeg, webp, or gif image", field)
	}

	header := make(textproto.MIMEHeader)
	// 显式设置上传文件 Content-Type，减少中转站因类型缺失拒绝请求的概率。
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

func normalizeImageRequest(input generateRequest) generateRequest {
	if strings.TrimSpace(input.Model) == "" {
		input.Model = defaultModel
	}
	if strings.TrimSpace(input.Size) == "" {
		input.Size = defaultSize
	}
	if strings.TrimSpace(input.Quality) == "" {
		input.Quality = "auto"
	}
	if strings.TrimSpace(input.Moderation) == "" {
		input.Moderation = "auto"
	}
	if strings.TrimSpace(input.Background) == "" {
		input.Background = "auto"
	}
	if strings.TrimSpace(input.OutputFormat) == "" {
		input.OutputFormat = "png"
	}
	if input.N < 1 {
		input.N = 1
	}
	return input
}

func firstImage(relayResp relayImageResponse, outputFormat string) (string, error) {
	fallbackMime := imageMimeType(outputFormat)
	for _, item := range relayResp.Data {
		if strings.TrimSpace(item.B64JSON) != "" {
			return normalizeBase64Image(item.B64JSON, fallbackMime), nil
		}
		if strings.TrimSpace(item.URL) != "" {
			return normalizeImageValue(item.URL, fallbackMime)
		}
	}
	if strings.TrimSpace(relayResp.Image) != "" {
		return normalizeImageValue(relayResp.Image, fallbackMime)
	}
	for _, image := range relayResp.Images {
		if strings.TrimSpace(image) == "" {
			continue
		}
		return normalizeImageValue(image, fallbackMime)
	}

	return "", fmt.Errorf("relay response has no image url or b64_json")
}

func normalizeImageValue(value string, fallbackMime string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), "data:") {
		return value, nil
	}
	if strings.HasPrefix(strings.ToLower(value), "http://") || strings.HasPrefix(strings.ToLower(value), "https://") {
		return fetchImageURLAsDataURL(value)
	}
	return normalizeBase64Image(value, fallbackMime), nil
}

func normalizeBase64Image(value string, fallbackMime string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), "data:") {
		return value
	}
	return "data:" + fallbackMime + ";base64," + value
}

func imageMimeType(outputFormat string) string {
	switch strings.ToLower(strings.TrimSpace(outputFormat)) {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "webp":
		return "image/webp"
	default:
		return "image/png"
	}
}

func fetchImageURLAsDataURL(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("relay image URL is invalid")
	}
	if parsed.User != nil || parsed.Fragment != "" {
		return "", fmt.Errorf("relay image URL must not contain credentials or fragments")
	}

	request, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create relay image request: %w", err)
	}
	client := &http.Client{Timeout: relayImageTimeout}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("download relay image: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("relay image returned status %d", response.StatusCode)
	}

	imageBytes, err := io.ReadAll(io.LimitReader(response.Body, maxRelayImageBytes+1))
	if err != nil {
		return "", fmt.Errorf("read relay image: %w", err)
	}
	if len(imageBytes) > maxRelayImageBytes {
		return "", fmt.Errorf("relay image is too large")
	}

	mimeType, err := detectImageMime(imageBytes, response.Header.Get("Content-Type"))
	if err != nil {
		return "", err
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(imageBytes), nil
}

func detectImageMime(imageBytes []byte, contentType string) (string, error) {
	if len(imageBytes) >= 12 && string(imageBytes[:4]) == "RIFF" && string(imageBytes[8:12]) == "WEBP" {
		return "image/webp", nil
	}
	if len(imageBytes) >= 8 && bytes.Equal(imageBytes[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}) {
		return "image/png", nil
	}
	if len(imageBytes) >= 3 && bytes.Equal(imageBytes[:3], []byte{0xFF, 0xD8, 0xFF}) {
		return "image/jpeg", nil
	}
	if len(imageBytes) >= 6 && (string(imageBytes[:6]) == "GIF87a" || string(imageBytes[:6]) == "GIF89a") {
		return "image/gif", nil
	}

	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if mediaType == "image/png" || mediaType == "image/jpeg" || mediaType == "image/gif" || mediaType == "image/webp" {
		return mediaType, nil
	}
	return "", fmt.Errorf("relay image is not a supported image")
}
