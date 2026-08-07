package main

import "testing"

func TestLoadConfigRequiresAPIKey(t *testing.T) {
	t.Setenv("IMG_API_KEY", "")

	if _, err := loadConfig(); err == nil {
		t.Fatal("loadConfig() accepted a missing IMG_API_KEY")
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
