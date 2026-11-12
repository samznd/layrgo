package templates

import (
	"fmt"
	"path/filepath"

	"github.com/samznd/layrgo/internal/fileutil"
)

// FeatureConfig holds the inputs for generating the sample feature package.
type FeatureConfig struct {
	ProjectPath string
	ProjectName string
	Framework   string
	ORM         string

	// OnFileCreated is called with each generated file's relative path, if set.
	OnFileCreated func(rel string)
}

// WriteFeature generates the sample feature package: its store, service,
// handler, and route files.
func WriteFeature(cfg FeatureConfig) error {
	store, err := StoreFile(cfg.ProjectName, cfg.ORM)
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
		{"store.go", store},
		{"service.go", service},
		{"handler.go", handler},
		{"routes.go", routes},
	}

	for _, f := range files {
		path := filepath.Join(cfg.ProjectPath, "internal", FeaturePackage, f.rel)
		if err := fileutil.CreateFile(path, f.content); err != nil {
			return fmt.Errorf("create %s: %w", f.rel, err)
		}
		if cfg.OnFileCreated != nil {
			cfg.OnFileCreated(filepath.Join("internal", FeaturePackage, f.rel))
		}
	}
	return nil
}
