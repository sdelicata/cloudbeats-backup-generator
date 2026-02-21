package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFrom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		want    *Credentials
		wantErr string
	}{
		{
			name: "file does not exist",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.json")
			},
			want: nil,
		},
		{
			name: "valid credentials",
			setup: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "creds.json")
				data := `{"app_key":"key1","app_secret":"secret1","refresh_token":"token1"}`
				require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
				return path
			},
			want: &Credentials{
				AppKey:       "key1",
				AppSecret:    "secret1",
				RefreshToken: "token1",
			},
		},
		{
			name: "invalid JSON",
			setup: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "bad.json")
				require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))
				return path
			},
			wantErr: "parsing credentials file",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			path := test.setup(t)
			got, err := loadFrom(path)

			if test.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSaveTo(t *testing.T) {
	t.Parallel()

	t.Run("creates directory and writes file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "subdir", "creds.json")
		creds := &Credentials{
			AppKey:       "key1",
			AppSecret:    "secret1",
			RefreshToken: "token1",
		}

		require.NoError(t, saveTo(path, creds))

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

		loaded, err := loadFrom(path)
		require.NoError(t, err)
		assert.Equal(t, creds, loaded)
	})
}
