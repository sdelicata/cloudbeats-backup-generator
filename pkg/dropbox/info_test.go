package dropbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeRemotePath(t *testing.T) {
	t.Parallel()

	// Create a real temp directory tree so EvalSymlinks works.
	root := t.TempDir()
	subDir := filepath.Join(root, "Music", "Rock")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := ComputeRemotePath(test.localAbs, test.dropboxRoot)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}
