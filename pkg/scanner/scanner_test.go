package scanner

import (
	"testing"

	"golang.org/x/text/unicode/norm"

	"github.com/simon/cloudbeats-backup-generator/pkg/dropbox"
)

func TestMatch_CaseInsensitive(t *testing.T) {
	localDir := "/music"
	remotePath := "/Music"

	localFiles := []string{"/music/Song.MP3"}
	entries := []dropbox.Entry{
		{Tag: "file", Name: "Song.MP3", PathLower: "/music/song.mp3", PathDisplay: "/Music/Song.MP3"},
	}

	result := Match(localDir, remotePath, localFiles, entries)

	if len(result.Matched) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result.Matched))
	}
	if len(result.UnmatchedLocal) != 0 {
		t.Errorf("expected 0 unmatched local, got %d", len(result.UnmatchedLocal))
	}
	if len(result.UnmatchedDropbox) != 0 {
		t.Errorf("expected 0 unmatched dropbox, got %d", len(result.UnmatchedDropbox))
	}
}

func TestMatch_NFCNormalization(t *testing.T) {
	localDir := "/music"
	remotePath := "/Music"

	// NFD decomposed form of "é" (e + combining acute accent)
	nfdName := norm.NFD.String("café.mp3")
	nfcName := norm.NFC.String("café.mp3")

	localFiles := []string{"/music/" + nfdName}
	entries := []dropbox.Entry{
		{Tag: "file", Name: nfcName, PathLower: "/music/" + nfcName, PathDisplay: "/Music/" + nfcName},
	}

	result := Match(localDir, remotePath, localFiles, entries)

	if len(result.Matched) != 1 {
		t.Fatalf("expected 1 match after NFC normalization, got %d", len(result.Matched))
	}
}

func TestMatch_UnmatchedFilterAudioOnly(t *testing.T) {
	localDir := "/music"
	remotePath := "/Music"

	entries := []dropbox.Entry{
		{Tag: "file", Name: "song.mp3", PathLower: "/music/song.mp3", PathDisplay: "/Music/song.mp3"},
		{Tag: "file", Name: "cover.jpg", PathLower: "/music/cover.jpg", PathDisplay: "/Music/cover.jpg"},
		{Tag: "file", Name: ".DS_Store", PathLower: "/music/.ds_store", PathDisplay: "/Music/.DS_Store"},
	}

	result := Match(localDir, remotePath, nil, entries)

	if len(result.UnmatchedDropbox) != 1 {
		t.Fatalf("expected 1 unmatched Dropbox entry (audio only), got %d", len(result.UnmatchedDropbox))
	}
	if result.UnmatchedDropbox[0].Name != "song.mp3" {
		t.Errorf("expected unmatched entry to be song.mp3, got %s", result.UnmatchedDropbox[0].Name)
	}
}

func TestIsAudioFile(t *testing.T) {
	tests := []struct {
		name string
		file string
		want bool
	}{
		{"mp3", "song.mp3", true},
		{"MP3 uppercase", "song.MP3", true},
		{"jpg", "cover.jpg", false},
		{"no extension", "README", false},
		{"flac", "track.flac", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAudioFile(tt.file); got != tt.want {
				t.Errorf("IsAudioFile(%q) = %v, want %v", tt.file, got, tt.want)
			}
		})
	}
}
