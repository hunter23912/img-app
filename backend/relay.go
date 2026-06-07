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
	"path/filepath"
	"strings"
	"time"
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

	// 中转站错误页可能不是 JSON；限制读取大小，避免异常响应撑爆内存。
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
		// 当前前端按同步结果设计；如果中转站返回异步任务，先明确报错，后续再接入轮询。
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

	// 编辑接口同样限制响应体大小，防止非预期大响应占用过多内存。
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
