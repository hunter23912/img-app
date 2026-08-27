package store

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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
	maxCustomModelCount  = 50
	maxModelName         = 120
	defaultModelScope    = "default"
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

type imageModel struct {
	ID        string
	ScopeKey  string
	Model     string
	CreatedAt string
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

type ImageTaskInput struct {
	Model   string
	Prompt  string
	Size    string
	Quality string
}

type generateRequest = ImageTaskInput

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
		applied = 3
	}
	if applied < 4 {
		return d.applyImageModelsMigration()
	}
	exists, err := d.tableExists("image_models")
	if err != nil {
		return fmt.Errorf("check image models table: %w", err)
	}
	if !exists {
		return d.applyImageModelsMigration()
	}
	return nil
}

func (d *appDatabase) tableExists(name string) (bool, error) {
	var count int
	if err := d.db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND name = ?
	`, name).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
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
		err := tx.QueryRow(`SELECT endpoint, api_key, updated_at FROM image_settings WHERE id = 1`).Scan(&legacy.Endpoint, &legacy.APIKey, &legacy.UpdatedAt)
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

func (d *appDatabase) applyImageModelsMigration() error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("begin image models migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS image_models (
			id TEXT PRIMARY KEY,
			scope_key TEXT NOT NULL,
			model TEXT NOT NULL COLLATE NOCASE,
			created_at TEXT NOT NULL,
			UNIQUE(scope_key, model)
		);
		CREATE INDEX IF NOT EXISTS idx_image_models_scope_created
			ON image_models (scope_key, created_at ASC, id ASC);
	`); err != nil {
		return fmt.Errorf("create image models table: %w", err)
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO schema_migrations (version, applied_at) VALUES (4, ?)`, nowString()); err != nil {
		return fmt.Errorf("record image models migration: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit image models migration: %w", err)
	}
	return nil
}
