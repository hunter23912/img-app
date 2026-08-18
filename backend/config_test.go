package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigAllowsMissingAPIKey(t *testing.T) {
	t.Setenv("IMG_API_KEY", "")

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() returned an error: %v", err)
	}
	if config.APIKey != "" {
		t.Fatalf("loadConfig() APIKey = %q, want empty", config.APIKey)
	}
}

func TestLoadConfigReadsAPIKeyFromEnvironment(t *testing.T) {
	t.Setenv("IMG_API_KEY", "env-only-key")
	t.Setenv("IMG_ENDPOINT", "")
	t.Setenv("APP_ADDR", "")

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() returned an error: %v", err)
	}
	if config.APIKey != "env-only-key" {
		t.Fatalf("loadConfig() APIKey = %q, want %q", config.APIKey, "env-only-key")
	}
	if config.Endpoint != defaultEndpoint {
		t.Fatalf("loadConfig() Endpoint = %q, want %q", config.Endpoint, defaultEndpoint)
	}
}

func TestLoadConfigReadsDotEnvAndPrefersProcessEnvironment(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, ".env"), []byte("# local settings\n\nIMG_API_KEY=dotenv-key\nIMG_ENDPOINT=https://dotenv.example\nAPP_ADDR=127.0.0.1:9000\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(directory)
	unsetEnvironment(t, "IMG_API_KEY")
	unsetEnvironment(t, "IMG_ENDPOINT")
	unsetEnvironment(t, "APP_ADDR")

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() returned an error: %v", err)
	}
	if config.APIKey != "dotenv-key" || config.Endpoint != "https://dotenv.example" || config.Addr != "127.0.0.1:9000" {
		t.Fatalf("loadConfig() from .env = %#v", config)
	}

	t.Setenv("IMG_API_KEY", "process-key")
	t.Setenv("IMG_ENDPOINT", "https://process.example")
	config, err = loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() with process environment returned an error: %v", err)
	}
	if config.APIKey != "process-key" || config.Endpoint != "https://process.example" {
		t.Fatalf("process environment did not take precedence: %#v", config)
	}
}

func TestLoadConfigStartsWithoutDotEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	unsetEnvironment(t, "IMG_API_KEY")
	unsetEnvironment(t, "IMG_ENDPOINT")
	unsetEnvironment(t, "APP_ADDR")

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() without .env returned an error: %v", err)
	}
	if config.Endpoint != defaultEndpoint {
		t.Fatalf("loadConfig() Endpoint = %q, want %q", config.Endpoint, defaultEndpoint)
	}
}

func TestLoadConfigFindsBackendDotEnvFromProjectRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "backend"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "backend", ".env"), []byte("IMG_API_KEY=backend-dotenv-key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	unsetEnvironment(t, "IMG_API_KEY")

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() returned an error: %v", err)
	}
	if config.APIKey != "backend-dotenv-key" {
		t.Fatalf("loadConfig() APIKey = %q, want backend .env value", config.APIKey)
	}
}

func unsetEnvironment(t *testing.T, key string) {
	t.Helper()
	previous, wasSet := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if wasSet {
			_ = os.Setenv(key, previous)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}
