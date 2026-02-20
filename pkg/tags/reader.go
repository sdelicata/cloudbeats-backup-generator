package tags

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.senan.xyz/taglib"
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
func ReadFile(path string) AudioMeta {
	tags, tagsErr := taglib.ReadTags(path)
	props, propsErr := taglib.ReadProperties(path)

	meta := AudioMeta{
		Title:       filenameWithoutExt(path),
		Artist:      "Unknown",
		Album:       "Unknown",
		AlbumArtist: "Unknown",
		TrackNumber: -1,
		DiskNumber:  1,
	}

	if tagsErr != nil {
		return meta
	}

	if v := firstTag(tags, "TITLE"); v != "" {
		meta.Title = v
	}
	if v := firstTag(tags, "ARTIST"); v != "" {
		meta.Artist = v
	}
	if v := firstTag(tags, "ALBUM"); v != "" {
		meta.Album = v
	}
	if v := firstTag(tags, "ALBUMARTIST"); v != "" {
		meta.AlbumArtist = v
	}
	if v := firstTag(tags, "GENRE"); v != "" {
		meta.Genre = v
	}
	if v := firstTag(tags, "DATE"); v != "" {
		meta.Year = parseYear(v)
	}
	if v := firstTag(tags, "TRACKNUMBER"); v != "" {
		meta.TrackNumber = parseSlashNumber(v, -1)
	}
	if v := firstTag(tags, "DISCNUMBER"); v != "" {
		meta.DiskNumber = parseSlashNumber(v, 1)
	}

	if propsErr == nil {
		meta.Duration = props.Length
	}

	return meta
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
