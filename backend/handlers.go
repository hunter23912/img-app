package main

import "strings"

func normalizeImageRequest(input *generateRequest) {
	input.Model = strings.TrimSpace(input.Model)
	input.Prompt = strings.TrimSpace(input.Prompt)
	input.Size = strings.TrimSpace(input.Size)
	input.Quality = strings.TrimSpace(input.Quality)

	if input.Model == "" {
		input.Model = defaultModel
	}
	if input.Quality == "" {
		input.Quality = "auto"
	}
}
