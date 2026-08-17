package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

const (
	defaultDatabasePath = "data/img-app.db"
	maxStoredTasks      = 50
	maxPresetCount      = 50
	maxPresetName       = 40
	maxPresetPrompt     = 5_000
)

var errNotFound = errors.New("not found")

type appDatabase struct{ db *sql.DB }

type promptPreset struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Prompt    string `json:"prompt"`
	Scope     string `json:"scope"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type promptPresetDraft struct {
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
	Scope  string `json:"scope"`
}

type promptPresetImport struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
	Scope  string `json:"scope"`
}

type imageTask struct {
	ID          string  `json:"id"`
	Mode        string  `json:"mode"`
	Prompt      string  `json:"prompt"`
	Model       string  `json:"model"`
	Size        string  `json:"size"`
	Quality     string  `json:"quality"`
	Status      string  `json:"status"`
	Image       string  `json:"image"`
	Error       string  `json:"error"`
	CreatedAt   string  `json:"created_at"`
	CompletedAt *string `json:"completed_at"`
}

type historyPage struct {
	Tasks      []imageTask `json:"tasks"`
	NextCursor string      `json:"next_cursor"`
	HasMore    bool        `json:"has_more"`
}

type historyCursor struct{ CreatedAt, ID string }

func openDatabase(path string) (*appDatabase, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = defaultDatabasePath
	}
	if parent := filepath.Dir(path); parent != "." {
		if err := os.MkdirAll(parent, fs.FileMode(0o755)); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}
	// WAL improves read behavior for the single backend process while busy_timeout
	// prevents short-lived lock contention from surfacing as request failures.
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	store := &appDatabase{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (d *appDatabase) Close() error {
	if d == nil || d.db == nil {
		return nil
	}
	return d.db.Close()
}

func (d *appDatabase) migrate() error {
	if _, err := d.db.Exec(`
		PRAGMA foreign_keys = ON;
		PRAGMA busy_timeout = 5000;
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		);`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}
	var applied int
	if err := d.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&applied); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if applied >= 1 {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("begin database migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS prompt_presets (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL COLLATE NOCASE UNIQUE,
			prompt TEXT NOT NULL,
			scope TEXT NOT NULL CHECK (scope IN ('generate', 'edit', 'all')),
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_prompt_presets_updated_at ON prompt_presets (updated_at DESC, id DESC);
		CREATE TABLE IF NOT EXISTS image_tasks (
			id TEXT PRIMARY KEY,
			mode TEXT NOT NULL CHECK (mode IN ('generate', 'edit')),
			prompt TEXT NOT NULL,
			model TEXT NOT NULL,
			size TEXT NOT NULL,
			quality TEXT NOT NULL,
			status TEXT NOT NULL CHECK (status IN ('pending', 'succeeded', 'failed')),
			image_url TEXT NOT NULL DEFAULT '',
			error_message TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			completed_at TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_image_tasks_created_at ON image_tasks (created_at DESC, id DESC);
	`); err != nil {
		return fmt.Errorf("create application tables: %w", err)
	}
	if err := seedPromptPresets(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO schema_migrations (version, applied_at) VALUES (1, ?)`, nowString()); err != nil {
		return fmt.Errorf("record schema migration: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit database migration: %w", err)
	}
	return nil
}

func (d *appDatabase) listPresets() ([]promptPreset, error) {
	rows, err := d.db.Query(`SELECT id, name, prompt, scope, created_at, updated_at FROM prompt_presets ORDER BY created_at ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list prompt presets: %w", err)
	}
	defer rows.Close()
	result := make([]promptPreset, 0, maxPresetCount)
	for rows.Next() {
		var preset promptPreset
		if err := rows.Scan(&preset.ID, &preset.Name, &preset.Prompt, &preset.Scope, &preset.CreatedAt, &preset.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, preset)
	}
	return result, rows.Err()
}

