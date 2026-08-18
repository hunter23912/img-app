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
	defaultDatabasePath  = "data/img-app.db"
	maxStoredTasks       = 50
	maxPresetCount       = 50
	maxPresetName        = 40
	maxPresetPrompt      = 5_000
	maxImageProfileCount = 50
	maxImageProfileName  = 40
)

var errNotFound = errors.New("not found")

type appDatabase struct{ db *sql.DB }

type imageSettings struct {
	Endpoint  string
	APIKey    string
	UpdatedAt string
}

type imageProfile struct {
	ID        string
	Name      string
	Endpoint  string
	APIKey    string
	IsActive  bool
	CreatedAt string
	UpdatedAt string
}

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
	if err := os.Chmod(path, 0o600); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("protect database file: %w", err)
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
	if applied < 1 {
		if err := d.applyInitialMigration(); err != nil {
			return err
		}
		applied = 1
	}
	if applied < 2 {
		if err := d.applyImageSettingsMigration(); err != nil {
			return err
		}
		applied = 2
	}
	if applied < 3 {
		if err := d.applyImageProfilesMigration(); err != nil {
			return err
		}
	}
	return nil
}

func (d *appDatabase) applyInitialMigration() error {
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

func (d *appDatabase) applyImageSettingsMigration() error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("begin image settings migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS image_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			endpoint TEXT NOT NULL DEFAULT '',
			api_key TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL
		);
	`); err != nil {
		return fmt.Errorf("create image settings table: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO schema_migrations (version, applied_at) VALUES (2, ?)`, nowString()); err != nil {
		return fmt.Errorf("record image settings migration: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit image settings migration: %w", err)
	}
	return nil
}

func (d *appDatabase) getImageSettings() (imageSettings, error) {
	var settings imageSettings
	err := d.db.QueryRow(`SELECT endpoint, api_key, updated_at FROM image_settings WHERE id = 1`).Scan(
		&settings.Endpoint,
		&settings.APIKey,
		&settings.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return imageSettings{}, nil
	}
	if err != nil {
		return imageSettings{}, fmt.Errorf("read image settings: %w", err)
	}
	return settings, nil
}

func (d *appDatabase) saveImageSettings(endpoint string, apiKey string) (imageSettings, error) {
	settings := imageSettings{
		Endpoint:  strings.TrimSpace(endpoint),
		APIKey:    strings.TrimSpace(apiKey),
		UpdatedAt: nowString(),
	}
	_, err := d.db.Exec(`
		INSERT INTO image_settings (id, endpoint, api_key, updated_at)
		VALUES (1, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			endpoint = excluded.endpoint,
			api_key = excluded.api_key,
			updated_at = excluded.updated_at
	`, settings.Endpoint, settings.APIKey, settings.UpdatedAt)
	if err != nil {
		return imageSettings{}, fmt.Errorf("save image settings: %w", err)
	}
	return settings, nil
}

func (d *appDatabase) applyImageProfilesMigration() error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("begin image profiles migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS image_profiles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL COLLATE NOCASE UNIQUE,
			endpoint TEXT NOT NULL,
			api_key TEXT NOT NULL DEFAULT '',
			is_active INTEGER NOT NULL DEFAULT 0 CHECK (is_active IN (0, 1)),
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_image_profiles_active
			ON image_profiles (is_active) WHERE is_active = 1;
		CREATE INDEX IF NOT EXISTS idx_image_profiles_updated_at
			ON image_profiles (updated_at DESC, id DESC);
	`); err != nil {
		return fmt.Errorf("create image profiles table: %w", err)
	}

	var profileCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM image_profiles`).Scan(&profileCount); err != nil {
		return fmt.Errorf("count image profiles: %w", err)
	}
	if profileCount == 0 {
		var legacy imageSettings
		err := tx.QueryRow(`SELECT endpoint, api_key, updated_at FROM image_settings WHERE id = 1`).Scan(
			&legacy.Endpoint, &legacy.APIKey, &legacy.UpdatedAt,
		)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("read legacy image settings: %w", err)
		}
		if err == nil && (strings.TrimSpace(legacy.Endpoint) != "" || strings.TrimSpace(legacy.APIKey) != "") {
			now := nowString()
			if strings.TrimSpace(legacy.UpdatedAt) != "" {
				now = legacy.UpdatedAt
			}
			if _, err := tx.Exec(`
				INSERT INTO image_profiles (id, name, endpoint, api_key, is_active, created_at, updated_at)
				VALUES (?, ?, ?, ?, 1, ?, ?)
			`, uuid.NewString(), "默认配置", strings.TrimSpace(legacy.Endpoint), strings.TrimSpace(legacy.APIKey), now, now); err != nil {
				return fmt.Errorf("migrate legacy image settings: %w", err)
			}
		}
	}

	if _, err := tx.Exec(`INSERT INTO schema_migrations (version, applied_at) VALUES (3, ?)`, nowString()); err != nil {
		return fmt.Errorf("record image profiles migration: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit image profiles migration: %w", err)
	}
	return nil
}

