package main

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestBuildImagesURL(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		action   string
		want     string
	}{
		{
			name:     "base endpoint",
			endpoint: "https://example.com",
			action:   "generations",
			want:     "https://example.com/v1/images/generations",
		},
		{
			name:     "v1 endpoint",
			endpoint: "https://example.com/v1",
			action:   "edits",
			want:     "https://example.com/v1/images/edits",
		},
		{
			name:     "full endpoint",
			endpoint: "https://example.com/v1/images/edits",
			action:   "edits",
			want:     "https://example.com/v1/images/edits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildImagesURL(tt.endpoint, tt.action)
			if err != nil {
				t.Fatalf("buildImagesURL returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("buildImagesURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildImagesURLRejectsInvalidEndpoint(t *testing.T) {
	if _, err := buildImagesURL("file:///tmp/image", "edits"); err == nil {
		t.Fatal("buildImagesURL accepted non-http endpoint")
	}
}

func TestCallRelayGenerate(t *testing.T) {
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode relay request: %v", err)
			return
		}
		if got := payload["moderation"]; got != "low" {
			t.Errorf("moderation = %v, want low", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"url":"https://images.example.com/generated.png"}]}`))
	}))
	defer relay.Close()

	image, err := callRelayGenerate(
		relay.URL+"/v1/images/generations",
		"test-key",
		generateRequest{Model: defaultModel, Prompt: "test prompt", Size: defaultSize, Quality: "auto"},
	)
	if err != nil {
		t.Fatalf("callRelayGenerate() error = %v", err)
	}
	if image != "https://images.example.com/generated.png" {
		t.Fatalf("image = %q, want generated URL", image)
	}
}

func TestCallRelayEditUsesSeedVRFields(t *testing.T) {
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/images/edits" {
			t.Errorf("relay request = %s %s, want POST /v1/images/edits", r.Method, r.URL.Path)
			return
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("ParseMultipartForm() error = %v", err)
			return
		}
		wantFields := map[string]string{
			"model":            seedVRModel,
			"prompt":           "Upscale this image",
			"size":             "2048x2048",
			"moderation":       "low",
			"seed":             "42",
			"color_correction": "wavelet",
			"resize_method":    "lanczos",
			"response_format":  "b64_json",
		}
		for name, want := range wantFields {
			if got := r.FormValue(name); got != want {
				t.Errorf("field %q = %q, want %q", name, got, want)
			}
		}
		if got := r.FormValue("quality"); got != "" {
			t.Errorf("quality = %q, want omitted", got)
		}
		if _, header, err := r.FormFile("image"); err != nil {
			t.Errorf("image form file error = %v", err)
		} else if header.Filename != "input.png" {
			t.Errorf("image filename = %q, want input.png", header.Filename)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"aGVsbG8="}]}`))
	}))
	defer relay.Close()

	imageFile, err := os.CreateTemp(t.TempDir(), "seedvr-input-*.png")
	if err != nil {
		t.Fatal(err)
	}
	defer imageFile.Close()
	if _, err := imageFile.WriteString("not-analyzed-by-relay-test"); err != nil {
		t.Fatal(err)
	}
	if _, err := imageFile.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	image, err := callRelayEdit(
		relay.URL+"/v1/images/edits",
		"test-key",
		generateRequest{Model: seedVRModel, Prompt: "Upscale this image", Size: "2048x2048", Quality: "auto"},
		imageFile,
		&multipart.FileHeader{Filename: "input.png"},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("callRelayEdit() error = %v", err)
	}
	if image != "data:image/png;base64,aGVsbG8=" {
		t.Fatalf("image = %q, want decoded data URL", image)
	}
}
