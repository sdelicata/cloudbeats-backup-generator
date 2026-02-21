// Package cache provides a persistent tag cache to avoid re-parsing unchanged audio files.
package cache

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"

	"github.com/sdelicata/cloudbeats-backup-generator/pkg/tags"
)

type fileKey struct {
	Size    int64 `json:"size"`
	ModTime int64 `json:"mod_time"` // UnixNano
}

type entry struct {
	Key  fileKey        `json:"key"`
	Meta tags.AudioMeta `json:"meta"`
}

// TagCache caches audio metadata keyed by file path and validated by size+mtime.
type TagCache struct {
	path    string
	entries map[string]entry // key = absolute file path
	dirty   bool
	logger  zerolog.Logger
}

// Load reads the cache from path. Returns an empty cache on any error.
func Load(path string, logger zerolog.Logger) *TagCache {
	tc := &TagCache{
		path:    path,
		entries: make(map[string]entry),
		logger:  logger,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Warn().Err(err).Msg("reading tag cache file")
		}
		return tc
	}

	if err := json.Unmarshal(data, &tc.entries); err != nil {
		logger.Warn().Err(err).Msg("parsing tag cache file")
		tc.entries = make(map[string]entry)
	}

	return tc
}

// Len returns the number of entries in the cache.
func (tc *TagCache) Len() int {
	return len(tc.entries)
}

// Lookup returns cached metadata if the file's size and mtime match the cached entry.
// It is goroutine-safe (read-only map access + os.Stat).
func (tc *TagCache) Lookup(filePath string) (tags.AudioMeta, bool) {
	e, ok := tc.entries[filePath]
	if !ok {
		return tags.AudioMeta{}, false
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return tags.AudioMeta{}, false
	}

	if info.Size() != e.Key.Size || info.ModTime().UnixNano() != e.Key.ModTime {
		return tags.AudioMeta{}, false
	}

	return e.Meta, true
}

// Store adds or updates a cache entry for the given file.
// It must be called from a single goroutine (after the worker pool completes).
func (tc *TagCache) Store(filePath string, meta tags.AudioMeta) {
	info, err := os.Stat(filePath)
	if err != nil {
		return
	}

	tc.entries[filePath] = entry{
		Key: fileKey{
			Size:    info.Size(),
			ModTime: info.ModTime().UnixNano(),
		},
		Meta: meta,
	}
	tc.dirty = true
}

// Save writes the cache to disk if it has been modified.
func (tc *TagCache) Save() error {
	if !tc.dirty {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(tc.path), 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(tc.entries)
	if err != nil {
		return err
	}

	return os.WriteFile(tc.path, data, 0o644)
}