func (d *appDatabase) listImageProfiles() ([]imageProfile, error) {
	rows, err := d.db.Query(`
		SELECT id, name, endpoint, api_key, is_active, created_at, updated_at
		FROM image_profiles
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list image profiles: %w", err)
	}
	defer rows.Close()
	profiles := make([]imageProfile, 0, maxImageProfileCount)
	for rows.Next() {
		var profile imageProfile
		var active int
		if err := rows.Scan(&profile.ID, &profile.Name, &profile.Endpoint, &profile.APIKey, &active, &profile.CreatedAt, &profile.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan image profile: %w", err)
		}
		profile.IsActive = active == 1
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read image profiles: %w", err)
	}
	return profiles, nil
}

func (d *appDatabase) getActiveImageProfile() (imageProfile, bool, error) {
	var profile imageProfile
	var active int
	err := d.db.QueryRow(`
		SELECT id, name, endpoint, api_key, is_active, created_at, updated_at
		FROM image_profiles WHERE is_active = 1 LIMIT 1
	`).Scan(&profile.ID, &profile.Name, &profile.Endpoint, &profile.APIKey, &active, &profile.CreatedAt, &profile.UpdatedAt)
	if err == sql.ErrNoRows {
		return imageProfile{}, false, nil
	}
	if err != nil {
		return imageProfile{}, false, fmt.Errorf("read active image profile: %w", err)
	}
	profile.IsActive = active == 1
	return profile, true, nil
}

func (d *appDatabase) createImageProfile(name, endpoint, apiKey string) (imageProfile, error) {
	name = strings.TrimSpace(name)
	endpoint = normalizeImageEndpoint(endpoint)
	apiKey = strings.TrimSpace(apiKey)
	var count int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM image_profiles`).Scan(&count); err != nil {
		return imageProfile{}, fmt.Errorf("count image profiles: %w", err)
	}
	if count >= maxImageProfileCount {
		return imageProfile{}, fmt.Errorf("最多保存 %d 个中转站配置", maxImageProfileCount)
	}
	if name == "" {
		var err error
		name, err = d.nextImageProfileName()
		if err != nil {
			return imageProfile{}, err
		}
	}
	if err := validateImageProfileInput(name, endpoint); err != nil {
		return imageProfile{}, err
	}
	now := nowString()
	profile := imageProfile{ID: uuid.NewString(), Name: name, Endpoint: endpoint, APIKey: apiKey, IsActive: count == 0, CreatedAt: now, UpdatedAt: now}
	active := 0
	if profile.IsActive {
		active = 1
	}
	if _, err := d.db.Exec(`
		INSERT INTO image_profiles (id, name, endpoint, api_key, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, profile.ID, profile.Name, profile.Endpoint, profile.APIKey, active, profile.CreatedAt, profile.UpdatedAt); err != nil {
		return imageProfile{}, imageProfileDBError(err)
	}
	return profile, nil
}

func (d *appDatabase) updateImageProfile(id, name, endpoint, apiKey string) (imageProfile, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	endpoint = normalizeImageEndpoint(endpoint)
	apiKey = strings.TrimSpace(apiKey)
	var profile imageProfile
	var active int
	err := d.db.QueryRow(`
		SELECT id, name, endpoint, api_key, is_active, created_at, updated_at
		FROM image_profiles WHERE id = ?
	`, id).Scan(&profile.ID, &profile.Name, &profile.Endpoint, &profile.APIKey, &active, &profile.CreatedAt, &profile.UpdatedAt)
	if err == sql.ErrNoRows {
		return imageProfile{}, errNotFound
	}
	if err != nil {
		return imageProfile{}, fmt.Errorf("read image profile: %w", err)
	}
	profile.IsActive = active == 1
	if name == "" {
		name = profile.Name
	}
	if err := validateImageProfileInput(name, endpoint); err != nil {
		return imageProfile{}, err
	}
	profile.Name, profile.Endpoint, profile.APIKey, profile.UpdatedAt = name, endpoint, apiKey, nowString()
	if _, err := d.db.Exec(`UPDATE image_profiles SET name = ?, endpoint = ?, api_key = ?, updated_at = ? WHERE id = ?`, profile.Name, profile.Endpoint, profile.APIKey, profile.UpdatedAt, id); err != nil {
		return imageProfile{}, imageProfileDBError(err)
	}
	return profile, nil
}

func (d *appDatabase) nextImageProfileName() (string, error) {
	rows, err := d.db.Query(`SELECT name FROM image_profiles`)
	if err != nil {
		return "", fmt.Errorf("list image profile names: %w", err)
	}
	defer rows.Close()

	names := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return "", fmt.Errorf("scan image profile name: %w", err)
		}
		names[strings.ToLower(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("read image profile names: %w", err)
	}

	for index := 1; ; index++ {
		name := fmt.Sprintf("中转站 %d", index)
		if _, exists := names[strings.ToLower(name)]; !exists {
			return name, nil
		}
	}
}

func (d *appDatabase) activateImageProfile(id string) (imageProfile, error) {
	id = strings.TrimSpace(id)
	tx, err := d.db.Begin()
	if err != nil {
		return imageProfile{}, fmt.Errorf("begin activate image profile: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var exists int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM image_profiles WHERE id = ?`, id).Scan(&exists); err != nil {
		return imageProfile{}, err
	}
	if exists == 0 {
		return imageProfile{}, errNotFound
	}
	if _, err := tx.Exec(`UPDATE image_profiles SET is_active = 0 WHERE is_active = 1`); err != nil {
		return imageProfile{}, err
	}
	updatedAt := nowString()
	if _, err := tx.Exec(`UPDATE image_profiles SET is_active = 1, updated_at = ? WHERE id = ?`, updatedAt, id); err != nil {
		return imageProfile{}, err
	}
	var profile imageProfile
	var active int
	if err := tx.QueryRow(`SELECT id, name, endpoint, api_key, is_active, created_at, updated_at FROM image_profiles WHERE id = ?`, id).Scan(&profile.ID, &profile.Name, &profile.Endpoint, &profile.APIKey, &active, &profile.CreatedAt, &profile.UpdatedAt); err != nil {
		return imageProfile{}, err
	}
	profile.IsActive = active == 1
	if err := tx.Commit(); err != nil {
		return imageProfile{}, fmt.Errorf("commit activate image profile: %w", err)
	}
	return profile, nil
}

func (d *appDatabase) deleteImageProfile(id string) error {
	id = strings.TrimSpace(id)
	var active int
	err := d.db.QueryRow(`SELECT is_active FROM image_profiles WHERE id = ?`, id).Scan(&active)
	if err == sql.ErrNoRows {
		return errNotFound
	}
	if err != nil {
		return err
	}
	if active == 1 {
		return fmt.Errorf("请先切换到其他中转站配置")
	}
	result, err := d.db.Exec(`DELETE FROM image_profiles WHERE id = ?`, id)
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

func validateImageProfileInput(name, endpoint string) error {
	if name == "" {
		return fmt.Errorf("请填写中转站名称")
	}
	if len([]rune(name)) > maxImageProfileName {
		return fmt.Errorf("中转站名称不能超过 %d 个字符", maxImageProfileName)
	}
	if err := validateImageEndpoint(endpoint); err != nil {
		return err
	}
	return nil
}

func imageProfileDBError(err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "unique") {
		return fmt.Errorf("已有同名中转站配置，请换一个名称")
	}
	return fmt.Errorf("保存中转站配置失败: %w", err)
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
