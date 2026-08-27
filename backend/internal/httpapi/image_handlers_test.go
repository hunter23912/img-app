package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"img-app/backend/internal/store"
)

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
	request := httptest.NewRequest(http.MethodPost, "/api/generate", bytes.NewReader(body))
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
