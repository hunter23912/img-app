package store

import (
	"encoding/base64"
	"net/url"
	"strings"
)

func normalizeHistoryImageURL(raw string) (string, bool) {
	image := strings.TrimSpace(raw)
	parsed, err := url.Parse(image)
	if err == nil && parsed.Scheme == "https" && parsed.Host != "" {
		return image, true
	}
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
	for _, item := range metadata[1:] {
		if strings.EqualFold(strings.TrimSpace(item), "base64") {
			encoded := strings.TrimSpace(value[comma+1:])
			if encoded == "" {
				return false
			}
			_, err := base64.StdEncoding.DecodeString(encoded)
			return err == nil
		}
	}
	return false
}
