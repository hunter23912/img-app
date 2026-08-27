package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

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
	var count, total int
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
