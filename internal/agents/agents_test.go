package agents_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samznd/layrgo/internal/agents"
	"github.com/samznd/layrgo/internal/project"
	"github.com/samznd/layrgo/internal/scaffold"
)

func TestMain(m *testing.M) {
	restore := scaffold.SetProgressOutput(io.Discard)
	code := m.Run()
	restore()
	os.Exit(code)
}

// newProject writes a current Layrgo project, plus the go.mod "init" would
// leave, without installing anything.
func newProject(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "shop")
	if err := scaffold.WriteFiles(scaffold.Config{
		ProjectName: "shop", ProjectPath: root,
		Framework: "Gin", Database: "Postgres", ORM: "none",
	}); err != nil {
		t.Fatal(err)
	}
	gomod := "module shop\n\ngo 1.23\n\nrequire (\n\tgithub.com/gin-gonic/gin v1.10.1\n\tgithub.com/lib/pq v1.10.9\n)\n"
	write(t, filepath.Join(root, "go.mod"), gomod)
	return root
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func fileResult(res *agents.Result, path string) agents.File {
	for _, f := range res.Files {
		if f.Path == path {
			return f
		}
	}
	return agents.File{}
}

func TestUpdate_UpToDateProjectIsUnchanged(t *testing.T) {
	t.Parallel()
	root := newProject(t)

	res, err := agents.Update(agents.Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if res.Changed() {
		t.Errorf("a fresh project should need no changes: %+v", res.Files)
	}
	if len(res.Files) != 4 {
		t.Errorf("got %d files, want 4", len(res.Files))
	}
}

func TestUpdate_BacksUpAnEditedFile(t *testing.T) {
	t.Parallel()
	root := newProject(t)
	agentsPath := filepath.Join(root, "AGENTS.md")
	current := read(t, agentsPath)

	old := "# AGENTS.md\n\nOld instructions, plus a note the team added by hand.\n"
	write(t, agentsPath, old)

	res, err := agents.Update(agents.Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	got := fileResult(res, "AGENTS.md")
	if got.Status != agents.Updated || got.Backup != "AGENTS.md.bak" {
		t.Fatalf("AGENTS.md result = %+v", got)
	}
	if read(t, agentsPath) != current {
		t.Error("AGENTS.md was not rewritten")
	}
	if read(t, agentsPath+".bak") != old {
		t.Error("the backup does not hold the old content")
	}
	if fileResult(res, "CLAUDE.md").Status != agents.Unchanged {
		t.Error("CLAUDE.md was touched")
	}

	// A second edit gets a second backup; the first is kept.
	write(t, agentsPath, "edited again\n")
	res, err = agents.Update(agents.Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if b := fileResult(res, "AGENTS.md").Backup; b != "AGENTS.md.bak.2" {
		t.Errorf("second backup = %q, want AGENTS.md.bak.2", b)
	}
	if read(t, agentsPath+".bak") != old {
		t.Error("the first backup was overwritten")
	}
}

func TestUpdate_CreatesMissingFiles(t *testing.T) {
	t.Parallel()
	root := newProject(t)
	if err := os.RemoveAll(filepath.Join(root, ".github")); err != nil {
		t.Fatal(err)
	}

	res, err := agents.Update(agents.Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if got := fileResult(res, ".github/copilot-instructions.md"); got.Status != agents.Created || got.Backup != "" {
		t.Errorf("copilot file result = %+v", got)
	}
	if !strings.Contains(read(t, filepath.Join(root, ".github", "copilot-instructions.md")), "AGENTS.md") {
		t.Error("copilot file not written")
	}
}

func TestUpdate_DryRunWritesNothing(t *testing.T) {
	t.Parallel()
	root := newProject(t)
	agentsPath := filepath.Join(root, "AGENTS.md")
	write(t, agentsPath, "old\n")
	if err := os.Remove(filepath.Join(root, project.StackFile)); err != nil {
		t.Fatal(err)
	}

	res, err := agents.Update(agents.Options{Root: root, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	got := fileResult(res, "AGENTS.md")
	if got.Status != agents.Updated || !strings.Contains(got.Diff, "-old") || !strings.Contains(got.Diff, "+# AGENTS.md") {
		t.Errorf("dry-run result = %+v", got)
	}
	if read(t, agentsPath) != "old\n" {
		t.Error("dry run changed AGENTS.md")
	}
	if _, err := os.Stat(agentsPath + ".bak"); !os.IsNotExist(err) {
		t.Error("dry run wrote a backup")
	}
	if _, err := os.Stat(filepath.Join(root, project.StackFile)); !os.IsNotExist(err) {
		t.Error("dry run wrote .layrgo.json")
	}
}

// Projects made before v1.5.0 have no .layrgo.json: the stack comes from
// go.mod and is recorded.
func TestUpdate_DetectsStackForOldProjects(t *testing.T) {
	t.Parallel()
	root := newProject(t)
	if err := os.Remove(filepath.Join(root, project.StackFile)); err != nil {
		t.Fatal(err)
	}

	res, err := agents.Update(agents.Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if !res.StackFileWritten {
		t.Error("the detected stack was not recorded")
	}
	if !strings.Contains(read(t, filepath.Join(root, "AGENTS.md")), "HTTP: Gin") {
		t.Error("AGENTS.md was not written for the detected framework")
	}
}

func TestUpdate_NotAProject(t *testing.T) {
	t.Parallel()

	if _, err := agents.Update(agents.Options{Root: t.TempDir()}); err == nil || !strings.Contains(err.Error(), "go.mod") {
		t.Fatalf("err = %v, want a go.mod error", err)
	}
}
