package fileutil

import (
	"fmt"
	"os"
)

// CreateFile writes content to path, creating parent directories as needed.
func CreateFile(filePath, content string) error {
	if err := os.MkdirAll(dirOf(filePath), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err = file.WriteString(content); err != nil {
		return err
	}

	fmt.Println("✅ Created file:", filePath)
	return nil
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == os.PathSeparator {
			return path[:i]
		}
	}
	return "."
}
