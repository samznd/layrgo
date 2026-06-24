package scaffold_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samznd/layrgo/internal/scaffold"
)

func TestScaffold_GinPostgresNoORM(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	cfg := scaffold.Config{
		ProjectName: "testapp",
		ProjectPath: filepath.Join(dir, "testapp"),
		Framework:   "Gin",
		Database:    "Postgres",
		ORM:         "none",
	}

	if err := scaffold.Scaffold(cfg); err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}

	mainPath := filepath.Join(cfg.ProjectPath, "cmd", "main.go")
	content, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(content), "handlers.NewHandler") {
		t.Error("main.go missing dependency injection wiring")
	}

	cmd := exec.Command("go", "build", "-o", filepath.Join(dir, "app"), "./cmd")
	cmd.Dir = cfg.ProjectPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
}
