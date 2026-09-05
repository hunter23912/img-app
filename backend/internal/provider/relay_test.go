package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

func TestBuildImagesURL(t *testing.T) {
	tests := []struct {
		name, endpoint, action, want string
	}{
		{name: "base endpoint", endpoint: "https://example.com", action: "generations", want: "https://example.com/v1/images/generations"},
		{name: "v1 endpoint", endpoint: "https://example.com/v1", action: "edits", want: "https://example.com/v1/images/edits"},
		{name: "full endpoint", endpoint: "https://example.com/v1/images/edits", action: "edits", want: "https://example.com/v1/images/edits"},
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

func TestCallRelayGenerateReturnsURLWithoutDownloading(t *testing.T) {
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode relay request: %v", err)
			return
		}
		for name, want := range map[string]any{
			"moderation":    "auto",
			"background":    "auto",
			"output_format": "png",
		} {
			if got := payload[name]; got != want {
				t.Errorf("%s = %v, want %v", name, got, want)
			}
		}
		if _, ok := payload["n"]; ok {
			t.Errorf("n should be omitted for one image")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"url":"https://images.example.com/generated.png"}]}`))
	}))
	defer relay.Close()

	image, err := callRelayGenerate(relay.URL+"/v1/images/generations", "test-key", ImageRequest{Model: defaultModel, Prompt: "test prompt", Size: defaultSize, Quality: "auto"})
	if err != nil {
		t.Fatalf("callRelayGenerate() error = %v", err)
	}
	if image != "https://images.example.com/generated.png" {
		t.Fatalf("image = %q, want URL result", image)
	}
}

func TestCallRelayEditUsesMultipartFiles(t *testing.T) {
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/images/edits" {
			t.Errorf("relay request = %s %s, want POST /v1/images/edits", r.Method, r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") {
			t.Errorf("Content-Type = %q, want multipart/form-data", r.Header.Get("Content-Type"))
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart request: %v", err)
			return
		}
		if r.FormValue("model") != defaultModel || r.FormValue("prompt") != "Upscale this image" || r.FormValue("size") != "2048x2048" || r.FormValue("quality") != "auto" || r.FormValue("moderation") != "auto" || r.FormValue("output_format") != "png" {
			t.Errorf("request fields = %#v", r.Form)
		}
		if _, ok := r.Form["n"]; ok {
			t.Errorf("n should be omitted for one image")
		}
		files := r.MultipartForm.File["image[]"]
		if len(files) != 1 || files[0].Filename != "input.png" {
			t.Errorf("image files = %#v", files)
		}
		file, err := files[0].Open()
		if err != nil {
			t.Errorf("open image file: %v", err)
			return
		}
		defer file.Close()
		imageBytes, err := io.ReadAll(file)
		if err != nil || !bytes.Equal(imageBytes, testPNGBytesForProvider()) {
			t.Errorf("image bytes = %v, error = %v", imageBytes, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"url":"https://images.example.com/edited.png"}]}`))
	}))
	defer relay.Close()

	path := filepath.Join(t.TempDir(), "input.png")
	if err := os.WriteFile(path, testPNGBytesForProvider(), 0o600); err != nil {
		t.Fatal(err)
	}
	imageFile, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer imageFile.Close()
	imageHeader := &multipart.FileHeader{Filename: "input.png", Header: textproto.MIMEHeader{"Content-Type": []string{"image/png"}}}
	image, err := callRelayEdit(relay.URL+"/v1/images/edits", "test-key", ImageRequest{Model: defaultModel, Prompt: "Upscale this image", Size: "2048x2048", Quality: "auto"}, imageFile, imageHeader, nil, nil)
	if err != nil {
		t.Fatalf("callRelayEdit() error = %v", err)
	}
	if image != "https://images.example.com/edited.png" {
		t.Fatalf("image = %q, want URL result", image)
	}
}

