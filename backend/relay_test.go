package main

import "testing"

func TestBuildImagesURL(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		action   string
		want     string
	}{
		{
			name:     "base endpoint",
			endpoint: "https://example.com",
			action:   "generations",
			want:     "https://example.com/v1/images/generations",
		},
		{
			name:     "v1 endpoint",
			endpoint: "https://example.com/v1",
			action:   "edits",
			want:     "https://example.com/v1/images/edits",
		},
		{
			name:     "full endpoint",
			endpoint: "https://example.com/v1/images/edits",
			action:   "edits",
			want:     "https://example.com/v1/images/edits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildImagesURL(tt.endpoint, tt.action)
			if err != nil {
				t.Fatalf("buildImagesURL returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("buildImagesURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildImagesURLRejectsInvalidEndpoint(t *testing.T) {
	if _, err := buildImagesURL("file:///tmp/image", "edits"); err == nil {
		t.Fatal("buildImagesURL accepted non-http endpoint")
	}
}
