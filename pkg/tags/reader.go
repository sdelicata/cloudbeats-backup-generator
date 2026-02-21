// Package tags reads audio metadata from local files using taglib.
package tags

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sentriz/audiotags"
)

// AudioMeta holds extracted metadata from an audio file.
type AudioMeta struct {
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	Genre       string
	Year        int
	TrackNumber int // -1 means absent
	DiskNumber  int
	Duration    time.Duration
}

// ReadFile extracts audio metadata from the file at path.
// On failure, returns defaults ("Unknown" for artist/album, filename for title, 0 for duration).
func ReadFile(path string) (meta AudioMeta, err error) {
	meta = AudioMeta{
		Title:       filenameWithoutExt(path),
		Artist:      "Unknown",
		Album:       "Unknown",
		AlbumArtist: "Unknown",
		TrackNumber: -1,
		DiskNumber:  1,
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("taglib panicked: %v", r)
		}
	}()

	f, openErr := audiotags.Open(path)
	if openErr != nil || f == nil {
		return meta, nil
	}
	defer f.Close()

	tags := f.ReadTags()
	props := f.ReadAudioProperties()

	if v := firstTag(tags, "title"); v != "" {
		meta.Title = v
	}
	if v := firstTag(tags, "artist"); v != "" {
		meta.Artist = v
	}
	if v := firstTag(tags, "album"); v != "" {
		meta.Album = v
	}
	if v := firstTag(tags, "albumartist"); v != "" {
		meta.AlbumArtist = v
	}
	if v := firstTag(tags, "genre"); v != "" {
		meta.Genre = v
	}
	if v := firstTag(tags, "date"); v != "" {
		meta.Year = parseYear(v)
	}
	if v := firstTag(tags, "tracknumber"); v != "" {
		meta.TrackNumber = parseSlashNumber(v, -1)
	}
	if v := firstTag(tags, "discnumber"); v != "" {
		meta.DiskNumber = parseSlashNumber(v, 1)
	}

	if props != nil {
		meta.Duration = time.Duration(props.LengthMs) * time.Millisecond
	}

	return meta, nil
}

func firstTag(tags map[string][]string, key string) string {
	if vals, ok := tags[key]; ok && len(vals) > 0 && vals[0] != "" {
		return vals[0]
	}
	return ""
}

// parseYear extracts a 4-digit year from a string that may be a full ISO date.
func parseYear(s string) int {
	if len(s) >= 4 {
		if y, err := strconv.Atoi(s[:4]); err == nil {
			return y
		}
	}
	return 0
}

// parseSlashNumber parses "3/12" format, returning the number before the slash.
func parseSlashNumber(s string, fallback int) int {
	s, _, _ = strings.Cut(s, "/")
	s = strings.TrimSpace(s)
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return fallback
}

func filenameWithoutExt(path string) string {
	name := filepath.Base(path)
	return strings.TrimSuffix(name, filepath.Ext(name))
}
