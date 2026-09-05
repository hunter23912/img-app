package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type asyncImageTask struct {
	TaskID string             `json:"task_id"`
	Status string             `json:"status"`
	Result relayImageResponse `json:"result"`
	Error  json.RawMessage    `json:"error"`
}

func isAsyncImagesURL(endpoint string) bool {
	parsed, err := url.Parse(endpoint)
	return err == nil && strings.HasSuffix(strings.TrimRight(parsed.Path, "/"), "/async")
}

func resolveAsyncImage(ctx context.Context, response *http.Response, endpoint, apiKey, outputFormat string) (string, int, error) {
	body, err := io.ReadAll(io.LimitReader(response.Body, relayResponseBodyLimit+1))
	response.Body.Close()
	if err != nil {
		return "", len(body), fmt.Errorf("read async image submission: %w", err)
	}
	if len(body) > relayResponseBodyLimit {
		return "", len(body), fmt.Errorf("async image response is too large")
	}
	var task asyncImageTask
	if err := json.Unmarshal(body, &task); err != nil {
		return "", len(body), nonJSONResponseError(response, body)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", len(body), relayError(response.StatusCode, relayImageResponse{Error: task.Error})
	}
	if task.TaskID == "" {
		return "", len(body), fmt.Errorf("异步提交未返回 task_id")
	}
	image, err := pollAsyncImage(ctx, endpoint, apiKey, outputFormat, task, 3*time.Second)
	return image, len(body), err
}

func pollAsyncImage(ctx context.Context, endpoint, apiKey, outputFormat string, task asyncImageTask, interval time.Duration) (string, error) {
	taskID := task.TaskID
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	// 从提交地址构造同源查询地址，避免把 API Key 发给响应中的任意 poll_url。
	base, _, ok := strings.Cut(parsed.Path, "/images/")
	if !ok {
		return "", fmt.Errorf("异步接口地址缺少 /images/ 路径")
	}
	parsed.Path = base + "/images/tasks/" + taskID
	parsed.RawPath = base + "/images/tasks/" + url.PathEscape(taskID)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	for {
		switch task.Status {
		case "success":
			return firstImage(task.Result, outputFormat)
		case "error", "failed":
			message := relayErrorMessage(task.Error)
			if message == "" {
				message = "图片生成失败"
			}
			return "", fmt.Errorf("任务 %s：%s", taskID, message)
		case "queued", "running":
		default:
			return "", fmt.Errorf("任务 %s 返回未知状态 %q", taskID, task.Status)
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", fmt.Errorf("等待任务 %s：%w", taskID, ctx.Err())
		case <-timer.C:
		}
		pollCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		request, err := http.NewRequestWithContext(pollCtx, http.MethodGet, parsed.String(), nil)
		if err != nil {
			cancel()
			return "", err
		}
		request.Header.Set("Authorization", "Bearer "+apiKey)
		request.Header.Set("Cache-Control", "no-store")
		response, err := client.Do(request)
		if err != nil {
			cancel()
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, relayResponseBodyLimit+1))
		response.Body.Close()
		cancel()
		// 只重试查询，不重发生成请求，避免重复扣费。
		if response.StatusCode == 408 || response.StatusCode == 429 || response.StatusCode >= 500 || readErr != nil {
			continue
		}
		if len(body) > relayResponseBodyLimit {
			return "", fmt.Errorf("任务 %s 的图片结果超过大小限制", taskID)
		}
		var next asyncImageTask
		if err := json.Unmarshal(body, &next); err != nil {
			return "", nonJSONResponseError(response, body)
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return "", relayError(response.StatusCode, relayImageResponse{Error: next.Error})
		}
		if next.TaskID != "" && next.TaskID != taskID {
			return "", fmt.Errorf("异步查询返回了不匹配的 task_id")
		}
		task = next
	}
}
