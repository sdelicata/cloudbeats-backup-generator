package backup

import (
	"encoding/json"
	"fmt"
	"os"
)

// Write serializes the backup as minified JSON and writes it to the given path.
func Write(path string, b *Backup) error {
	data, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("marshaling backup: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing backup file: %w", err)
	}
	return nil
}
