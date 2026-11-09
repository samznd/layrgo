package templates

import (
	"fmt"
	"path/filepath"

	"github.com/samznd/layrgo/internal/fileutil"
)

// LayerConfig holds inputs for internal layer generation.
type LayerConfig struct {
	ProjectPath string
	ProjectName string
	Framework   string
	ORM         string

	// OnFileCreated, when set, is called with each generated file's relative
	// path. Keeps this package free of any opinion about progress output.
	OnFileCreated func(rel string)
}

// WriteLayers generates repository, service, handler, and route files.
func WriteLayers(cfg LayerConfig) error {
	repository, err := RepositoryFile(cfg.ProjectName, cfg.ORM)
	if err != nil {
		return err
	}
	service, err := ServiceFile(cfg.ProjectName)
	if err != nil {
		return err
	}
	handler, err := HandlerFile(cfg.ProjectName, cfg.Framework)
	if err != nil {
		return err
	}
	routes, err := RoutesFile(cfg.ProjectName, cfg.Framework)
	if err != nil {
		return err
	}

	files := []struct {
		rel     string
		content string
	}{
		{"repositories/repository.go", repository},
		{"services/service.go", service},
		{"handlers/handler.go", handler},
		{"routes/routes.go", routes},
	}

	for _, f := range files {
		path := filepath.Join(cfg.ProjectPath, "internal", f.rel)
		if err := fileutil.CreateFile(path, f.content); err != nil {
			return fmt.Errorf("create %s: %w", f.rel, err)
		}
		if cfg.OnFileCreated != nil {
			cfg.OnFileCreated(filepath.Join("internal", f.rel))
		}
	}
	return nil
}
