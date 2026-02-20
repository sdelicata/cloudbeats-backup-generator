package scanner

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/simon/cloudbeats-backup-generator/pkg/dropbox"
)

// IsAudioFile reports whether the filename has a supported audio extension.
func IsAudioFile(name string) bool {
	return audioExtensions[strings.ToLower(filepath.Ext(name))]
}

// Supported audio file extensions.
var audioExtensions = map[string]bool{
	".mp3":  true,
	".m4a":  true,
	".flac": true,
	".ogg":  true,
	".opus": true,
	".wav":  true,
	".wma":  true,
	".aac":  true,
	".dsf":  true,
	".aiff": true,
	".aif":  true,
	".ape":  true,
	".wv":   true,
	".mpc":  true,
}

// MatchedFile represents a local file matched to its Dropbox entry.
type MatchedFile struct {
	LocalPath string
	Entry     dropbox.Entry
}

// ScanResult holds the result of matching local files against Dropbox entries.
type ScanResult struct {
	Matched          []MatchedFile
	UnmatchedLocal   []string
	UnmatchedDropbox []dropbox.Entry
}

// ScanLocal walks the directory recursively and returns paths of audio files.
func ScanLocal(dir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if audioExtensions[ext] {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

// Match matches local files against Dropbox entries by relative path.
// remotePath is the Dropbox remote path prefix (e.g. "/Music" or "" for root).
// localDir is the local directory that was scanned.
func Match(localDir, remotePath string, localFiles []string, entries []dropbox.Entry) ScanResult {
	// Build lookup from Dropbox entries: lowercase path → entry
	dbLookup := make(map[string]dropbox.Entry, len(entries))
	for _, e := range entries {
		dbLookup[e.PathLower] = e
	}

	matched := make(map[string]bool) // tracks which Dropbox paths were matched
	var result ScanResult

	remotePrefix := strings.ToLower(remotePath)

	for _, localPath := range localFiles {
		rel, err := filepath.Rel(localDir, localPath)
		if err != nil {
			result.UnmatchedLocal = append(result.UnmatchedLocal, localPath)
			continue
		}

		// NFC normalize the local relative path (macOS uses NFD)
		nfcRel := norm.NFC.String(rel)
		// Build the lookup key: lowercase(remotePath/nfcRel) with forward slashes
		key := remotePrefix + "/" + strings.ToLower(filepath.ToSlash(nfcRel))

		if entry, ok := dbLookup[key]; ok {
			result.Matched = append(result.Matched, MatchedFile{
				LocalPath: localPath,
				Entry:     entry,
			})
			matched[key] = true
		} else {
			result.UnmatchedLocal = append(result.UnmatchedLocal, localPath)
		}
	}

	// Find unmatched Dropbox entries (audio files only)
	for key, entry := range dbLookup {
		if !matched[key] {
			ext := strings.ToLower(filepath.Ext(entry.Name))
			if audioExtensions[ext] {
				result.UnmatchedDropbox = append(result.UnmatchedDropbox, entry)
			}
		}
	}

	return result
}
