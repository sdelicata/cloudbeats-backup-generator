package matcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"

	"github.com/simon/cloudbeats-backup-generator/pkg/dropbox"
)

func TestMatch_CaseInsensitive(t *testing.T) {
	t.Parallel()

	localDir := "/music"
	remotePath := "/Music"

	localFiles := []string{"/music/Song.MP3"}
	entries := []dropbox.Entry{
		{Tag: "file", Name: "Song.MP3", PathLower: "/music/song.mp3", PathDisplay: "/Music/Song.MP3"},
	}

	result := Match(localDir, remotePath, localFiles, entries)

	require.Len(t, result.Matched, 1)
	assert.Empty(t, result.UnmatchedLocal)
	assert.Empty(t, result.UnmatchedDropbox)
}

func TestMatch_NFCNormalization(t *testing.T) {
	t.Parallel()

	localDir := "/music"
	remotePath := "/Music"

	// NFD decomposed form of "e" (e + combining acute accent)
	nfdName := norm.NFD.String("cafe.mp3")
	nfcName := norm.NFC.String("cafe.mp3")

	localFiles := []string{"/music/" + nfdName}
	entries := []dropbox.Entry{
		{Tag: "file", Name: nfcName, PathLower: "/music/" + nfcName, PathDisplay: "/Music/" + nfcName},
	}

	result := Match(localDir, remotePath, localFiles, entries)

	require.Len(t, result.Matched, 1)
}

func TestMatch_UnmatchedFilterAudioOnly(t *testing.T) {
	t.Parallel()

	localDir := "/music"
	remotePath := "/Music"

	entries := []dropbox.Entry{
		{Tag: "file", Name: "song.mp3", PathLower: "/music/song.mp3", PathDisplay: "/Music/song.mp3"},
		{Tag: "file", Name: "cover.jpg", PathLower: "/music/cover.jpg", PathDisplay: "/Music/cover.jpg"},
		{Tag: "file", Name: ".DS_Store", PathLower: "/music/.ds_store", PathDisplay: "/Music/.DS_Store"},
	}

	result := Match(localDir, remotePath, nil, entries)

	require.Len(t, result.UnmatchedDropbox, 1)
	assert.Equal(t, "song.mp3", result.UnmatchedDropbox[0].Name)
}

func TestIsAudioFile(t *testing.T) {
	t.Parallel()

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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.want, IsAudioFile(test.file))
		})
	}
}
