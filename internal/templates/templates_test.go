package templates_test

import (
	"strings"
	"testing"

	"github.com/samznd/layrgo/internal/templates"
)

func TestRepositoryTemplate_RawSQL(t *testing.T) {
	t.Parallel()
	content := templates.RepositoryTemplate("none")
	if !strings.Contains(content, "func NewRepository(db *sql.DB)") {
		t.Error("expected sql.DB constructor")
	}
	if !strings.Contains(content, "GetMessage() (string, error)") {
		t.Error("expected consistent return signature")
	}
}

func TestRepositoryTemplate_XORM_ErrorPath(t *testing.T) {
	t.Parallel()
	content := templates.RepositoryTemplate("xorm")
	if strings.Contains(content, `return ""`) && !strings.Contains(content, `return "", err`) {
		t.Error("xorm error path should return error")
	}
}

func TestRoutesTemplate_FiberNoDoublePrefix(t *testing.T) {
	t.Parallel()
	content := templates.RoutesTemplate("myapp", "fiber")
	if strings.Contains(content, `Group("/api")`) {
		t.Error("fiber routes should not add extra /api prefix")
	}
}

func TestRoutesTemplate_EchoAcceptsGroup(t *testing.T) {
	t.Parallel()
	content := templates.RoutesTemplate("myapp", "echo")
	if !strings.Contains(content, "func SetupRoutes(api *echo.Group") {
		t.Error("echo routes should accept *echo.Group")
	}
}

func TestHandlerTemplate_AllFrameworks(t *testing.T) {
	t.Parallel()
	frameworks := []string{"fiber", "gin", "echo", "chi", "iris"}
	for _, fw := range frameworks {
		content := templates.HandlerTemplate("myapp", fw)
		if content == "" {
			t.Errorf("empty handler template for %s", fw)
		}
		if !strings.Contains(content, "NewHandler(s services.Service)") {
			t.Errorf("%s handler missing NewHandler constructor", fw)
		}
	}
}
