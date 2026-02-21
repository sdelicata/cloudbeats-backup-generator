package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sdelicata/cloudbeats-backup-generator/pkg/tags"
)

var nopLogger = zerolog.Nop()

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T, path string)
		wantLen int
	}{
		{
			name:    "nonexistent file returns empty cache",
			setup:   func(t *testing.T, path string) {},
			wantLen: 0,
		},
		{
			name: "corrupt file returns empty cache",
			setup: func(t *testing.T, path string) {
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, []byte("{invalid json"), 0o644))
			},
			wantLen: 0,
		},
		{
			name: "valid file loads entries",
			setup: func(t *testing.T, path string) {
				entries := map[string]entry{
					"/music/song.mp3": {
						Key:  fileKey{Size: 1000, ModTime: 123456789},
						Meta: tags.AudioMeta{Title: "Song", Artist: "Artist"},
					},
				}
				data, err := json.Marshal(entries)
				require.NoError(t, err)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, data, 0o644))
			},
			wantLen: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, "cache.json")
			test.setup(t, path)

			tc := Load(path, nopLogger)

			assert.Equal(t, test.wantLen, tc.Len())
		})
	}
}

func TestLookup(t *testing.T) {
	t.Parallel()

	// Create a real temp file to stat against.
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.mp3")
	require.NoError(t, os.WriteFile(filePath, []byte("fake audio data"), 0o644))

	info, err := os.Stat(filePath)
	require.NoError(t, err)

	cachedMeta := tags.AudioMeta{
		Title:       "Test Song",
		Artist:      "Test Artist",
		Album:       "Test Album",
		AlbumArtist: "Test Album Artist",
		Genre:       "Rock",
		Year:        2024,
		TrackNumber: 1,
		DiskNumber:  1,
		Duration:    3*time.Minute + 30*time.Second,
	}

	tests := []struct {
		name     string
		entries  map[string]entry
		lookup   string
		wantMeta tags.AudioMeta
		wantOK   bool
	}{
		{
			name:   "hit on matching size and mtime",
			lookup: filePath,
			entries: map[string]entry{
				filePath: {
					Key:  fileKey{Size: info.Size(), ModTime: info.ModTime().UnixNano()},
					Meta: cachedMeta,
				},
			},
			wantMeta: cachedMeta,
			wantOK:   true,
		},
		{
			name:   "miss on different size",
			lookup: filePath,
			entries: map[string]entry{
				filePath: {
					Key:  fileKey{Size: info.Size() + 100, ModTime: info.ModTime().UnixNano()},
					Meta: cachedMeta,
				},
			},
			wantOK: false,
		},
		{
			name:   "miss on different mtime",
			lookup: filePath,
			entries: map[string]entry{
				filePath: {
					Key:  fileKey{Size: info.Size(), ModTime: info.ModTime().UnixNano() + 1},
					Meta: cachedMeta,
				},
			},
			wantOK: false,
		},
		{
			name:    "miss on absent entry",
			lookup:  filePath,
			entries: map[string]entry{},
			wantOK:  false,
		},
		{
			name:   "miss on nonexistent file",
			lookup: filepath.Join(dir, "nonexistent.mp3"),
			entries: map[string]entry{
				filepath.Join(dir, "nonexistent.mp3"): {
					Key:  fileKey{Size: 100, ModTime: 123456789},
					Meta: cachedMeta,
				},
			},
			wantOK: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			tc := &TagCache{entries: test.entries}

			meta, ok := tc.Lookup(test.lookup)

			assert.Equal(t, test.wantOK, ok)
			if test.wantOK {
				assert.Equal(t, test.wantMeta, meta)
			}
		})
	}
}

func TestStoreAndSaveRoundtrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cachePath := filepath.Join(dir, "sub", "cache.json")

	// Create a test file.
	audioFile := filepath.Join(dir, "song.flac")
	require.NoError(t, os.WriteFile(audioFile, []byte("flac content"), 0o644))

	meta := tags.AudioMeta{
		Title:       "My Song",
		Artist:      "The Band",
		Album:       "Greatest Hits",
		AlbumArtist: "The Band",
		Genre:       "Pop",
		Year:        2023,
		TrackNumber: 5,
		DiskNumber:  1,
		Duration:    4*time.Minute + 12*time.Second,
	}

	// Store and save.
	tc := Load(cachePath, nopLogger)
	tc.Store(audioFile, meta)
	require.NoError(t, tc.Save())

	// Reload and verify.
	tc2 := Load(cachePath, nopLogger)
	assert.Equal(t, 1, tc2.Len())

	got, ok := tc2.Lookup(audioFile)
	assert.True(t, ok)
	assert.Equal(t, meta, got)
}

func TestSaveNoop(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")

	tc := Load(cachePath, nopLogger)
	require.NoError(t, tc.Save())

	// File should not be created.
	_, err := os.Stat(cachePath)
	assert.True(t, os.IsNotExist(err))
}
