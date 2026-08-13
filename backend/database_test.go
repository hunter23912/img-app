package main

import (
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
		if err := database.completeTask(id, "https://images.example.com/"+string(rune('a'+i))+".png"); err != nil {
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
