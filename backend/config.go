package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultEndpoint = "https://img-cn.65535.space"
	defaultModel    = "gpt-image-2"
)

type appConfig struct {
	Endpoint string
	APIKey   string
	Addr     string
}

func loadConfig() appConfig {
	loadDotEnv()

	endpoint := strings.TrimSpace(os.Getenv("IMG_ENDPOINT"))
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	addr := strings.TrimSpace(os.Getenv("APP_ADDR"))
	if addr == "" {
		addr = "localhost:8080"
	}

	return appConfig{
		Endpoint: endpoint,
		APIKey:   strings.TrimSpace(os.Getenv("IMG_API_KEY")),
		Addr:     addr,
	}
}

func loadDotEnv() {
	// 兼容从项目根目录或 backend 目录启动 Go 服务，两种位置都尝试读取 .env。
	for _, path := range []string{".env", filepath.Join("backend", ".env")} {
		if err := applyDotEnvFile(path); err == nil {
			log.Printf("loaded config from %s", path)
			return
		}
	}
}

func applyDotEnvFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)

		if key == "" {
			continue
		}

		// 系统环境变量优先级更高，避免本地 .env 覆盖服务器上的真实配置。
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return nil
}
