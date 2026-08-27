package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"img-app/backend/internal/history"
	_ "modernc.org/sqlite"
)

func TestDatabaseSeedsAndPersistsData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "img-app.db")
	database, err := openDatabase(path)
	if err != nil {
		t.Fatalf("openDatabase() error = %v", err)
	}
	if presets, err := database.listPresets(); err != nil {
		t.Fatalf("listPresets() error = %v", err)
	} else if len(presets) != 7 {
		t.Fatalf("seeded presets = %d, want 7", len(presets))
	}
	custom, err := database.createPreset(promptPresetDraft{Name: "自定义", Prompt: "保留主体", Scope: "all"})
	if err != nil {
		t.Fatalf("createPreset() error = %v", err)
	}
	taskID, err := database.createTask("generate", generateRequest{Model: defaultModel, Prompt: "test", Size: defaultSize, Quality: "auto"})
	if err != nil {
		t.Fatalf("createTask() error = %v", err)
	}
	if err := database.completeTask(taskID, "https://images.example.com/test.png"); err != nil {
		t.Fatalf("completeTask() error = %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	database, err = openDatabase(path)
	if err != nil {
		t.Fatalf("reopen database error = %v", err)
	}
	defer database.Close()
	presets, err := database.listPresets()
	if err != nil || len(presets) != 8 {
		t.Fatalf("persisted presets = %d, error = %v; want 8", len(presets), err)
	}
	page, err := database.listTasks(5, "")
	if err != nil || len(page.Tasks) != 1 || page.Tasks[0].ID != taskID || page.Tasks[0].Image == "" {
		t.Fatalf("persisted task page = %#v, error = %v", page, err)
	}
	if _, err := database.createPreset(promptPresetDraft{Name: custom.Name, Prompt: "other", Scope: "all"}); err == nil {
		t.Fatal("duplicate preset name was accepted")
	}
}

func TestDatabasePersistsAndClearsImageSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "img-app.db")
	database, err := openDatabase(path)
	if err != nil {
		t.Fatal(err)
	}

	saved, err := database.saveImageSettings("https://saved.example", "saved-key")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Endpoint != "https://saved.example" || saved.APIKey != "saved-key" || saved.UpdatedAt == "" {
		t.Fatalf("saved image settings = %#v", saved)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	database, err = openDatabase(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	loaded, err := database.getImageSettings()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Endpoint != saved.Endpoint || loaded.APIKey != saved.APIKey {
		t.Fatalf("reloaded image settings = %#v, want %#v", loaded, saved)
	}

	if _, err := database.saveImageSettings("", ""); err != nil {
		t.Fatal(err)
	}
	cleared, err := database.getImageSettings()
	if err != nil || cleared.Endpoint != "" || cleared.APIKey != "" {
		t.Fatalf("cleared settings = %#v, error = %v", cleared, err)
	}
}

func TestDatabaseImageProfilesCanBeCreatedActivatedAndDeleted(t *testing.T) {
	database, err := openDatabase(filepath.Join(t.TempDir(), "img-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	first, err := database.createImageProfile("中转站 A", "https://a.example", "key-a")
	if err != nil {
		t.Fatal(err)
	}
	if !first.IsActive {
		t.Fatal("first image profile should be active")
	}
	second, err := database.createImageProfile("中转站 B", "https://b.example", "key-b")
	if err != nil {
		t.Fatal(err)
	}
	if second.IsActive {
		t.Fatal("second image profile should not be active")
	}

	active, found, err := database.getActiveImageProfile()
	if err != nil || !found || active.ID != first.ID {
		t.Fatalf("initial active profile = %#v, found = %v, error = %v", active, found, err)
	}
	if _, err := database.activateImageProfile(second.ID); err != nil {
		t.Fatal(err)
	}
	active, found, err = database.getActiveImageProfile()
	if err != nil || !found || active.ID != second.ID || active.APIKey != "key-b" {
		t.Fatalf("switched active profile = %#v, found = %v, error = %v", active, found, err)
	}
	if err := database.deleteImageProfile(second.ID); err == nil {
		t.Fatal("deleting active image profile should fail")
	}
	if err := database.deleteImageProfile(first.ID); err != nil {
		t.Fatal(err)
	}
}

func TestDatabaseMigratesLegacyImageSettingsToActiveProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = legacy.Exec(`
		CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
		INSERT INTO schema_migrations (version, applied_at) VALUES (1, '2026-01-01T00:00:00Z'), (2, '2026-01-01T00:00:00Z');
		CREATE TABLE image_settings (id INTEGER PRIMARY KEY CHECK (id = 1), endpoint TEXT NOT NULL DEFAULT '', api_key TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL);
		INSERT INTO image_settings (id, endpoint, api_key, updated_at) VALUES (1, 'https://legacy.example', 'legacy-key', '2026-01-02T00:00:00Z');
	`)
	if err != nil {
		_ = legacy.Close()
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	database, err := openDatabase(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	profiles, err := database.listImageProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 || !profiles[0].IsActive || profiles[0].Name != "默认配置" || profiles[0].APIKey != "legacy-key" {
		t.Fatalf("migrated profiles = %#v", profiles)
	}
}

func TestDatabaseRepairsMissingImageModelsTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-models-table.db")
	database, err := openDatabase(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.Exec(`DROP TABLE image_models`); err != nil {
		_ = legacy.Close()
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	database, err = openDatabase(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if models, err := database.listImageModels(defaultModelScope); err != nil || len(models) != 0 {
		t.Fatalf("repaired image models = %#v, error = %v", models, err)
	}
}

func TestDatabasePersistsBase64ImageResult(t *testing.T) {
	path := filepath.Join(t.TempDir(), "img-app.db")
	database, err := openDatabase(path)
	if err != nil {
		t.Fatal(err)
	}

	image := "data:image/png;base64,aGVsbG8="
	taskID, err := database.createTask("edit", generateRequest{Model: defaultModel, Prompt: "Upscale", Size: "2048x2048"})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.completeTask(taskID, image); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	database, err = openDatabase(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	page, err := database.listTasks(5, "")
	if err != nil {
		t.Fatal(err)
	}
	wantReference := history.ImagePath(taskID)
	if len(page.Tasks) != 1 || page.Tasks[0].ID != taskID || page.Tasks[0].Image != wantReference {
		t.Fatalf("persisted base64 task = %#v, want image reference %q", page.Tasks, wantReference)
	}
}

func TestDatabaseTaskHistoryKeepsFiftyNewestTasks(t *testing.T) {
	database, err := openDatabase(filepath.Join(t.TempDir(), "img-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	for i := 0; i < maxStoredTasks+3; i++ {
		id, err := database.createTask("generate", generateRequest{Model: defaultModel, Prompt: "test", Size: defaultSize, Quality: "auto"})
		if err != nil {
			t.Fatal(err)
		}
		if err := database.completeTask(id, fmt.Sprintf("https://images.example.com/%d.png", i)); err != nil {
			t.Fatal(err)
		}
	}
	page, err := database.listTasks(5, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Tasks) != 5 || !page.HasMore {
		t.Fatalf("first history page = %d tasks, has_more = %v", len(page.Tasks), page.HasMore)
	}
	count := len(page.Tasks)
	cursor := page.NextCursor
	for page.HasMore {
		page, err = database.listTasks(5, cursor)
		if err != nil {
			t.Fatal(err)
		}
		count += len(page.Tasks)
		cursor = page.NextCursor
	}
	if count != maxStoredTasks {
		t.Fatalf("paginated task count = %d, want %d", count, maxStoredTasks)
	}
}

func TestDatabaseHistoryExcludesUnsuccessfulTasks(t *testing.T) {
	database, err := openDatabase(filepath.Join(t.TempDir(), "img-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	failedID, err := database.createTask("generate", generateRequest{Model: defaultModel, Prompt: "failed", Size: defaultSize, Quality: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.db.Exec(`UPDATE image_tasks SET status = 'failed', completed_at = ? WHERE id = ?`, nowString(), failedID); err != nil {
		t.Fatal(err)
	}

	pendingID, err := database.createTask("generate", generateRequest{Model: defaultModel, Prompt: "pending", Size: defaultSize, Quality: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	succeededID, err := database.createTask("generate", generateRequest{Model: defaultModel, Prompt: "succeeded", Size: defaultSize, Quality: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.completeTask(succeededID, "https://images.example.com/succeeded.png"); err != nil {
		t.Fatal(err)
	}

	page, err := database.listTasks(5, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Tasks) != 1 || page.Tasks[0].ID != succeededID {
		t.Fatalf("history page = %#v, want only succeeded task %q (pending %q, failed %q)", page, succeededID, pendingID, failedID)
	}
}
