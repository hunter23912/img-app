package provider

import (
	"bufio"
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

const relayRequestTimeout = 10 * time.Minute

func buildImagesURL(endpoint string, action string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("endpoint must be a valid http or https url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("endpoint must use http or https")
	}

	path := strings.TrimRight(parsed.Path, "/")
	async := strings.HasSuffix(path, "/async") || strings.EqualFold(parsed.Hostname(), "api.mikoto.vip")
	path = strings.TrimSuffix(path, "/async")
	for _, suffix := range []string{"/images/generations", "/images/edits"} {
		path = strings.TrimSuffix(path, suffix)
	}
	if path == "" {
		path = "/v1"
	}
	parsed.Path = path + "/images/" + action
	parsed.RawPath = ""
	if async {
		parsed.Path += "/async"
	}
	return parsed.String(), nil
}

func callRelayGenerate(generateURL, apiKey string, input ImageRequest) (string, error) {
	return callRelayGenerateWithContext(context.Background(), generateURL, apiKey, input, nil)
}

func callRelayGenerateWithContext(ctx context.Context, generateURL, apiKey string, input ImageRequest, onEvent ImageEventHandler) (string, error) {
	input = normalizeImageRequest(input)
	payload := relayGenerateRequest{
		Model:        input.Model,
		Prompt:       input.Prompt,
		Size:         input.Size,
		Quality:      input.Quality,
		Moderation:   input.Moderation,
		Background:   input.Background,
		OutputFormat: input.OutputFormat,
		N:            relayN(input.N),
	}
	if isAsyncImagesURL(generateURL) {
		if input.N != 1 {
			return "", fmt.Errorf("异步生图只支持 n=1")
		}
		payload.N = 1
		payload.ResponseFormat = "b64_json"
	}
	return callRelayJSON(ctx, generateURL, apiKey, payload, input.OutputFormat, "generate", onEvent)
}

func callRelayEditWithContext(
	ctx context.Context,
	editURL string,
	apiKey string,
	input ImageRequest,
	imageFile multipart.File,
	imageHeader *multipart.FileHeader,
	maskFile multipart.File,
	maskHeader *multipart.FileHeader,
	onEvent ImageEventHandler,
) (string, error) {
	return callRelayEditImagesWithContext(ctx, editURL, apiKey, input, []ImageFile{{File: imageFile, Header: imageHeader}}, maskFile, maskHeader, onEvent)
}

func callRelayEditImagesWithContext(
	ctx context.Context,
	editURL string,
	apiKey string,
	input ImageRequest,
	images []ImageFile,
	maskFile multipart.File,
	maskHeader *multipart.FileHeader,
	onEvent ImageEventHandler,
) (string, error) {
	input = normalizeImageRequest(input)
	if len(images) == 0 {
		return "", fmt.Errorf("input image is required")
	}
	for _, image := range images {
		if image.File == nil || image.Header == nil {
			return "", fmt.Errorf("input image is required")
		}
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if isAsyncImagesURL(editURL) {
		if input.N != 1 {
			return "", fmt.Errorf("异步图编辑只支持 n=1")
		}
		if err := writer.WriteField("response_format", "b64_json"); err != nil {
			return "", err
		}
	}
	for name, value := range map[string]string{
		"model":         input.Model,
		"prompt":        editPrompt(input.Prompt, len(images)),
		"size":          input.Size,
		"quality":       input.Quality,
		"moderation":    input.Moderation,
		"output_format": input.OutputFormat,
	} {
		if err := writer.WriteField(name, value); err != nil {
			return "", fmt.Errorf("write %s field: %w", name, err)
		}
	}
	if input.N > 1 {
		if err := writer.WriteField("n", fmt.Sprint(input.N)); err != nil {
			return "", fmt.Errorf("write n field: %w", err)
		}
	}
	for _, image := range images {
		if err := copyMultipartFile(writer, "image[]", image.Header.Filename, image.Header.Header.Get("Content-Type"), image.File); err != nil {
			return "", err
		}
	}
	if maskFile != nil && maskHeader != nil {
		if err := copyMultipartFile(writer, "mask", maskHeader.Filename, maskHeader.Header.Get("Content-Type"), maskFile); err != nil {
			return "", err
		}
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	return callRelayBodyWithContext(ctx, editURL, apiKey, &body, writer.FormDataContentType(), input.OutputFormat, "edit", onEvent)
}

func editPrompt(prompt string, imageCount int) string {
	if imageCount < 2 {
		return prompt
	}
	var mapping strings.Builder
	mapping.WriteString(prompt)
	mapping.WriteString("\n\n图片引用说明：@1 为主图")
	for index := 2; index <= imageCount; index++ {
		mapping.WriteString(fmt.Sprintf("；@%d 为第 %d 张参考图", index, index))
	}
	mapping.WriteString("。请优先按照提示词中明确指定的 @编号使用对应图片。")
	return mapping.String()
}

func callRelayEdit(
	editURL string,
	apiKey string,
	input ImageRequest,
	imageFile multipart.File,
	imageHeader *multipart.FileHeader,
	maskFile multipart.File,
	maskHeader *multipart.FileHeader,
) (string, error) {
	return callRelayEditWithContext(context.Background(), editURL, apiKey, input, imageFile, imageHeader, maskFile, maskHeader, nil)
}

func callRelayJSON(ctx context.Context, endpoint, apiKey string, payload any, outputFormat, operation string, onEvent ImageEventHandler) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode relay request: %w", err)
	}
	return callRelayBodyWithContext(ctx, endpoint, apiKey, bytes.NewReader(body), "application/json", outputFormat, operation, onEvent)
}

func callRelayBodyWithContext(ctx context.Context, endpoint, apiKey string, body io.Reader, contentType, outputFormat, operation string, onEvent ImageEventHandler) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	requestContext, cancel := context.WithTimeout(ctx, relayRequestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodPost, endpoint, body)
	if err != nil {
		return "", fmt.Errorf("create relay request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Cache-Control", "no-store")

	start := time.Now()
	slog.Info("relay "+operation+" call", "url", endpoint)
	response, err := (&http.Client{}).Do(request)
	if err != nil {
		slog.Error("relay "+operation+" network error", "duration_ms", time.Since(start).Milliseconds(), "error", err)
		return "", fmt.Errorf("call relay: %w", err)
	}
	defer response.Body.Close()

	var image string
	var responseBytes int
	var decodeErr error
	if isAsyncImagesURL(endpoint) {
		image, responseBytes, decodeErr = resolveAsyncImage(requestContext, response, endpoint, apiKey, outputFormat)
	} else {
		image, responseBytes, decodeErr = decodeRelayResponse(response, outputFormat, onEvent)
	}
	slog.Log(context.Background(), relayResponseLevel(response.StatusCode), "relay "+operation+" response",
		"status", response.StatusCode,
		"duration_ms", time.Since(start).Milliseconds(),
		"content_type", response.Header.Get("Content-Type"),
		"content_length", response.ContentLength,
		"transfer_encoding", response.TransferEncoding,
		"bytes", responseBytes,
	)
	if decodeErr != nil {
		return "", decodeErr
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("relay returned status %d", response.StatusCode)
	}
	return image, nil
}

func decodeRelayResponse(response *http.Response, outputFormat string, onEvent ImageEventHandler) (string, int, error) {
	contentType := strings.ToLower(response.Header.Get("Content-Type"))
	if strings.Contains(contentType, "text/event-stream") {
		return decodeRelaySSE(response, outputFormat, onEvent)
	}
	if isDirectImageContentType(contentType) || isPotentialBinaryImageContentType(contentType) {
		return decodeRelayBinaryImageResponse(response, contentType)
	}

	relayResponse, responseBytes, err := decodeRelayImageResponse(response)
	if err != nil {
		return "", responseBytes, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", responseBytes, relayError(response.StatusCode, relayResponse)
	}
	if response.StatusCode == http.StatusAccepted {
		if relayResponse.JobID != "" {
			return "", responseBytes, fmt.Errorf("image job %s is %s; sync mode was expected", relayResponse.JobID, relayResponse.Status)
		}
		return "", responseBytes, fmt.Errorf("relay returned async job; sync image data was expected")
	}

	image, err := firstImage(relayResponse, outputFormat)
	return image, responseBytes, err
}

func isDirectImageContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])
	}
	return strings.HasPrefix(strings.ToLower(mediaType), "image/")
}

func isPotentialBinaryImageContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])
	}
	return strings.EqualFold(mediaType, "application/octet-stream")
}

func decodeRelayBinaryImageResponse(response *http.Response, contentType string) (string, int, error) {
	body, err := io.ReadAll(io.LimitReader(response.Body, relayResponseBodyLimit+1))
	if err != nil {
		return "", len(body), fmt.Errorf("read relay image response: %w", err)
	}
	if len(body) > relayResponseBodyLimit {
		return "", len(body), fmt.Errorf("relay image response is too large")
	}
	if len(body) == 0 {
		return "", 0, fmt.Errorf("relay image response is empty")
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(strings.ToLower(mediaType), "image/") {
		mediaType = http.DetectContentType(body)
	}
	if !strings.HasPrefix(strings.ToLower(mediaType), "image/") {
		return "", len(body), nonJSONResponseError(response, body)
	}

	return "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(body), len(body), nil
}

func decodeRelayImageResponse(response *http.Response) (relayImageResponse, int, error) {
	var relayResponse relayImageResponse
	var captured bytes.Buffer
	limitedBody := io.LimitReader(io.TeeReader(response.Body, &captured), relayResponseBodyLimit)
	var raw json.RawMessage
	if err := json.NewDecoder(limitedBody).Decode(&raw); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || errors.Is(err, io.ErrUnexpectedEOF) {
			return relayResponse, captured.Len(), fmt.Errorf("read relay response: %w", err)
		}
		return relayResponse, captured.Len(), nonJSONResponseError(response, captured.Bytes())
	}

	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &relayResponse.Data); err != nil {
			return relayResponse, captured.Len(), nonJSONResponseError(response, captured.Bytes())
		}
	} else if err := json.Unmarshal(trimmed, &relayResponse); err != nil {
		return relayResponse, captured.Len(), nonJSONResponseError(response, captured.Bytes())
	}
	return relayResponse, captured.Len(), nil
}

