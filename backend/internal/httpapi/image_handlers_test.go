package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"img-app/backend/internal/store"
)

func TestValidateImageReferences(t *testing.T) {
	tests := []struct {
		prompt, want string
	}{
		{prompt: "edit @1 and @2", want: ""},
		{prompt: "edit @2,@3", want: "图片引用 @3 无效，请使用 @1 到 @2。"},
		{prompt: "edit email@example.com", want: ""},
		{prompt: "edit @3", want: "图片引用 @3 无效，请使用 @1 到 @2。"},
	}
	for _, tt := range tests {
		t.Run(tt.prompt, func(t *testing.T) {
			if got := validateImageReferences(tt.prompt, 2); got != tt.want {
				t.Fatalf("validateImageReferences() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEditHandlerForwardsMultipleImages(t *testing.T) {
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(2 << 20); err != nil {
			t.Fatalf("parse relay multipart form: %v", err)
		}
		files := r.MultipartForm.File["image[]"]
		if len(files) != 2 || files[0].Filename != "main.png" || files[1].Filename != "reference.png" {
			t.Errorf("relay image files = %#v", files)
		}
		if got := r.FormValue("prompt"); !strings.Contains(got, "@2 为第 2 张参考图") {
			t.Errorf("relay prompt = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"url":"https://images.example.com/edited.png"}]}`))
	}))
	defer relay.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("model", defaultModel)
	_ = writer.WriteField("prompt", "combine @1 with details from @2")
	for _, item := range []struct{ name, content string }{{"main.png", "main"}, {"reference.png", "reference"}} {
		part, err := writer.CreateFormFile("image", item.name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = fmt.Fprint(part, item.content)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/edit?stream=false", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	editHandler(appConfig{Endpoint: relay.URL, APIKey: "test-key"}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "edited.png") {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestGenerateHandlerDoesNotPersistFailedTask(t *testing.T) {
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Fatalf("relay path = %q, want /v1/images/generations", r.URL.Path)
		}
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "generation failed"})
	}))
	defer relay.Close()

	database, err := store.OpenDatabase(filepath.Join(t.TempDir(), "img-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	body, err := json.Marshal(generateRequest{Prompt: "test"})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/generate?stream=false", bytes.NewReader(body))
	generateHandler(appConfig{
		Endpoint: relay.URL,
		APIKey:   "test-key",
		Database: database,
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadGateway)
	}
	page, err := database.ListTasks(5, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Tasks) != 0 {
		t.Fatalf("failed task history = %#v, want empty", page.Tasks)
	}
}

func TestGenerateHandlerDefaultsToSSE(t *testing.T) {
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"` + testPNGBase64 + `"}]}`))
	}))
	defer relay.Close()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/generate", bytes.NewBufferString(`{"prompt":"test"}`))
	generateHandler(appConfig{Endpoint: relay.URL, APIKey: "test-key"}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("response = %d %q %s", recorder.Code, recorder.Header().Get("Content-Type"), recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "image_generate.completed") || !strings.Contains(recorder.Body.String(), "data:image/png;base64,") {
		t.Fatalf("SSE response = %s", recorder.Body.String())
	}
}
