package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

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
	endpoint = NormalizeImageEndpoint(endpoint)
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
	endpoint = NormalizeImageEndpoint(endpoint)
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
	if _, err := d.db.Exec(`DELETE FROM image_models WHERE scope_key = ?`, imageModelScopeKey(id)); err != nil {
		return fmt.Errorf("delete image models: %w", err)
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
	return ValidateImageEndpoint(endpoint)
}

func imageProfileDBError(err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "unique") {
		return fmt.Errorf("已有同名中转站配置，请换一个名称")
	}
	return fmt.Errorf("保存中转站配置失败: %w", err)
}
