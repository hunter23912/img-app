package main

import (
	"fmt"
	"path/filepath"
	"testing"
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

func TestDatabasePersistsBase64ImageResult(t *testing.T) {
	path := filepath.Join(t.TempDir(), "img-app.db")
	database, err := openDatabase(path)
	if err != nil {
		t.Fatal(err)
	}

	image := "data:image/png;base64,aGVsbG8="
	taskID, err := database.createTask("edit", generateRequest{Model: seedVRModel, Prompt: "Upscale", Size: "2048x2048"})
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
	wantReference := historyImagePath(taskID)
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