func (d *appDatabase) createPreset(input promptPresetDraft) (promptPreset, error) {
	if err := validatePromptPreset(input); err != nil {
		return promptPreset{}, err
	}
	if err := d.ensurePresetAvailable("", input.Name); err != nil {
		return promptPreset{}, err
	}
	now := nowString()
	preset := promptPreset{ID: uuid.NewString(), Name: strings.TrimSpace(input.Name), Prompt: strings.TrimSpace(input.Prompt), Scope: input.Scope, CreatedAt: now, UpdatedAt: now}
	if _, err := d.db.Exec(`INSERT INTO prompt_presets (id, name, prompt, scope, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, preset.ID, preset.Name, preset.Prompt, preset.Scope, now, now); err != nil {
		return promptPreset{}, presetDBError(err)
	}
	return preset, nil
}

func (d *appDatabase) updatePreset(id string, input promptPresetDraft) (promptPreset, error) {
	if err := validatePromptPreset(input); err != nil {
		return promptPreset{}, err
	}
	var preset promptPreset
	err := d.db.QueryRow(`SELECT id, name, prompt, scope, created_at, updated_at FROM prompt_presets WHERE id = ?`, id).Scan(&preset.ID, &preset.Name, &preset.Prompt, &preset.Scope, &preset.CreatedAt, &preset.UpdatedAt)
	if err == sql.ErrNoRows {
		return promptPreset{}, errNotFound
	}
	if err != nil {
		return promptPreset{}, err
	}
	if err := d.ensurePresetAvailable(id, input.Name); err != nil {
		return promptPreset{}, err
	}
	preset.Name, preset.Prompt, preset.Scope = strings.TrimSpace(input.Name), strings.TrimSpace(input.Prompt), input.Scope
	preset.UpdatedAt = nowString()
	if _, err := d.db.Exec(`UPDATE prompt_presets SET name = ?, prompt = ?, scope = ?, updated_at = ? WHERE id = ?`, preset.Name, preset.Prompt, preset.Scope, preset.UpdatedAt, id); err != nil {
		return promptPreset{}, presetDBError(err)
	}
	return preset, nil
}

func (d *appDatabase) deletePreset(id string) error {
	result, err := d.db.Exec(`DELETE FROM prompt_presets WHERE id = ?`, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errNotFound
	}
	return nil
}

func (d *appDatabase) importPresets(inputs []promptPresetImport) (int, error) {
	if len(inputs) == 0 {
		return 0, fmt.Errorf("presets must not be empty")
	}
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var count int
	var total int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM prompt_presets`).Scan(&total); err != nil {
		return 0, err
	}
	names := make(map[string]struct{})
	rows, err := tx.Query(`SELECT name FROM prompt_presets`)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			_ = rows.Close()
			return 0, err
		}
		names[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	for _, input := range inputs {
		if err := validatePromptPreset(promptPresetDraft{Name: input.Name, Prompt: input.Prompt, Scope: input.Scope}); err != nil {
			return 0, err
		}
		id := strings.TrimSpace(input.ID)
		if id == "" {
			id = uuid.NewString()
		}
		var exists int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM prompt_presets WHERE id = ?`, id).Scan(&exists); err != nil {
			return 0, err
		}
		if exists > 0 {
			continue
		}
		nameKey := strings.ToLower(strings.TrimSpace(input.Name))
		if _, exists := names[nameKey]; exists {
			continue
		}
		if total+count >= maxPresetCount {
			return 0, fmt.Errorf("最多保存 %d 套预设", maxPresetCount)
		}
		now := nowString()
		if _, err := tx.Exec(`INSERT INTO prompt_presets (id, name, prompt, scope, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, id, strings.TrimSpace(input.Name), strings.TrimSpace(input.Prompt), input.Scope, now, now); err != nil {
			return 0, presetDBError(err)
		}
		count++
		names[nameKey] = struct{}{}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}

func (d *appDatabase) ensurePresetAvailable(editingID, rawName string) error {
	name := strings.TrimSpace(rawName)
	var count int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM prompt_presets`).Scan(&count); err != nil {
		return err
	}
	if editingID == "" && count >= maxPresetCount {
		return fmt.Errorf("最多保存 %d 套预设", maxPresetCount)
	}
	presets, err := d.listPresets()
	if err != nil {
		return err
	}
	for _, preset := range presets {
		if preset.ID != editingID && strings.EqualFold(strings.TrimSpace(preset.Name), name) {
			return fmt.Errorf("已有同名预设，请换一个名称")
		}
	}
	return nil
}

func (d *appDatabase) createTask(mode string, input generateRequest) (string, error) {
	id := uuid.NewString()
	if _, err := d.db.Exec(`INSERT INTO image_tasks (id, mode, prompt, model, size, quality, status, created_at) VALUES (?, ?, ?, ?, ?, ?, 'pending', ?)`, id, mode, input.Prompt, input.Model, input.Size, input.Quality, nowString()); err != nil {
		return "", err
	}
	if _, err := d.db.Exec(`DELETE FROM image_tasks WHERE id IN (SELECT id FROM image_tasks ORDER BY created_at DESC, id DESC LIMIT -1 OFFSET ?)`, maxStoredTasks); err != nil {
		return "", err
	}
	return id, nil
}

func (d *appDatabase) completeTask(id, image string) error {
	imageURL, ok := normalizeHistoryImageURL(image)
	if !ok {
		imageURL = ""
	}
	_, err := d.db.Exec(`UPDATE image_tasks SET status = 'succeeded', image_url = ?, error_message = '', completed_at = ? WHERE id = ?`, imageURL, nowString(), id)
	return err
}

func (d *appDatabase) historyImageData(id string) (string, error) {
	var image string
	err := d.db.QueryRow(`SELECT image_url FROM image_tasks WHERE id = ? AND status = 'succeeded'`, id).Scan(&image)
	if err == sql.ErrNoRows {
		return "", errNotFound
	}
	if err != nil {
		return "", err
	}
	if !isBase64ImageDataURL(image) {
		return "", errNotFound
	}
	return image, nil
}

func (d *appDatabase) listTasks(limit int, cursor string) (historyPage, error) {
	if limit < 1 {
		limit = 5
	}
	if limit > 5 {
		limit = 5
	}
	args := []any{}
	where := "WHERE status = 'succeeded' AND image_url <> ''"
	if cursor != "" {
		position, err := decodeHistoryCursor(cursor)
		if err != nil {
			return historyPage{}, err
		}
		where += " AND (created_at < ? OR (created_at = ? AND id < ?))"
		args = append(args, position.CreatedAt, position.CreatedAt, position.ID)
	}
	args = append(args, limit+1)
	rows, err := d.db.Query(`SELECT id, mode, prompt, model, size, quality, status, image_url, error_message, created_at, completed_at FROM image_tasks `+where+` ORDER BY created_at DESC, id DESC LIMIT ?`, args...)
	if err != nil {
		return historyPage{}, err
	}
	defer rows.Close()
	page := historyPage{Tasks: make([]imageTask, 0, limit)}
	for rows.Next() {
		var task imageTask
		if err := rows.Scan(&task.ID, &task.Mode, &task.Prompt, &task.Model, &task.Size, &task.Quality, &task.Status, &task.Image, &task.Error, &task.CreatedAt, &task.CompletedAt); err != nil {
			return historyPage{}, err
		}
		task.Image = historyImageReference(task.ID, task.Image)
		page.Tasks = append(page.Tasks, task)
	}
	if err := rows.Err(); err != nil {
		return historyPage{}, err
	}
	page.HasMore = len(page.Tasks) > limit
	if page.HasMore {
		page.Tasks = page.Tasks[:limit]
		last := page.Tasks[len(page.Tasks)-1]
		page.NextCursor = encodeHistoryCursor(historyCursor{CreatedAt: last.CreatedAt, ID: last.ID})
	}
	return page, nil
}

func (d *appDatabase) deleteTask(id string) error {
	result, err := d.db.Exec(`DELETE FROM image_tasks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errNotFound
	}
	return nil
}

