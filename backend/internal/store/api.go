package store

import (
	"fmt"
	"net/url"
	"strings"
)

type Database = appDatabase
type ImageSettings = imageSettings
type ImageProfile = imageProfile
type ImageModel = imageModel
type PromptPreset = promptPreset
type PromptPresetDraft = promptPresetDraft
type PromptPresetImport = promptPresetImport
type ImageTask = imageTask
type HistoryPage = historyPage

var ErrNotFound = errNotFound

func OpenDatabase(path string) (*Database, error) {
	return openDatabase(path)
}

func (d *Database) GetImageSettings() (ImageSettings, error) { return d.getImageSettings() }
func (d *Database) SaveImageSettings(endpoint, apiKey string) (ImageSettings, error) {
	return d.saveImageSettings(endpoint, apiKey)
}
func (d *Database) ListImageProfiles() ([]ImageProfile, error) { return d.listImageProfiles() }
func (d *Database) GetActiveImageProfile() (ImageProfile, bool, error) {
	return d.getActiveImageProfile()
}
func (d *Database) CreateImageProfile(name, endpoint, apiKey string) (ImageProfile, error) {
	return d.createImageProfile(name, endpoint, apiKey)
}
func (d *Database) UpdateImageProfile(id, name, endpoint, apiKey string) (ImageProfile, error) {
	return d.updateImageProfile(id, name, endpoint, apiKey)
}
func (d *Database) ActivateImageProfile(id string) (ImageProfile, error) {
	return d.activateImageProfile(id)
}
func (d *Database) DeleteImageProfile(id string) error { return d.deleteImageProfile(id) }

func (d *Database) ListImageModels(scopeKey string) ([]ImageModel, error) {
	return d.listImageModels(scopeKey)
}
func (d *Database) CreateImageModel(scopeKey, modelName string) (ImageModel, error) {
	return d.createImageModel(scopeKey, modelName)
}
func (d *Database) DeleteImageModel(scopeKey, id string) error {
	return d.deleteImageModel(scopeKey, id)
}
func (d *Database) HasImageModel(scopeKey, modelName string) (bool, error) {
	return d.hasImageModel(scopeKey, modelName)
}

func (d *Database) ListPresets() ([]PromptPreset, error) { return d.listPresets() }
func (d *Database) CreatePreset(input PromptPresetDraft) (PromptPreset, error) {
	return d.createPreset(input)
}
func (d *Database) UpdatePreset(id string, input PromptPresetDraft) (PromptPreset, error) {
	return d.updatePreset(id, input)
}
func (d *Database) DeletePreset(id string) error { return d.deletePreset(id) }
func (d *Database) ImportPresets(inputs []PromptPresetImport) (int, error) {
	return d.importPresets(inputs)
}

func (d *Database) CreateTask(mode string, input ImageTaskInput) (string, error) {
	return d.createTask(mode, input)
}
func (d *Database) CompleteTask(id, image string) error { return d.completeTask(id, image) }
func (d *Database) HistoryImageData(id string) (string, error) {
	return d.historyImageData(id)
}
func (d *Database) ListTasks(limit int, cursor string) (HistoryPage, error) {
	return d.listTasks(limit, cursor)
}
func (d *Database) DeleteTask(id string) error          { return d.deleteTask(id) }
func (d *Database) ListImageSources() ([]string, error) { return d.listImageSources() }

func ImageModelScopeKey(profileID string) string { return imageModelScopeKey(profileID) }
func IsBuiltInModel(modelName string) bool       { return isBuiltInModel(modelName) }

func NormalizeImageEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" || strings.Contains(endpoint, "://") {
		return endpoint
	}
	return "https://" + endpoint
}

func ValidateImageEndpoint(endpoint string) error {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("endpoint must be a valid http or https url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("endpoint must use http or https")
	}
	return nil
}
