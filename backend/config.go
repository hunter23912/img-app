package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultEndpoint = "https://task-api-1-cn.65535.space"
	defaultModel    = "gpt-image-2-lite"
	defaultSize     = "720x1280"
	seedVRModel     = "seedvr2-7b"
)

type appConfig struct {
	Endpoint            string
	APIKey              string
	Addr                string
	ImageSourceRegistry *imageSourceRegistry
	ImageHistory        *imageHistory
	Database            *appDatabase
}

type resolvedImageSettings struct {
	Endpoint string
	APIKey   string
}

func loadConfig() (appConfig, error) {
	dotEnv, err := loadDotEnv()
	if err != nil {
		return appConfig{}, err
	}

	endpoint := configuredValue("IMG_ENDPOINT", dotEnv)
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	addr := configuredValue("APP_ADDR", dotEnv)
	if addr == "" {
		addr = "localhost:8080"
	}

	apiKey := configuredValue("IMG_API_KEY", dotEnv)

	return appConfig{
		Endpoint: endpoint,
		APIKey:   apiKey,
		Addr:     addr,
	}, nil
}

func (config appConfig) effectiveImageSettings() (resolvedImageSettings, error) {
	settings := resolvedImageSettings{
		Endpoint: strings.TrimSpace(config.Endpoint),
		APIKey:   strings.TrimSpace(config.APIKey),
	}
	if settings.Endpoint == "" {
		settings.Endpoint = defaultEndpoint
	}

	if config.Database == nil {
		return settings, nil
	}
	profile, found, err := config.Database.getActiveImageProfile()
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

	// Keep the legacy single-setting table as a fallback for databases created
	// before image profiles were introduced. New writes go through profiles.
	saved, err := config.Database.getImageSettings()
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

func configuredValue(key string, dotEnv map[string]string) string {
	if value, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(dotEnv[key])
}

func loadDotEnv() (map[string]string, error) {
	values := make(map[string]string)
	paths := []string{".env"}

	// When the process starts from the project root, also support the backend's
	// local .env file. From backend/ itself the first path already covers it.
	if _, err := os.Stat(filepath.Join("backend", ".env")); err == nil {
		paths = append(paths, filepath.Join("backend", ".env"))
	}

	for _, path := range paths {
		file, err := os.Open(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}

		if err := parseDotEnv(file, values); err != nil {
			_ = file.Close()
			return nil, err
		}
		if err := file.Close(); err != nil {
			return nil, err
		}
	}

	return values, nil
}

func parseDotEnv(file *os.File, values map[string]string) error {
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := values[key]; exists {
			continue
		}
		values[key] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return scanner.Err()
}

func databasePathFromEnvironment() string {
	if value, ok := os.LookupEnv("APP_DB_PATH"); ok {
		return strings.TrimSpace(value)
	}
	dotEnv, err := loadDotEnv()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(dotEnv["APP_DB_PATH"])
}
