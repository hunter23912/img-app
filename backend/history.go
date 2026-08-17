package main

import (
	"encoding/base64"
	"net/url"
	"strings"
	"sync"
)

const maxImageHistory = 5

type imageHistory struct {
	mu     sync.RWMutex
	images []string
}

func newImageHistory() *imageHistory {
	return &imageHistory{images: make([]string, 0, maxImageHistory)}
}

func (h *imageHistory) List() []string {
	if h == nil {
		return []string{}
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	images := make([]string, len(h.images))
	copy(images, h.images)
	return images
}

func (h *imageHistory) Add(image string) {
	if h == nil {
		return
	}

	image, ok := normalizeHistoryImageURL(image)
	if !ok {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	images := make([]string, 0, maxImageHistory)
	images = append(images, image)
	for _, existing := range h.images {
		if existing != image {
			images = append(images, existing)
		}
		if len(images) == maxImageHistory {
			break
		}
	}
	h.images = images
}

func (h *imageHistory) Remove(image string) {
	if h == nil {
		return
	}

	image, ok := normalizeHistoryImageURL(image)
	if !ok {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for index, existing := range h.images {
		if existing == image {
			h.images = append(h.images[:index], h.images[index+1:]...)
			return
		}
	}
}

func normalizeHistoryImageURL(raw string) (string, bool) {
	image := strings.TrimSpace(raw)
	parsed, err := url.Parse(image)
	if err == nil && parsed.Scheme == "https" && parsed.Host != "" {
		return image, true
	}

	// SeedVR returns b64_json, which the relay converts to a data URL. Keep
	// these results in the database as well as externally hosted HTTPS images.
	if !isBase64ImageDataURL(image) {
		return "", false
	}
	return image, true
}

func historyImagePath(id string) string {
	return "/api/history/" + url.PathEscape(id) + "/image"
}

func historyImageReference(id, image string) string {
	if isBase64ImageDataURL(image) {
		return historyImagePath(id)
	}
	return image
}

func parseHistoryImagePath(path string) (string, bool) {
	const prefix = "/api/history/"
	const suffix = "/image"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}

	encodedID := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	if encodedID == "" || strings.Contains(encodedID, "/") {
		return "", false
	}
	id, err := url.PathUnescape(encodedID)
	if err != nil || id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

func isBase64ImageDataURL(value string) bool {
	if !strings.HasPrefix(strings.ToLower(value), "data:image/") {
		return false
	}

	comma := strings.IndexByte(value, ',')
	if comma <= len("data:image/") {
		return false
	}
	metadata := strings.Split(value[len("data:"):comma], ";")
	if len(metadata) < 2 || !strings.HasPrefix(strings.ToLower(strings.TrimSpace(metadata[0])), "image/") {
		return false
	}

	isBase64 := false
	for _, item := range metadata[1:] {
		if strings.EqualFold(strings.TrimSpace(item), "base64") {
			isBase64 = true
			break
		}
	}
	if !isBase64 {
		return false
	}

	encoded := strings.TrimSpace(value[comma+1:])
	if encoded == "" {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(encoded)
	return err == nil
}
