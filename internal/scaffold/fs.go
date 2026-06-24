package scaffold

import (
	"os"
	"path/filepath"
)

func joinPath(elem ...string) string {
	return filepath.Join(elem...)
}

func mkdirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}
