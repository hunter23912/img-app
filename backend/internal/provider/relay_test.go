package provider

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

const testPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

func testPNGBytes() []byte {
	imageBytes, err := base64.StdEncoding.DecodeString(testPNGBase64)
	if err != nil {
		panic(err)
	}
	return imageBytes
}

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
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(testPNGBytes())
	}))
	defer imageServer.Close()

	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode relay request: %v", err)
			return
		}
		wantFields := map[string]any{
			"moderation":    "auto",
			"background":    "auto",
			"output_format": "png",
			"n":             float64(1),
		}
		for name, want := range wantFields {
			if got := payload[name]; got != want {
				t.Errorf("%s = %v, want %v", name, got, want)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"url":"` + imageServer.URL + `/generated.png"}]}`))
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
	wantImage := "data:image/png;base64," + testPNGBase64
	if image != wantImage {
		t.Fatalf("image = %q, want %q", image, wantImage)
	}
}

func TestCallRelayEditUsesStandardFields(t *testing.T) {
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
			"model":         defaultModel,
			"prompt":        "Upscale this image",
			"size":          "2048x2048",
			"moderation":    "auto",
			"quality":       "auto",
			"output_format": "png",
			"n":             "1",
		}
		for name, want := range wantFields {
			if got := r.FormValue(name); got != want {
				t.Errorf("field %q = %q, want %q", name, got, want)
			}
		}
		if _, header, err := r.FormFile("image[]"); err != nil {
			t.Errorf("image form file error = %v", err)
		} else if header.Filename != "input.png" {
			t.Errorf("image filename = %q, want input.png", header.Filename)
		}
		if r.FormValue("response_format") != "" {
			t.Errorf("response_format should not be sent")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"aGVsbG8="}]}`))
	}))
	defer relay.Close()

	imageFile, err := os.CreateTemp(t.TempDir(), "image-edit-input-*.png")
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
		generateRequest{Model: defaultModel, Prompt: "Upscale this image", Size: "2048x2048", Quality: "auto"},
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

func TestDecodeRelayImageResponseDoesNotWaitForEOF(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()

	result := make(chan struct {
		response relayImageResponse
		bytes    int
		err      error
	}, 1)
	go func() {
		response, responseBytes, err := decodeRelayImageResponse(&http.Response{
			Body:   reader,
			Header: make(http.Header),
		})
		result <- struct {
			response relayImageResponse
			bytes    int
			err      error
		}{response, responseBytes, err}
	}()

	go func() {
		_, _ = writer.Write([]byte(`{"data":[{"url":"https://images.example.com/edited.png"}]}`))
	}()

	select {
	case got := <-result:
		if got.err != nil {
			t.Fatalf("decodeRelayImageResponse() error = %v", got.err)
		}
		if got.bytes == 0 || len(got.response.Data) != 1 || got.response.Data[0].URL != "https://images.example.com/edited.png" {
			t.Fatalf("decoded response = %#v, bytes = %d", got.response, got.bytes)
		}
	case <-time.After(time.Second):
		t.Fatal("decodeRelayImageResponse() waited for response EOF")
	}

	_ = writer.Close()
}

func TestFirstImageSupportsAlternateResponseShapes(t *testing.T) {
	tests := []struct {
		name         string
		response     relayImageResponse
		outputFormat string
		want         string
	}{
		{
			name: "top-level image base64",
			response: relayImageResponse{
				Image: "aGVsbG8=",
			},
			outputFormat: "jpeg",
			want:         "data:image/jpeg;base64,aGVsbG8=",
		},
		{
			name: "top-level images data url",
			response: relayImageResponse{
				Images: []string{"data:image/webp;base64,aGVsbG8="},
			},
			outputFormat: "png",
			want:         "data:image/webp;base64,aGVsbG8=",
		},
		{
			name: "data b64 already data url",
			response: func() relayImageResponse {
				var response relayImageResponse
				if err := json.Unmarshal([]byte(`{"data":[{"b64_json":"data:image/jpeg;base64,aGVsbG8="}]}`), &response); err != nil {
					t.Fatalf("decode test response: %v", err)
				}
				return response
			}(),
			outputFormat: "png",
			want:         "data:image/jpeg;base64,aGVsbG8=",
		},
		{
			name: "data url in url field",
			response: func() relayImageResponse {
				var response relayImageResponse
				if err := json.Unmarshal([]byte(`{"data":[{"url":"data:image/png;base64,aGVsbG8="}]}`), &response); err != nil {
					t.Fatalf("decode test response: %v", err)
				}
				return response
			}(),
			outputFormat: "png",
			want:         "data:image/png;base64,aGVsbG8=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := firstImage(tt.response, tt.outputFormat)
			if err != nil {
				t.Fatalf("firstImage() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("firstImage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFetchImageURLAsDataURLDetectsImageMIME(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jpeg":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte{0xff, 0xd8, 0xff, 0x00})
		case "/webp":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte("RIFF0000WEBP"))
		case "/not-image":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("not an image"))
		case "/missing":
			w.WriteHeader(http.StatusNotFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	for _, tt := range []struct {
		path string
		want string
	}{
		{path: "/jpeg", want: "data:image/jpeg;base64,/9j/AA=="},
		{path: "/webp", want: "data:image/webp;base64,UklGRjAwMDBXRUJQ"},
	} {
		response := relayImageResponse{}
		responseJSON := []byte(`{"data":[{"url":"` + server.URL + tt.path + `"}]}`)
		if err := json.Unmarshal(responseJSON, &response); err != nil {
			t.Fatalf("decode URL response: %v", err)
		}
		got, err := firstImage(response, "png")
		if err != nil {
			t.Fatalf("firstImage(%s) error = %v", tt.path, err)
		}
		if got != tt.want {
			t.Fatalf("firstImage(%s) = %q, want %q", tt.path, got, tt.want)
		}
	}

	for _, path := range []string{"/not-image", "/missing"} {
		response := relayImageResponse{}
		responseJSON := []byte(`{"data":[{"url":"` + server.URL + path + `"}]}`)
		if err := json.Unmarshal(responseJSON, &response); err != nil {
			t.Fatalf("decode URL response: %v", err)
		}
		if _, err := firstImage(response, "png"); err == nil {
			t.Fatalf("firstImage(%s) succeeded for invalid image response", path)
		}
	}
}