func copyMultipartFile(writer *multipart.Writer, field, filename, declaredContentType string, file multipart.File) error {
	contentType := imageContentType(filename, declaredContentType)
	if contentType == "" {
		return fmt.Errorf("%s must be a png, jpeg, webp, or gif image", field)
	}

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{
		"name":     field,
		"filename": filepath.Base(filename),
	}))
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

func imageContentType(filename, declaredContentType string) string {
	declaredContentType = strings.ToLower(strings.TrimSpace(strings.SplitN(declaredContentType, ";", 2)[0]))
	switch declaredContentType {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
		return declaredContentType
	}

	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
	switch contentType {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
		return contentType
	default:
		return ""
	}
}

func decodeRelaySSE(response *http.Response, outputFormat string, onEvent ImageEventHandler) (string, int, error) {
	reader := bufio.NewReaderSize(io.LimitReader(response.Body, relayResponseBodyLimit), 64*1024)
	var dataLines []string
	bytesRead := 0
	for {
		line, err := reader.ReadString('\n')
		bytesRead += len(line)
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		if line == "" {
			image, done, eventErr := processRelaySSEEvent(dataLines, outputFormat, onEvent)
			dataLines = nil
			if eventErr != nil {
				return "", bytesRead, eventErr
			}
			if done {
				return image, bytesRead, nil
			}
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				image, done, eventErr := processRelaySSEEvent(dataLines, outputFormat, onEvent)
				if eventErr != nil {
					return "", bytesRead, eventErr
				}
				if done {
					return image, bytesRead, nil
				}
				return "", bytesRead, fmt.Errorf("relay SSE ended without a completed image")
			}
			return "", bytesRead, fmt.Errorf("read relay SSE: %w", err)
		}
	}
}

func processRelaySSEEvent(dataLines []string, outputFormat string, onEvent ImageEventHandler) (string, bool, error) {
	data := strings.TrimSpace(strings.Join(dataLines, "\n"))
	if data == "" || data == "[DONE]" {
		return "", false, nil
	}

	var event map[string]json.RawMessage
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return "", false, fmt.Errorf("decode relay SSE event: %w", err)
	}
	if message := relayErrorMessage(event["error"]); message != "" {
		return "", false, errors.New(message)
	}

	typeName := rawString(event["type"])
	if strings.Contains(typeName, "partial_image") {
		partial := rawString(event["b64_json"])
		if partial == "" {
			partial = rawString(event["partial_image_b64"])
		}
		if partial != "" && onEvent != nil {
			onEvent(ImageEvent{
				Type:              "partial_image",
				Image:             normalizeBase64Image(partial, imageMimeType(outputFormat)),
				PartialImageIndex: rawInt(event["partial_image_index"]),
				HasPartialIndex:   event["partial_image_index"] != nil,
			})
		}
		return "", false, nil
	}
	if typeName == "" && len(event) == 0 {
		return "", false, nil
	}

	if relayResponse, ok := relayResponseFromEvent(event); ok {
		if image, imageErr := firstImage(relayResponse, outputFormat); imageErr == nil {
			return image, true, nil
		}
	}

	if strings.HasSuffix(typeName, ".failed") || strings.HasSuffix(typeName, ".error") {
		message := rawString(event["message"])
		if message == "" {
			message = "relay SSE image request failed"
		}
		return "", false, errors.New(message)
	}
	return "", false, nil
}

