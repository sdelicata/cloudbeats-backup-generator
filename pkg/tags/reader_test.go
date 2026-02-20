package tags

import (
	"testing"
)

func TestParseYear(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int
	}{
		{"four-digit year", "2023", 2023},
		{"ISO date", "2023-05-15", 2023},
		{"short string", "99", 0},
		{"empty", "", 0},
		{"non-numeric prefix", "abcd", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseYear(tt.s); got != tt.want {
				t.Errorf("parseYear(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}

func TestParseSlashNumber(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		fallback int
		want     int
	}{
		{"simple number", "3", -1, 3},
		{"slash format", "3/12", -1, 3},
		{"with spaces", " 5 / 10 ", -1, 5},
		{"non-numeric", "abc", -1, -1},
		{"empty string", "", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseSlashNumber(tt.s, tt.fallback); got != tt.want {
				t.Errorf("parseSlashNumber(%q, %d) = %d, want %d", tt.s, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestFilenameWithoutExt(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"with extension", "/music/song.mp3", "song"},
		{"no extension", "/music/README", "README"},
		{"multiple dots", "/music/my.great.song.flac", "my.great.song"},
		{"just filename", "track.ogg", "track"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filenameWithoutExt(tt.path); got != tt.want {
				t.Errorf("filenameWithoutExt(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
