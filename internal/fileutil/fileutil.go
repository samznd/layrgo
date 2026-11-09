// Package fileutil provides small filesystem helpers for writing generated files.
package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// CreateFile writes content to path, creating parent directories as needed.
//
// It deliberately reports nothing to stdout: callers decide how (and whether) to
// surface progress, which keeps this usable from tests and from quiet code paths.
func CreateFile(filePath, content string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("create parent dir for %s: %w", filePath, err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}
	return nil
}
