package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectStack(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		gomod string
		want  Stack
	}{
		{"require (\n\tgithub.com/gin-gonic/gin v1.10.1\n\tgithub.com/lib/pq v1.10.9\n)", NewStack("gin", "postgres", "none")},
		{"require github.com/go-chi/chi/v5 v5.2.1\nrequire github.com/mattn/go-sqlite3 v1.14.24", NewStack("chi", "sqlite", "none")},
		{"require (\n\tgithub.com/gofiber/fiber/v3 v3.0.0-beta.4\n\tgorm.io/driver/mysql v1.5.7\n\tgorm.io/gorm v1.25.12 // indirect\n)", NewStack("fiber", "mysql", "gorm")},
		{"require (\n\tgithub.com/go-sql-driver/mysql v1.8.1\n\txorm.io/xorm v1.3.9\n)", NewStack("stdlib", "mysql", "xorm")},
		{"require (\n\tgithub.com/kataras/iris/v12 v12.2.11\n\tgithub.com/lib/pq v1.10.9\n\tentgo.io/ent v0.14.1\n)", NewStack("iris", "postgres", "ent")},
	} {
		got, err := DetectStack([]byte("module shop\n\ngo 1.23\n\n" + tc.gomod + "\n"))
		if err != nil {
			t.Errorf("DetectStack(%q): %v", tc.gomod, err)
			continue
		}
		if got != tc.want {
			t.Errorf("DetectStack(%q) = %+v, want %+v", tc.gomod, got, tc.want)
		}
	}
}

func TestDetectStack_NoDatabase(t *testing.T) {
	t.Parallel()

	_, err := DetectStack([]byte("module shop\n\nrequire github.com/gin-gonic/gin v1.10.1\n"))
	if err == nil || !strings.Contains(err.Error(), StackFile) {
		t.Fatalf("err = %v, want it to point to %s", err, StackFile)
	}
}

func TestLoadStack(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	write := func(content string) {
		if err := os.WriteFile(filepath.Join(root, StackFile), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(NewStack("Gin", "Postgres", "GORM").Marshal())
	got, detected, err := LoadStack(root, nil)
	if err != nil || detected || got != NewStack("gin", "postgres", "gorm") {
		t.Errorf("LoadStack = %+v, %v, %v", got, detected, err)
	}

	write(`{"version":1,"framework":"rails","database":"postgres"}`)
	if _, _, err := LoadStack(root, nil); err == nil || !strings.Contains(err.Error(), "rails") {
		t.Errorf("err = %v, want unsupported framework", err)
	}
}

func TestModulePath(t *testing.T) {
	t.Parallel()

	if got := ModulePath([]byte("// comment\nmodule github.com/acme/shop // main module\n\ngo 1.23\n")); got != "github.com/acme/shop" {
		t.Errorf("ModulePath = %q", got)
	}
}
