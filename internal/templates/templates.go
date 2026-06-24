package templates

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/samznd/layrgo/internal/fileutil"
)

// LayerConfig holds inputs for internal layer generation.
type LayerConfig struct {
	ProjectPath string
	ProjectName string
	Framework   string
	ORM         string
}

// WriteLayers generates repository, service, handler, and route files.
func WriteLayers(cfg LayerConfig) error {
	orm := strings.ToLower(cfg.ORM)
	framework := strings.ToLower(cfg.Framework)

	files := map[string]string{
		"repositories/repository.go": RepositoryTemplate(orm),
		"services/service.go":        ServiceTemplate(cfg.ProjectName),
		"handlers/handler.go":        HandlerTemplate(cfg.ProjectName, framework),
		"routes/routes.go":           RoutesTemplate(cfg.ProjectName, framework),
	}

	for rel, content := range files {
		path := filepath.Join(cfg.ProjectPath, "internal", rel)
		if err := fileutil.CreateFile(path, content); err != nil {
			return fmt.Errorf("create %s: %w", rel, err)
		}
	}
	return nil
}
