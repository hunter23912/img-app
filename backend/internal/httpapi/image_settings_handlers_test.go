package httpapi

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"path/filepath"
	"testing"

	"img-app/backend/internal/history"
	"img-app/backend/internal/imageops"
	"img-app/backend/internal/store"
)

type testRelayImageResponse struct {
	Data []testRelayImage `json:"data"`
}

type testRelayImage struct {
	URL     string `json:"url,omitempty"`
	B64JSON string `json:"b64_json,omitempty"`
}

func TestImageSettingsHandlerReturnsEffectiveValuesAndValidatesEndpoint(t *testing.T) {
	database, err := store.OpenDatabase(filepath.Join(t.TempDir(), "img-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	config := appConfig{Endpoint: "https://env.example", APIKey: "env-key", Database: database}
	get := httptest.NewRecorder()
	imageSettingsHandler(config).ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/settings/image", nil))
	if get.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", get.Code, http.StatusOK)
	}
	var initial imageSettingsResponse
	if err := json.Unmarshal(get.Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}
	if initial.Endpoint != config.Endpoint || initial.APIKey != config.APIKey {
		t.Fatalf("initial settings = %#v, want fallback values", initial)
	}

	body, err := json.Marshal(imageSettingsRequest{Endpoint: "https://saved.example/v1", APIKey: "saved-key"})
	if err != nil {
		t.Fatal(err)
	}
	put := httptest.NewRecorder()
	imageSettingsHandler(config).ServeHTTP(put, httptest.NewRequest(http.MethodPut, "/api/settings/image", bytes.NewReader(body)))
	if put.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d: %s", put.Code, http.StatusOK, put.Body.String())
	}
	var saved imageSettingsResponse
	if err := json.Unmarshal(put.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Endpoint != "https://saved.example/v1" || saved.APIKey != "saved-key" {
		t.Fatalf("saved settings = %#v", saved)
	}

	invalidBody := bytes.NewBufferString(`{"endpoint":"ftp://invalid.example","api_key":"key"}`)
	invalid := httptest.NewRecorder()
	imageSettingsHandler(config).ServeHTTP(invalid, httptest.NewRequest(http.MethodPut, "/api/settings/image", invalidBody))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid endpoint status = %d, want %d", invalid.Code, http.StatusBadRequest)
	}
}

func TestImageSettingsHandlerAllowsEmptyValuesToUseEnvironment(t *testing.T) {
	database, err := store.OpenDatabase(filepath.Join(t.TempDir(), "img-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	config := appConfig{Endpoint: "https://env.example", APIKey: "env-key", Database: database}

	body := bytes.NewBufferString(`{"endpoint":"","api_key":""}`)
	recorder := httptest.NewRecorder()
	imageSettingsHandler(config).ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/api/settings/image", body))
	if recorder.Code != http.StatusOK {
		t.Fatalf("clear settings status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response imageSettingsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Endpoint != config.Endpoint || response.APIKey != config.APIKey {
		t.Fatalf("cleared settings = %#v, want fallback values", response)
	}
}

func TestHealthAndSettingsWorkWithoutAPIKey(t *testing.T) {
	database, err := store.OpenDatabase(filepath.Join(t.TempDir(), "img-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	config := appConfig{Endpoint: defaultEndpoint, Database: database}

	health := httptest.NewRecorder()
	healthHandler(config).ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", health.Code, http.StatusOK)
	}
	var healthResponseValue healthResponse
	if err := json.Unmarshal(health.Body.Bytes(), &healthResponseValue); err != nil {
		t.Fatal(err)
	}
	if !healthResponseValue.OK || healthResponseValue.Configured {
		t.Fatalf("health response = %#v, want online and unconfigured", healthResponseValue)
	}

	settings := httptest.NewRecorder()
	imageSettingsHandler(config).ServeHTTP(settings, httptest.NewRequest(http.MethodGet, "/api/settings/image", nil))
	if settings.Code != http.StatusOK {
		t.Fatalf("settings status = %d, want %d", settings.Code, http.StatusOK)
	}

	body := bytes.NewBufferString(`{"prompt":"test"}`)
	generate := httptest.NewRecorder()
	generateHandler(config).ServeHTTP(generate, httptest.NewRequest(http.MethodPost, "/api/generate", body))
	if generate.Code != http.StatusInternalServerError {
		t.Fatalf("generate without key status = %d, want %d", generate.Code, http.StatusInternalServerError)
	}
}

func TestGenerateUsesSavedImageSettings(t *testing.T) {
	const savedKey = "saved-key"
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Fatalf("relay path = %q, want /v1/images/generations", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer "+savedKey {
			t.Fatalf("relay authorization = %q, want saved key", r.Header.Get("Authorization"))
		}
		writeJSON(w, http.StatusOK, testRelayImageResponse{Data: []testRelayImage{{URL: "https://images.example.com/saved.png"}}})
	}))
	defer relay.Close()

	database, err := store.OpenDatabase(filepath.Join(t.TempDir(), "img-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.SaveImageSettings(relay.URL, savedKey); err != nil {
		t.Fatal(err)
	}

	body := bytes.NewBufferString(`{"prompt":"test"}`)
	recorder := httptest.NewRecorder()
	generateHandler(appConfig{
		Endpoint:            "https://fallback.example",
		APIKey:              "fallback-key",
		Database:            database,
		ImageSourceRegistry: imageops.NewSourceRegistry(),
		ImageHistory:        history.New(),
	}).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/generate", body))
	if recorder.Code != http.StatusOK {
		t.Fatalf("generate status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
}

func TestEditUsesSavedImageSettings(t *testing.T) {
	const savedKey = "saved-edit-key"
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/edits" {
			t.Fatalf("relay path = %q, want /v1/images/edits", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer "+savedKey {
			t.Fatalf("relay authorization = %q, want saved key", r.Header.Get("Authorization"))
		}
		writeJSON(w, http.StatusOK, testRelayImageResponse{Data: []testRelayImage{{URL: "https://images.example.com/edited.png"}}})
	}))
	defer relay.Close()

	database, err := store.OpenDatabase(filepath.Join(t.TempDir(), "img-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.SaveImageSettings(relay.URL, savedKey); err != nil {
		t.Fatal(err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("prompt", "edit"); err != nil {
		t.Fatal(err)
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="image"; filename="input.png"`)
	header.Set("Content-Type", "image/png")
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("not-an-image")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/edit", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	editHandler(appConfig{
		Endpoint:            "https://fallback.example",
		APIKey:              "fallback-key",
		Database:            database,
		ImageSourceRegistry: imageops.NewSourceRegistry(),
		ImageHistory:        history.New(),
	}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("edit status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
}
