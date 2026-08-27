package store

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

const (
	DefaultModel     = "gpt-image-2"
	GrokImageModel   = "grok-imagine-image-2.0"
	GeminiImageModel = "gemini3.1-flash-image"
	defaultModel     = DefaultModel
	defaultSize      = "720x1280"
)

var builtInModelValues = []string{
	DefaultModel,
	GrokImageModel,
	GeminiImageModel,
}

func isBuiltInModel(modelName string) bool {
	modelName = strings.TrimSpace(modelName)
	for _, value := range builtInModelValues {
		if strings.EqualFold(value, modelName) {
			return true
		}
	}
	return false
}

func normalizeCustomModelName(modelName string) (string, error) {
	for _, char := range modelName {
		if unicode.IsControl(char) {
			return "", fmt.Errorf("模型名称不能包含控制字符")
		}
	}
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return "", fmt.Errorf("请填写模型名称")
	}
	if len([]rune(modelName)) > maxModelName {
		return "", fmt.Errorf("模型名称不能超过 %d 个字符", maxModelName)
	}
	return modelName, nil
}

type ModelOption struct {
	ID      string `json:"id"`
	Value   string `json:"value"`
	Label   string `json:"label"`
	BuiltIn bool   `json:"built_in"`
}

func FixedModelOptions() []ModelOption {
	options := make([]ModelOption, 0, len(builtInModelValues))
	for _, model := range builtInModelValues {
		options = append(options, ModelOption{
			ID:      "builtin:" + model,
			Value:   model,
			Label:   model,
			BuiltIn: true,
		})
	}
	return options
}

func (d *appDatabase) listImageModels(scopeKey string) ([]imageModel, error) {
	rows, err := d.db.Query(`
		SELECT id, scope_key, model, created_at
		FROM image_models
		WHERE scope_key = ?
		ORDER BY created_at ASC, id ASC
	`, scopeKey)
	if err != nil {
		return nil, fmt.Errorf("list image models: %w", err)
	}
	defer rows.Close()
	models := make([]imageModel, 0, maxCustomModelCount)
	for rows.Next() {
		var model imageModel
		if err := rows.Scan(&model.ID, &model.ScopeKey, &model.Model, &model.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan image model: %w", err)
		}
		models = append(models, model)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read image models: %w", err)
	}
	return models, nil
}

func (d *appDatabase) createImageModel(scopeKey, modelName string) (imageModel, error) {
	modelName, err := normalizeCustomModelName(modelName)
	if err != nil {
		return imageModel{}, err
	}
	if isBuiltInModel(modelName) {
		return imageModel{}, fmt.Errorf("固定模型无需添加")
	}
	var count int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM image_models WHERE scope_key = ?`, scopeKey).Scan(&count); err != nil {
		return imageModel{}, fmt.Errorf("count image models: %w", err)
	}
	if count >= maxCustomModelCount {
		return imageModel{}, fmt.Errorf("每个中转站最多保存 %d 个自定义模型", maxCustomModelCount)
	}
	model := imageModel{ID: uuid.NewString(), ScopeKey: scopeKey, Model: modelName, CreatedAt: nowString()}
	if _, err := d.db.Exec(`INSERT INTO image_models (id, scope_key, model, created_at) VALUES (?, ?, ?, ?)`, model.ID, model.ScopeKey, model.Model, model.CreatedAt); err != nil {
		return imageModel{}, imageModelDBError(err)
	}
	return model, nil
}

func (d *appDatabase) deleteImageModel(scopeKey, id string) error {
	result, err := d.db.Exec(`DELETE FROM image_models WHERE scope_key = ? AND id = ?`, scopeKey, strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("delete image model: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check deleted image model: %w", err)
	}
	if count == 0 {
		return errNotFound
	}
	return nil
}

func (d *appDatabase) hasImageModel(scopeKey, modelName string) (bool, error) {
	var count int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM image_models WHERE scope_key = ? AND model = ?`, scopeKey, strings.TrimSpace(modelName)).Scan(&count); err != nil {
		return false, fmt.Errorf("check image model: %w", err)
	}
	return count > 0, nil
}

func imageModelScopeKey(profileID string) string {
	if strings.TrimSpace(profileID) == "" {
		return defaultModelScope
	}
	return "profile:" + strings.TrimSpace(profileID)
}

func imageModelDBError(err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "unique") {
		return fmt.Errorf("已有同名模型，请换一个名称")
	}
	return fmt.Errorf("保存自定义模型失败: %w", err)
}
