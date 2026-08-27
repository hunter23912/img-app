package httpapi

import (
	"img-app/backend/internal/config"
	"img-app/backend/internal/history"
	"img-app/backend/internal/imageops"
	"img-app/backend/internal/store"
)

type appDatabase = store.Database
type imageHistory = history.History
type imageProfile = store.ImageProfile
type imageModel = store.ImageModel
type promptPreset = store.PromptPreset
type promptPresetDraft = store.PromptPresetDraft
type promptPresetImport = store.PromptPresetImport
type imageTask = store.ImageTask
type historyPage = store.HistoryPage

var errNotFound = store.ErrNotFound

type appConfig struct {
	Endpoint            string
	APIKey              string
	Addr                string
	ImageSourceRegistry *imageops.SourceRegistry
	ImageHistory        *history.History
	Database            *store.Database
}

type ServerConfig = appConfig
type resolvedImageSettings = config.ResolvedImageSettings
type ResolvedImageSettings = config.ResolvedImageSettings

const (
	defaultEndpoint  = config.DefaultEndpoint
	defaultModel     = config.DefaultModel
	defaultSize      = config.DefaultSize
	grokImageModel   = config.GrokImageModel
	geminiImageModel = config.GeminiImageModel
)

func (config appConfig) effectiveImageSettings() (resolvedImageSettings, error) {
	settings := resolvedImageSettings{Endpoint: config.Endpoint, APIKey: config.APIKey}
	if settings.Endpoint == "" {
		settings.Endpoint = defaultEndpoint
	}
	if config.Database == nil {
		return settings, nil
	}
	profile, found, err := config.Database.GetActiveImageProfile()
	if err != nil {
		return resolvedImageSettings{}, err
	}
	if found {
		if profile.Endpoint != "" {
			settings.Endpoint = profile.Endpoint
		}
		settings.APIKey = profile.APIKey
		return settings, nil
	}
	saved, err := config.Database.GetImageSettings()
	if err != nil {
		return resolvedImageSettings{}, err
	}
	if saved.Endpoint != "" {
		settings.Endpoint = saved.Endpoint
	}
	if saved.APIKey != "" {
		settings.APIKey = saved.APIKey
	}
	return settings, nil
}

func (config appConfig) EffectiveImageSettings() (ResolvedImageSettings, error) {
	return config.effectiveImageSettings()
}

func (config appConfig) imageModelScope() (string, error) {
	if config.Database == nil {
		return "default", nil
	}
	profile, found, err := config.Database.GetActiveImageProfile()
	if err != nil {
		return "", err
	}
	if found {
		return store.ImageModelScopeKey(profile.ID), nil
	}
	return "default", nil
}

func (config appConfig) imageModelAvailable(modelName string) (bool, error) {
	if store.IsBuiltInModel(modelName) {
		return true, nil
	}
	if config.Database == nil {
		return false, nil
	}
	scopeKey, err := config.imageModelScope()
	if err != nil {
		return false, err
	}
	return config.Database.HasImageModel(scopeKey, modelName)
}
