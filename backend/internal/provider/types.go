package provider

import "encoding/json"

const (
	defaultModel = "gpt-image-2"
	defaultSize  = "720x1280"
)

type ImageRequest struct {
	Model        string
	Prompt       string
	Size         string
	Quality      string
	Moderation   string
	Background   string
	OutputFormat string
	N            int
}

type generateRequest = ImageRequest

type relayGenerateRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	Moderation     string `json:"moderation,omitempty"`
	Background     string `json:"background,omitempty"`
	OutputFormat   string `json:"output_format,omitempty"`
	N              int    `json:"n,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

type ImageEvent struct {
	Type              string
	Image             string
	PartialImageIndex int
	HasPartialIndex   bool
}

type ImageEventHandler func(ImageEvent)

type relayImageResponse struct {
	Created int64  `json:"created,omitempty"`
	JobID   string `json:"job_id,omitempty"`
	Status  string `json:"status,omitempty"`
	Data    []struct {
		URL     string `json:"url,omitempty"`
		B64JSON string `json:"b64_json,omitempty"`
	} `json:"data"`
	Image  string          `json:"image,omitempty"`
	Images []string        `json:"images,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"`
}
