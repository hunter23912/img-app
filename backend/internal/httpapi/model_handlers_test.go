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

func TestImageModelsHandlerReturnsBuiltInsAndSupportsCustomCRUD(t *testing.T) {
	database, err := store.OpenDatabase(filepath.Join(t.TempDir(), "img-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	config := appConfig{Database: database}

	list := func() imageModelsResponse {
		recorder := httptest.NewRecorder()
		modelsHandler(config).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/models", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("list status = %d: %s", recorder.Code, recorder.Body.String())
		}
		var response imageModelsResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return response
	}

	initial := list()
	if len(initial.Models) != 3 || initial.Models[0].Value != defaultModel || initial.Models[1].Value != grokImageModel || initial.Models[2].Value != geminiImageModel {
		t.Fatalf("initial models = %#v", initial.Models)
	}
	for _, model := range initial.Models {
		if !model.BuiltIn {
			t.Fatalf("model %q should be built in", model.Value)
		}
	}

	body, err := json.Marshal(imageModelRequest{Model: " vendor/custom-image "})
	if err != nil {
		t.Fatal(err)
	}
	create := httptest.NewRecorder()
	modelsHandler(config).ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewReader(body)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d: %s", create.Code, create.Body.String())
	}
	var created modelOptionResponse
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Value != "vendor/custom-image" || created.BuiltIn || created.ID == "" {
		t.Fatalf("created model = %#v", created)
	}

	loaded := list()
	if len(loaded.Models) != 4 || loaded.Models[3].Value != created.Value {
		t.Fatalf("loaded models = %#v", loaded.Models)
	}

	duplicate := httptest.NewRecorder()
	modelsHandler(config).ServeHTTP(duplicate, httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBufferString(`{"model":"VENDOR/CUSTOM-IMAGE"}`)))
	if duplicate.Code != http.StatusBadRequest {
		t.Fatalf("duplicate status = %d: %s", duplicate.Code, duplicate.Body.String())
	}

	remove := httptest.NewRecorder()
	modelsHandler(config).ServeHTTP(remove, httptest.NewRequest(http.MethodDelete, "/api/models/"+created.ID, nil))
	if remove.Code != http.StatusNoContent {
		t.Fatalf("remove status = %d: %s", remove.Code, remove.Body.String())
	}
	if got := len(list().Models); got != 3 {
		t.Fatalf("models after remove = %d, want 3", got)
	}
}

func TestImageModelsAreIsolatedByActiveProfile(t *testing.T) {
	database, err := store.OpenDatabase(filepath.Join(t.TempDir(), "img-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	config := appConfig{Database: database}

	first, err := database.CreateImageProfile("A", "https://a.example", "key-a")
	if err != nil {
		t.Fatal(err)
	}
	second, err := database.CreateImageProfile("B", "https://b.example", "key-b")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.CreateImageModel(store.ImageModelScopeKey(first.ID), "model-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.CreateImageModel(store.ImageModelScopeKey(second.ID), "model-b"); err != nil {
		t.Fatal(err)
	}

	models, err := database.ListImageModels(store.ImageModelScopeKey(first.ID))
	if err != nil || len(models) != 1 || models[0].Model != "model-a" {
		t.Fatalf("first profile models = %#v, error = %v", models, err)
	}
	if _, err := database.ActivateImageProfile(second.ID); err != nil {
		t.Fatal(err)
	}
	models, err = database.ListImageModels(store.ImageModelScopeKey(second.ID))
	if err != nil || len(models) != 1 || models[0].Model != "model-b" {
		t.Fatalf("second profile models = %#v, error = %v", models, err)
	}

	available, err := config.imageModelAvailable("model-a")
	if err != nil {
		t.Fatal(err)
	}
	if available {
		t.Fatal("model from inactive profile should not be available")
	}
}

func TestDeletingProfileRemovesItsCustomModels(t *testing.T) {
	database, err := store.OpenDatabase(filepath.Join(t.TempDir(), "img-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	first, err := database.CreateImageProfile("A", "https://a.example", "key-a")
	if err != nil {
		t.Fatal(err)
	}
	second, err := database.CreateImageProfile("B", "https://b.example", "key-b")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.CreateImageModel(store.ImageModelScopeKey(first.ID), "model-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.CreateImageModel(store.ImageModelScopeKey(second.ID), "model-b"); err != nil {
		t.Fatal(err)
	}
	if err := database.DeleteImageProfile(second.ID); err != nil {
		t.Fatal(err)
	}

	models, err := database.ListImageModels(store.ImageModelScopeKey(second.ID))
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 0 {
		t.Fatalf("deleted profile models = %#v, want empty", models)
	}
	models, err = database.ListImageModels(store.ImageModelScopeKey(first.ID))
	if err != nil || len(models) != 1 || models[0].Model != "model-a" {
		t.Fatalf("remaining profile models = %#v, error = %v", models, err)
	}
}

func TestImageModelsRejectInvalidNamesAndBuiltInDeletion(t *testing.T) {
	database, err := store.OpenDatabase(filepath.Join(t.TempDir(), "img-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	config := appConfig{Database: database}

	for _, body := range []string{
		`{"model":""}`,
		`{"model":"\nmodel"}`,
		`{"model":"gpt-image-2"}`,
		`{"model":"vendor/custom-image"}`, // A valid custom model name is still allowed.
	} {
		recorder := httptest.NewRecorder()
		modelsHandler(config).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBufferString(body)))
		if body == `{"model":"vendor/custom-image"}` {
			if recorder.Code != http.StatusCreated {
				t.Fatalf("custom model status = %d: %s", recorder.Code, recorder.Body.String())
			}
			continue
		}
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("invalid model %s status = %d: %s", body, recorder.Code, recorder.Body.String())
		}
	}

	recorder := httptest.NewRecorder()
	modelsHandler(config).ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/api/models/builtin:"+defaultModel, nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("built-in delete status = %d: %s", recorder.Code, recorder.Body.String())
	}
}