func (d *appDatabase) listImageSources() ([]string, error) {
	rows, err := d.db.Query(`SELECT image_url FROM image_tasks WHERE status = 'succeeded' AND image_url <> '' ORDER BY created_at DESC, id DESC LIMIT ?`, maxStoredTasks)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sources []string
	for rows.Next() {
		var source string
		if err := rows.Scan(&source); err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, rows.Err()
}

func encodeHistoryCursor(cursor historyCursor) string {
	value, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(value)
}

func decodeHistoryCursor(value string) (historyCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return historyCursor{}, fmt.Errorf("invalid history cursor")
	}
	var cursor historyCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil || cursor.ID == "" || cursor.CreatedAt == "" {
		return historyCursor{}, fmt.Errorf("invalid history cursor")
	}
	return cursor, nil
}

func validatePromptPreset(input promptPresetDraft) error {
	name, prompt, scope := strings.TrimSpace(input.Name), strings.TrimSpace(input.Prompt), input.Scope
	if name == "" {
		return fmt.Errorf("请填写名称")
	}
	if len([]rune(name)) > maxPresetName {
		return fmt.Errorf("名称不能超过 %d 个字符", maxPresetName)
	}
	if prompt == "" {
		return fmt.Errorf("请填写提示词内容")
	}
	if len([]rune(prompt)) > maxPresetPrompt {
		return fmt.Errorf("提示词不能超过 %d 个字符", maxPresetPrompt)
	}
	if scope != "generate" && scope != "edit" && scope != "all" {
		return fmt.Errorf("请选择有效的适用模式")
	}
	return nil
}

func presetDBError(err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "unique") {
		return fmt.Errorf("已有同名预设，请换一个名称")
	}
	return err
}

func nowString() string { return time.Now().UTC().Format(time.RFC3339Nano) }
