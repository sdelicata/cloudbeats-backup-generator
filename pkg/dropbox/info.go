package dropbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// infoJSON represents the structure of Dropbox's info.json file.
type infoJSON struct {
	Personal *infoAccount `json:"personal"`
	Business *infoAccount `json:"business"`
}

type infoAccount struct {
	Path string `json:"path"`
}

// DetectRootPath finds the local Dropbox root path by reading info.json.
// It searches ~/.dropbox/info.json then ~/Library/Application Support/Dropbox/info.json.
func DetectRootPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}

	candidates := []string{
		filepath.Join(home, ".dropbox", "info.json"),
		filepath.Join(home, "Library", "Application Support", "Dropbox", "info.json"),
	}

	for _, path := range candidates {
		root, err := readInfoJSON(path)
		if err == nil {
			return root, nil
		}
	}

	return "", fmt.Errorf("dropbox desktop does not appear to be installed. " +
		"Verify that Dropbox Desktop is installed and that info.json exists " +
		"(checked ~/.dropbox/info.json and ~/Library/Application Support/Dropbox/info.json)")
}

func readInfoJSON(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var info infoJSON
	if err := json.Unmarshal(data, &info); err != nil {
		return "", fmt.Errorf("parsing %s: %w", path, err)
	}

	if info.Personal != nil && info.Personal.Path != "" {
		return info.Personal.Path, nil
	}
	if info.Business != nil && info.Business.Path != "" {
		return info.Business.Path, nil
	}

	return "", fmt.Errorf("no personal or business path found in %s", path)
}

// ComputeRemotePath computes the Dropbox remote path from a local absolute path
// and the Dropbox root path. Both paths are resolved via EvalSymlinks for consistency.
// Returns "" if localAbs equals the root (Dropbox API expects "" for root, not "/").
func ComputeRemotePath(localAbs, dropboxRoot string) (string, error) {
	resolvedLocal, err := filepath.EvalSymlinks(localAbs)
	if err != nil {
		return "", fmt.Errorf("resolving local path %s: %w", localAbs, err)
	}

	resolvedRoot, err := filepath.EvalSymlinks(dropboxRoot)
	if err != nil {
		return "", fmt.Errorf("resolving Dropbox root %s: %w", dropboxRoot, err)
	}

	// Normalize both to clean paths
	resolvedLocal = filepath.Clean(resolvedLocal)
	resolvedRoot = filepath.Clean(resolvedRoot)

	if resolvedLocal == resolvedRoot {
		return "", nil
	}

	// Ensure local is under root
	prefix := resolvedRoot + string(filepath.Separator)
	if !strings.HasPrefix(resolvedLocal, prefix) {
		return "", fmt.Errorf("the --local folder (%s) is not located inside the Dropbox folder (%s). Verify the path", localAbs, dropboxRoot)
	}

	rel := resolvedLocal[len(resolvedRoot):]
	// Convert to forward slashes for Dropbox API
	remote := filepath.ToSlash(rel)
	return remote, nil
}
