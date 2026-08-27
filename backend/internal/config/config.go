package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultEndpoint  = "https://task-api-1-cn.65535.space"
	DefaultModel     = "gpt-image-2"
	DefaultSize      = "720x1280"
	GrokImageModel   = "grok-imagine-image-2.0"
	GeminiImageModel = "gemini3.1-flash-image"
)

type Config struct {
	Endpoint     string
	APIKey       string
	Addr         string
	DatabasePath string
}

type ResolvedImageSettings struct {
	Endpoint string
	APIKey   string
}

func Load() (Config, error) {
	dotEnv, err := loadDotEnv()
	if err != nil {
		return Config{}, err
	}

	endpoint := configuredValue("IMG_ENDPOINT", dotEnv)
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}

	addr := configuredValue("APP_ADDR", dotEnv)
	if addr == "" {
		addr = "localhost:8080"
	}

	apiKey := configuredValue("IMG_API_KEY", dotEnv)

	return Config{
		Endpoint:     endpoint,
		APIKey:       apiKey,
		Addr:         addr,
		DatabasePath: strings.TrimSpace(dotEnv["APP_DB_PATH"]),
	}, nil
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

func DatabasePath(config Config) string {
	if value, ok := os.LookupEnv("APP_DB_PATH"); ok {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(config.DatabasePath)
}

const (
	defaultEndpoint  = DefaultEndpoint
	defaultModel     = DefaultModel
	defaultSize      = DefaultSize
	grokImageModel   = GrokImageModel
	geminiImageModel = GeminiImageModel
)

func loadConfig() (Config, error) { return Load() }
