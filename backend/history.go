package main

import (
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
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", false
	}
	return image, true
}