func relayResponseFromEvent(event map[string]json.RawMessage) (relayImageResponse, bool) {
	var response relayImageResponse
	if data, ok := event["data"]; ok {
		if err := json.Unmarshal(data, &response.Data); err == nil && len(response.Data) > 0 {
			return response, true
		}
		if err := json.Unmarshal(data, &response); err == nil && len(response.Data) > 0 {
			return response, true
		}
		var item struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
			Image   string `json:"image"`
		}
		if err := json.Unmarshal(data, &item); err == nil {
			if item.URL != "" {
				response.Data = append(response.Data, struct {
					URL     string `json:"url,omitempty"`
					B64JSON string `json:"b64_json,omitempty"`
				}{URL: item.URL})
				return response, true
			}
			if item.B64JSON != "" {
				response.Data = append(response.Data, struct {
					URL     string `json:"url,omitempty"`
					B64JSON string `json:"b64_json,omitempty"`
				}{B64JSON: item.B64JSON})
				return response, true
			}
			if item.Image != "" {
				response.Image = item.Image
				return response, true
			}
		}
	}
	if images, ok := event["images"]; ok {
		if json.Unmarshal(images, &response.Images) == nil && len(response.Images) > 0 {
			return response, true
		}
	}
	for _, key := range []string{"url", "image", "b64_json", "partial_image_b64"} {
		value := rawString(event[key])
		if value == "" {
			continue
		}
		if key == "b64_json" || key == "partial_image_b64" {
			response.Data = append(response.Data, struct {
				URL     string `json:"url,omitempty"`
				B64JSON string `json:"b64_json,omitempty"`
			}{B64JSON: value})
		} else if key == "url" {
			response.Data = append(response.Data, struct {
				URL     string `json:"url,omitempty"`
				B64JSON string `json:"b64_json,omitempty"`
			}{URL: value})
		} else {
			response.Image = value
		}
		return response, true
	}
	return response, false
}

func rawString(value json.RawMessage) string {
	var result string
	if len(value) == 0 || json.Unmarshal(value, &result) != nil {
		return ""
	}
	return strings.TrimSpace(result)
}

func rawInt(value json.RawMessage) int {
	var result int
	if len(value) == 0 || json.Unmarshal(value, &result) != nil {
		return 0
	}
	return result
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

func relayError(statusCode int, relayResponse relayImageResponse) error {
	if message := relayErrorMessage(relayResponse.Error); message != "" {
		return errors.New(message)
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

func relayErrorMessage(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var message string
	if json.Unmarshal(raw, &message) == nil {
		return strings.TrimSpace(message)
	}
	var value struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &value) == nil {
		return strings.TrimSpace(value.Message)
	}
	return ""
}

func nonJSONResponseError(response *http.Response, body []byte) error {
	snippet := strings.TrimSpace(string(body))
	if len(snippet) > 300 {
		snippet = snippet[:300] + "..."
	}
	if snippet == "" {
		snippet = "<empty body>"
	}
	return fmt.Errorf("relay returned non-json response: status %d, content-type %q, body %q", response.StatusCode, response.Header.Get("Content-Type"), snippet)
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
	input.Model = strings.TrimSpace(input.Model)
	input.Size = strings.TrimSpace(input.Size)
	input.Quality = strings.TrimSpace(input.Quality)
	input.Moderation = strings.TrimSpace(input.Moderation)
	input.Background = strings.TrimSpace(input.Background)
	input.OutputFormat = strings.TrimSpace(input.OutputFormat)
	input.N = normalizedN(input.N)
	return input
}

func normalizedN(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func relayN(n int) int {
	if n > 1 {
		return n
	}
	return 0
}

func firstImage(relayResponse relayImageResponse, outputFormat string) (string, error) {
	fallbackMime := imageMimeType(outputFormat)
	for _, item := range relayResponse.Data {
		if value := strings.TrimSpace(item.B64JSON); value != "" {
			return normalizeBase64Image(value, fallbackMime), nil
		}
		if value := strings.TrimSpace(item.URL); value != "" {
			return normalizeImageValue(value, fallbackMime)
		}
	}
	if value := strings.TrimSpace(relayResponse.Image); value != "" {
		return normalizeImageValue(value, fallbackMime)
	}
	for _, value := range relayResponse.Images {
		if value = strings.TrimSpace(value); value != "" {
			return normalizeImageValue(value, fallbackMime)
		}
	}
	return "", fmt.Errorf("relay response has no image url or b64_json")
}

func normalizeImageValue(value, fallbackMime string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), "data:") {
		return value, nil
	}
	parsed, err := url.Parse(value)
	if err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.User == nil && parsed.Fragment == "" {
		// URL 结果直接交给手机端，避免 Go 再下载并编码一次几 MB 的图片。
		return value, nil
	}
	return normalizeBase64Image(value, fallbackMime), nil
}

func normalizeBase64Image(value, fallbackMime string) string {
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
