package httpapi

type healthResponse struct {
	OK         bool `json:"ok"`
	Configured bool `json:"configured"`
}

type imageSettingsRequest struct {
	Endpoint string `json:"endpoint"`
	APIKey   string `json:"api_key"`
}

type imageSettingsResponse struct {
	Endpoint string `json:"endpoint"`
	APIKey   string `json:"api_key"`
}

type imageProfileRequest struct {
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
	APIKey   string `json:"api_key"`
}

type imageProfileResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Endpoint  string `json:"endpoint"`
	APIKey    string `json:"api_key"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type imageModelRequest struct {
	Model string `json:"model"`
}

type modelOptionResponse struct {
	ID      string `json:"id"`
	Value   string `json:"value"`
	Label   string `json:"label"`
	BuiltIn bool   `json:"built_in"`
}

type imageModelsResponse struct {
	Models []modelOptionResponse `json:"models"`
}

type imageProfilesResponse struct {
	Profiles []imageProfileResponse `json:"profiles"`
}

type generateRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Size    string `json:"size"`
	Quality string `json:"quality"`
}

type imageResponse struct {
	Image string `json:"image"`
}

type historyDeleteRequest struct {
	Image string `json:"image"`
}
type historyTaskDeleteRequest struct {
	ID string `json:"id"`
}

// imageHistoryResponse is retained for tests and compatibility with the legacy in-memory helper.
type imageHistoryResponse struct {
	Images []string `json:"images"`
}

type downloadImageRequest struct {
	Source  string `json:"source"`
	Format  string `json:"format"`
	Quality *int   `json:"quality"`
}

type errorResponse struct {
	Error string `json:"error"`
}
