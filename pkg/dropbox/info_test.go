package dropbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeRemotePath(t *testing.T) {
	// Create a real temp directory tree so EvalSymlinks works.
	root := t.TempDir()
	subDir := filepath.Join(root, "Music", "Rock")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		localAbs    string
		dropboxRoot string
		want        string
		wantErr     bool
	}{
		{
			name:        "equal paths",
			localAbs:    root,
			dropboxRoot: root,
			want:        "",
		},
		{
			name:        "sub-path",
			localAbs:    subDir,
			dropboxRoot: root,
			want:        "/Music/Rock",
		},
		{
			name:        "unrelated paths",
			localAbs:    t.TempDir(),
			dropboxRoot: root,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ComputeRemotePath(tt.localAbs, tt.dropboxRoot)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ComputeRemotePath() = %q, want %q", got, tt.want)
			}
		})
	}
}
