package imageops

import (
	"net/url"
	"strings"
	"sync"
)

const maxTrustedImageSources = 256

type SourceRegistry struct {
	mu      sync.RWMutex
	sources map[string]struct{}
	order   []string
}

func NewSourceRegistry() *SourceRegistry {
	return &SourceRegistry{sources: make(map[string]struct{})}
}

func (r *SourceRegistry) Trust(source string) {
	if r == nil {
		return
	}
	source = strings.TrimSpace(source)
	parsed, err := url.Parse(source)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.sources[source]; exists {
		return
	}
	if len(r.order) >= maxTrustedImageSources {
		delete(r.sources, r.order[0])
		r.order = r.order[1:]
	}
	r.sources[source] = struct{}{}
	r.order = append(r.order, source)
}

func (r *SourceRegistry) Contains(source string) bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.sources[strings.TrimSpace(source)]
	return exists
}
