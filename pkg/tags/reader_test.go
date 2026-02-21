package tags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseYear(t *testing.T) {
	t.Parallel()

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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.want, parseYear(test.s))
		})
	}
}

func TestParseSlashNumber(t *testing.T) {
	t.Parallel()

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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.want, parseSlashNumber(test.s, test.fallback))
		})
	}
}

func TestFilenameWithoutExt(t *testing.T) {
	t.Parallel()

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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.want, filenameWithoutExt(test.path))
		})
	}
}
