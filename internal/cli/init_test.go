package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samznd/layrgo/internal/templates"
)

func TestConfigFromFlags_RequiresFrameworkAndDatabase(t *testing.T) {
	resetInitFlags()
	t.Cleanup(resetInitFlags)

	_, err := configFromFlags("myapp")
	if err == nil || !strings.Contains(err.Error(), "--framework and --database") {
		t.Fatalf("expected required flags error, got %v", err)
	}
}

func TestConfigFromFlags_ORMConflict(t *testing.T) {
	resetInitFlags()
	t.Cleanup(resetInitFlags)

	flagFramework = "Gin"
	flagDatabase = "Postgres"
	flagORM = "GORM"
	flagNoORM = true

	_, err := configFromFlags("myapp")
	if err == nil || !strings.Contains(err.Error(), "not both") {
		t.Fatalf("expected mutual exclusion error, got %v", err)
	}
}

func TestConfigFromFlags_Valid(t *testing.T) {
	resetInitFlags()
	t.Cleanup(resetInitFlags)

	flagFramework = "Gin"
	flagDatabase = "Postgres"
	flagORM = "GORM"

	cfg, err := configFromFlags("myapi")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProjectName != "myapi" || cfg.Framework != "Gin" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}

func TestCanonicalChoice_Invalid(t *testing.T) {
	_, err := canonicalChoice("framework", "beego", []string{"Gin", "Echo"})
	if err == nil || !strings.Contains(err.Error(), "invalid framework") {
		t.Fatalf("expected invalid framework error, got %v", err)
	}
}

func TestCanonicalChoice_NormalizesCase(t *testing.T) {
	got, err := canonicalChoice("database", "sqlite", []string{"Postgres", "MySQL", "SQLite"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "SQLite" {
		t.Fatalf("expected canonical %q, got %q", "SQLite", got)
	}
}

func resetInitFlags() {
	flagFramework = ""
	flagDatabase = ""
	flagORM = ""
	flagNoORM = false
}

func TestValidateProjectName(t *testing.T) {
	valid := []string{"myapi", "my-api", "my_api", "api2", "my.api"}
	for _, name := range valid {
		if err := validateProjectName(name); err != nil {
			t.Errorf("validateProjectName(%q) = %v, want nil", name, err)
		}
	}

	invalid := []string{"", "   ", "my app", "../escaped", "a/b", `a\b`, "..", ".", "/abs", "wat?"}
	for _, name := range invalid {
		if err := validateProjectName(name); err == nil {
			t.Errorf("validateProjectName(%q) = nil, want error", name)
		}
	}
}

func TestConfigFromFlags_RejectsTraversal(t *testing.T) {
	resetInitFlags()
	t.Cleanup(resetInitFlags)

	flagFramework = "Gin"
	flagDatabase = "Postgres"

	if _, err := configFromFlags("../escaped"); err == nil {
		t.Fatal("expected error for name containing path separators")
	}
}

func TestConfigFromFlags_CanonicalizesCase(t *testing.T) {
	resetInitFlags()
	t.Cleanup(resetInitFlags)

	flagFramework = "gin"
	flagDatabase = "sqlite"
	flagORM = "gorm"

	cfg, err := configFromFlags("myapi")
	if err != nil {
		t.Fatal(err)
	}
	// Lowercase input should come out in the canonical spelling.
	if cfg.Framework != "Gin" || cfg.Database != "SQLite" || cfg.ORM != "GORM" {
		t.Fatalf("not canonicalized: %+v", cfg)
	}
}

func TestEnsureTargetDir(t *testing.T) {
	t.Run("missing dir is fine", func(t *testing.T) {
		if err := ensureTargetDir(filepath.Join(t.TempDir(), "nope"), false); err != nil {
			t.Fatalf("expected nil for missing dir, got %v", err)
		}
	})

	t.Run("empty dir is fine", func(t *testing.T) {
		if err := ensureTargetDir(t.TempDir(), false); err != nil {
			t.Fatalf("expected nil for empty dir, got %v", err)
		}
	})

	t.Run("non-empty dir is refused", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		err := ensureTargetDir(dir, false)
		if err == nil || !strings.Contains(err.Error(), "not empty") {
			t.Fatalf("expected not-empty error, got %v", err)
		}
	})

	t.Run("force overrides", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ensureTargetDir(dir, true); err != nil {
			t.Fatalf("expected --force to override, got %v", err)
		}
	})
}

// Every framework we offer must have templates behind it.
func TestFrameworks_MatchRenderableTemplates(t *testing.T) {
	t.Parallel()

	for _, fw := range Frameworks {
		fw := fw
		t.Run(fw, func(t *testing.T) {
			t.Parallel()

			if _, err := templates.MainFile("m", fw); err != nil {
				t.Errorf("Frameworks offers %q but MainFile fails: %v", fw, err)
			}
			if _, err := templates.RoutesFile("m", fw); err != nil {
				t.Errorf("Frameworks offers %q but RoutesFile fails: %v", fw, err)
			}
			if _, err := templates.HandlerFile("m", fw); err != nil {
				t.Errorf("Frameworks offers %q but HandlerFile fails: %v", fw, err)
			}
		})
	}
}

func TestFrameworkFlag_HelpListsEveryFramework(t *testing.T) {
	t.Parallel()

	flag := InitCmd.Flags().Lookup("framework")
	if flag == nil {
		t.Fatal("no --framework flag registered")
	}
	for _, fw := range Frameworks {
		if !strings.Contains(flag.Usage, fw) {
			t.Errorf("--framework help %q omits %q", flag.Usage, fw)
		}
	}
}
