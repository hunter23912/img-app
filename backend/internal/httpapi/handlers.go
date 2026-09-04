package httpapi

import (
	"strconv"
	"strings"
)

func normalizeImageRequest(input *generateRequest) {
	input.Model = strings.TrimSpace(input.Model)
	input.Prompt = strings.TrimSpace(input.Prompt)
	input.Size = strings.TrimSpace(input.Size)
	input.Quality = strings.TrimSpace(input.Quality)

	if input.Model == "" {
		input.Model = defaultModel
	}
	if input.Size == "" {
		input.Size = defaultSize
	}
	if input.Quality == "" {
		input.Quality = "auto"
	}
	if input.Moderation == "" {
		input.Moderation = "auto"
	}
	if input.Background == "" {
		input.Background = "auto"
	}
	if input.OutputFormat == "" {
		input.OutputFormat = "png"
	}
	if input.N < 1 {
		input.N = 1
	}
}

func parsePositiveInt(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 1 {
		return 1
	}
	return parsed
}
