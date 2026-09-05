package httpapi

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"img-app/backend/internal/history"
	"img-app/backend/internal/imageops"
)

type imageCacheKey struct {
	db     *appDatabase
	source string
}

type imageCacheJob struct {
	done chan struct{}
	data []byte
	err  error
}

var imageCacheMu sync.Mutex
var imageCacheJobs = make(map[imageCacheKey]*imageCacheJob)

func loadCachedImage(ctx context.Context, db *appDatabase, source string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cached, err := db.CachedImageSource(source)
	if err != nil && err != errNotFound {
		return nil, err
	}
	if err == nil && history.IsBase64ImageDataURL(cached) {
		return imageops.DecodeDataURL(cached)
	}

	key := imageCacheKey{db: db, source: source}
	imageCacheMu.Lock()
	job, exists := imageCacheJobs[key]
	if !exists {
		job = &imageCacheJob{done: make(chan struct{})}
		imageCacheJobs[key] = job
		go func() {
			defer func() {
				imageCacheMu.Lock()
				delete(imageCacheJobs, key)
				close(job.done)
				imageCacheMu.Unlock()
			}()
			// 浏览器断开只结束本次等待；后台缓存有独立期限。同一张图的下载复用已有任务。
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			cached, err := db.CachedImageSource(source)
			if err != nil {
				job.err = err
				return
			}
			if history.IsBase64ImageDataURL(cached) {
				job.data, job.err = imageops.DecodeDataURL(cached)
				return
			}
			registry := imageops.NewSourceRegistry()
			registry.Trust(source)
			job.data, job.err = imageops.LoadImageWithContext(cacheCtx, source, registry)
			if job.err != nil {
				return
			}
			encoded, err := imageops.DataURL(job.data)
			if err != nil {
				job.err = err
				return
			}
			if err := db.CacheImageSource(source, encoded); err != nil {
				slog.Warn("cache downloaded image failed", "error", err)
			}
		}()
	}
	imageCacheMu.Unlock()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-job.done:
		return job.data, job.err
	}
}
