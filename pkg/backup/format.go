package backup

import (
	"strconv"
)

// Backup represents the top-level structure of a .cbbackup file.
type Backup struct {
	Items     []Item     `json:"items"`
	Playlists []Playlist `json:"playlists"`
}

// Playlist represents a CloudBeats playlist. We generate an empty slice.
type Playlist struct{}

// Item represents a single audio file entry in the backup.
// JSON keys are alphabetically ordered to match the CloudBeats format.
type Item struct {
	AccountID   string   `json:"account_id"`
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	Service     string   `json:"service"`
	Album       string   `json:"tag_album"`
	AlbumArtist string   `json:"tag_albumArtist"`
	Artist      string   `json:"tag_artist"`
	DiskNumber  int      `json:"tag_diskNumber"`
	Duration    Duration `json:"tag_duration"`
	Genre       *string  `json:"tag_genre,omitempty"`
	TagName     string   `json:"tag_name"`
	TrackNumber *int     `json:"tag_trackNumber,omitempty"`
	Year        int      `json:"tag_year"`
}

// Duration is a float64 that always serializes with one decimal place (e.g. 294.0).
type Duration float64

func (d Duration) MarshalJSON() ([]byte, error) {
	s := strconv.FormatFloat(float64(d), 'f', 1, 64)
	return []byte(s), nil
}
