package main

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
