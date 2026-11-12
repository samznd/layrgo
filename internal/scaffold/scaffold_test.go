package scaffold_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/samznd/layrgo/internal/scaffold"
)

// TestMain silences per-file progress output so the 60-combination table does
// not flood the test log. This is what SetProgressOutput exists for.
func TestMain(m *testing.M) {
	restore := scaffold.SetProgressOutput(io.Discard)
	code := m.Run()
	restore()
	os.Exit(code)
}

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
	if !strings.Contains(string(content), "message.NewHandler") {
		t.Error("main.go missing dependency injection wiring")
	}

	cmd := exec.Command("go", "build", "-o", filepath.Join(dir, "app"), "./cmd")
	cmd.Dir = cfg.ProjectPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
}

// TestScaffold_ComposeMatchesDatabase covers every database in both the canonical
// and lowercase spelling. The lowercase cases are the regression: docker-compose
// used to fall through to Postgres whenever the caller passed "sqlite"/"mysql".
func TestScaffold_ComposeMatchesDatabase(t *testing.T) {
	t.Parallel()

	cases := []struct {
		database  string
		wantImage string
	}{
		{"Postgres", "postgres:latest"},
		{"postgres", "postgres:latest"},
		{"MySQL", "mysql:latest"},
		{"mysql", "mysql:latest"},
		{"SQLite", "nouchka/sqlite3"},
		{"sqlite", "nouchka/sqlite3"},
	}

	for _, tc := range cases {
		t.Run(tc.database, func(t *testing.T) {
			t.Parallel()

			cfg := scaffold.Config{
				ProjectName: "composeapp",
				ProjectPath: filepath.Join(t.TempDir(), "composeapp"),
				Framework:   "Chi",
				Database:    tc.database,
				ORM:         "none",
			}

			if err := scaffold.WriteFiles(cfg); err != nil {
				t.Fatalf("WriteFiles() error = %v", err)
			}

			compose, err := os.ReadFile(filepath.Join(cfg.ProjectPath, "docker-compose.yml"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(compose), tc.wantImage) {
				t.Errorf("database %q: compose missing image %q\n%s", tc.database, tc.wantImage, compose)
			}
		})
	}
}

// TestScaffold_AllStacksWriteFiles exercises every framework x database x ORM
// combination at the file-generation level, so an unhandled combination surfaces
// as an empty template rather than at a user's first build.
func TestScaffold_AllStacksWriteFiles(t *testing.T) {
	t.Parallel()

	frameworks := []string{"Fiber", "Gin", "Echo", "Chi", "Iris", "stdlib"}
	databases := []string{"Postgres", "MySQL", "SQLite"}
	orms := []string{"none", "GORM", "XORM", "Ent"}

	for _, fw := range frameworks {
		for _, db := range databases {
			for _, orm := range orms {
				name := fw + "_" + db + "_" + orm
				t.Run(name, func(t *testing.T) {
					t.Parallel()

					cfg := scaffold.Config{
						ProjectName: "stackapp",
						ProjectPath: filepath.Join(t.TempDir(), "stackapp"),
						Framework:   fw,
						Database:    db,
						ORM:         orm,
					}

					if err := scaffold.WriteFiles(cfg); err != nil {
						t.Fatalf("WriteFiles() error = %v", err)
					}

					for _, rel := range []string{
						filepath.Join("cmd", "main.go"),
						filepath.Join("config", "database.go"),
						filepath.Join("internal", "message", "handler.go"),
						filepath.Join("internal", "message", "service.go"),
						filepath.Join("internal", "message", "store.go"),
						filepath.Join("internal", "message", "routes.go"),
						".gitignore",
					} {
						path := filepath.Join(cfg.ProjectPath, rel)
						info, err := os.Stat(path)
						if err != nil {
							t.Fatalf("missing %s: %v", rel, err)
						}
						if info.Size() == 0 {
							t.Errorf("%s is empty", rel)
						}
					}

					// The by-layer packages must be gone, not just unused.
					for _, rel := range []string{"handlers", "services", "repositories", "routes", "models"} {
						if _, err := os.Stat(filepath.Join(cfg.ProjectPath, "internal", rel)); err == nil {
							t.Errorf("internal/%s should not be generated", rel)
						}
					}
				})
			}
		}
	}
}

// TestScaffold_RunsWithoutDotEnv is the regression for the Docker path: the image
// ships only the binary, so a generated app must start from real environment
// variables with no .env on disk.
func TestScaffold_RunsWithoutDotEnv(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	cfg := scaffold.Config{
		ProjectName: "envapp",
		ProjectPath: filepath.Join(dir, "envapp"),
		Framework:   "Chi",
		Database:    "SQLite",
		ORM:         "none",
	}

	if err := scaffold.Scaffold(cfg); err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}

	bin := filepath.Join(dir, "envapp-bin")
	build := exec.Command("go", "build", "-o", bin, "./cmd")
	build.Dir = cfg.ProjectPath
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}

	// Run from a directory with no .env, config supplied via the environment.
	runDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	run := exec.CommandContext(ctx, bin)
	run.Dir = runDir
	run.Env = append(os.Environ(), "DB_NAME="+filepath.Join(runDir, "test.db"))

	var buf bytes.Buffer
	run.Stdout = &buf
	run.Stderr = &buf

	if err := run.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() {
		_ = run.Process.Kill()
		_, _ = run.Process.Wait()
	})

	// Poll the endpoint until the server is up.
	var lastErr error
	for i := 0; i < 40; i++ {
		time.Sleep(250 * time.Millisecond)
		resp, err := http.Get("http://localhost:3000/api/v1/message")
		if err != nil {
			lastErr = err
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
		}
		if !strings.Contains(string(body), "data from the store") {
			t.Fatalf("unexpected body: %s", body)
		}
		return // served correctly with no .env present
	}

	t.Fatalf("server never became ready (last error: %v)\nprocess output:\n%s", lastErr, buf.String())
}
