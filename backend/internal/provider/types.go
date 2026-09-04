package provider

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
	Model        string `json:"model"`
	Prompt       string `json:"prompt"`
	Size         string `json:"size,omitempty"`
	Quality      string `json:"quality,omitempty"`
	Moderation   string `json:"moderation,omitempty"`
	Background   string `json:"background,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
	N            int    `json:"n,omitempty"`
}

type relayImageResponse struct {
	Created int64  `json:"created,omitempty"`
	JobID   string `json:"job_id,omitempty"`
	Status  string `json:"status,omitempty"`
	Data    []struct {
		URL     string `json:"url,omitempty"`
		B64JSON string `json:"b64_json,omitempty"`
	} `json:"data"`
	Image  string   `json:"image,omitempty"`
	Images []string `json:"images,omitempty"`
	Error  *struct {
		Message string `json:"message,omitempty"`
		Type    string `json:"type,omitempty"`
	} `json:"error,omitempty"`
}
