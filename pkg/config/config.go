// Package config handles reading and writing stored Dropbox credentials.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	appDir    = "cloudbeats-backup-generator"
	credsFile = "credentials.json"
	dirPerms  = 0o700
	filePerms = 0o600
)

// Credentials holds the Dropbox OAuth2 credentials needed for refresh-token auth.
type Credentials struct {
	AppKey       string `json:"app_key"`
	AppSecret    string `json:"app_secret"`
	RefreshToken string `json:"refresh_token"`
}

// Load reads stored credentials from the default config path.
// Returns (nil, nil) if the file does not exist.
func Load() (*Credentials, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("determining config directory: %w", err)
	}
	return loadFrom(filepath.Join(dir, appDir, credsFile))
}

// Save writes credentials to the default config path.
func Save(creds *Credentials) error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("determining config directory: %w", err)
	}
	return saveTo(filepath.Join(dir, appDir, credsFile), creds)
}

func loadFrom(path string) (*Credentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading credentials file: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parsing credentials file: %w", err)
	}

	return &creds, nil
}

func saveTo(path string, creds *Credentials) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerms); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding credentials: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, filePerms); err != nil {
		return fmt.Errorf("writing credentials file: %w", err)
	}

	return nil
}