func TestCallRelayEditUsesMultipleMultipartFilesInOrder(t *testing.T) {
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart request: %v", err)
			return
		}
		files := r.MultipartForm.File["image[]"]
		if len(files) != 2 {
			t.Errorf("image files = %d, want 2", len(files))
			return
		}
		for index, fileHeader := range files {
			file, err := fileHeader.Open()
			if err != nil {
				t.Errorf("open image file %d: %v", index, err)
				continue
			}
			imageBytes, readErr := io.ReadAll(file)
			_ = file.Close()
			want := []byte{byte(index + 1), byte(index + 2), byte(index + 3)}
			if readErr != nil || !bytes.Equal(imageBytes, want) {
				t.Errorf("image %d bytes = %v, error = %v, want %v", index, imageBytes, readErr, want)
			}
		}
		if r.FormValue("prompt") != "Use @2 for the color and @3 for the shape.\n\n图片引用说明：@1 为主图；@2 为第 2 张参考图。请优先按照提示词中明确指定的 @编号使用对应图片。" {
			t.Errorf("prompt = %q", r.FormValue("prompt"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"url":"https://images.example.com/multi-edited.png"}]}`))
	}))
	defer relay.Close()

	paths := []string{
		filepath.Join(t.TempDir(), "main.png"),
		filepath.Join(t.TempDir(), "reference.png"),
	}
	for index, path := range paths {
		if err := os.WriteFile(path, []byte{byte(index + 1), byte(index + 2), byte(index + 3)}, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	files := make([]ImageFile, 0, len(paths))
	opened := make([]*os.File, 0, len(paths))
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		opened = append(opened, file)
		files = append(files, ImageFile{
			File:   file,
			Header: &multipart.FileHeader{Filename: filepath.Base(path), Header: textproto.MIMEHeader{"Content-Type": []string{"image/png"}}},
		})
	}
	defer func() {
		for _, file := range opened {
			_ = file.Close()
		}
	}()

	image, err := callRelayEditImagesWithContext(context.Background(), relay.URL+"/v1/images/edits", "test-key", ImageRequest{Model: defaultModel, Prompt: "Use @2 for the color and @3 for the shape."}, files, nil, nil, nil)
	if err != nil {
		t.Fatalf("callRelayEditImagesWithContext() error = %v", err)
	}
	if image != "https://images.example.com/multi-edited.png" {
		t.Fatalf("image = %q, want URL result", image)
	}
}

func testPNGBytesForProvider() []byte {
	imageBytes, err := base64.StdEncoding.DecodeString(testPNGBase64)
	if err != nil {
		panic(err)
	}
	return imageBytes
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
		response, responseBytes, err := decodeRelayImageResponse(&http.Response{Body: reader, Header: make(http.Header)})
		result <- struct {
			response relayImageResponse
			bytes    int
			err      error
		}{response, responseBytes, err}
	}()
	go func() { _, _ = writer.Write([]byte(`{"data":[{"url":"https://images.example.com/edited.png"}]}`)) }()
	select {
	case got := <-result:
		if got.err != nil || got.bytes == 0 || len(got.response.Data) != 1 || got.response.Data[0].URL != "https://images.example.com/edited.png" {
			t.Fatalf("decoded response = %#v, bytes = %d, error = %v", got.response, got.bytes, got.err)
		}
	case <-time.After(time.Second):
		t.Fatal("decodeRelayImageResponse() waited for response EOF")
	}
	_ = writer.Close()
}

func TestFirstImageSupportsBase64AndDataURLs(t *testing.T) {
	var response relayImageResponse
	if err := json.Unmarshal([]byte(`{"data":[{"b64_json":"`+testPNGBase64+`"}]}`), &response); err != nil {
		t.Fatal(err)
	}
	image, err := firstImage(response, "png")
	if err != nil || image != "data:image/png;base64,"+testPNGBase64 {
		t.Fatalf("firstImage() = %q, error = %v", image, err)
	}
	response = relayImageResponse{}
	if err := json.Unmarshal([]byte(`{"data":[{"url":"data:image/jpeg;base64,aGVsbG8="}]}`), &response); err != nil {
		t.Fatal(err)
	}
	image, err = firstImage(response, "png")
	if err != nil || image != "data:image/jpeg;base64,aGVsbG8=" {
		t.Fatalf("firstImage() = %q, error = %v", image, err)
	}
}

func TestDecodeRelayResponseSupportsDirectBinaryImage(t *testing.T) {
	imageBytes := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0x01, 0x02}
	response := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"image/png"}},
		Body:       io.NopCloser(strings.NewReader(string(imageBytes))),
	}

	image, responseBytes, err := decodeRelayResponse(response, "png", nil)
	if err != nil {
		t.Fatalf("decodeRelayResponse() error = %v", err)
	}
	if responseBytes != len(imageBytes) {
		t.Fatalf("response bytes = %d, want %d", responseBytes, len(imageBytes))
	}
	want := "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageBytes)
	if image != want {
		t.Fatalf("image = %q, want %q", image, want)
	}
}

func TestDecodeRelayResponseSupportsOctetStreamImage(t *testing.T) {
	imageBytes := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F'}
	response := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/octet-stream"}},
		Body:       io.NopCloser(strings.NewReader(string(imageBytes))),
	}

	image, _, err := decodeRelayResponse(response, "png", nil)
	if err != nil {
		t.Fatalf("decodeRelayResponse() error = %v", err)
	}
	if !strings.HasPrefix(image, "data:image/jpeg;base64,") {
		t.Fatalf("image = %q, want detected JPEG data URL", image)
	}
}

func TestRelaySSEPartialImageAndCompleted(t *testing.T) {
	var events []ImageEvent
	response := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(`data: {"type":"image_edit.partial_image","b64_json":"` + testPNGBase64 + `"}

data: {"type":"image_edit.completed","url":"https://images.example.com/final.png"}

`)),
	}
	image, _, err := decodeRelaySSE(response, "png", func(event ImageEvent) { events = append(events, event) })
	if err != nil {
		t.Fatalf("decodeRelaySSE() error = %v", err)
	}
	if len(events) != 1 || events[0].Image == "" || image != "https://images.example.com/final.png" {
		t.Fatalf("events = %#v, image = %q", events, image)
	}
}

func TestCallRelayStopsWhenContextIsCanceled(t *testing.T) {
	started := make(chan struct{})
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
		case <-time.After(time.Second):
		}
	}))
	defer relay.Close()

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := callRelayGenerateWithContext(ctx, relay.URL, "test-key", ImageRequest{Prompt: "test"}, nil)
		result <- err
	}()
	<-started
	cancel()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("callRelayGenerateWithContext() returned nil after cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("callRelayGenerateWithContext() did not stop after cancellation")
	}
}
