package cli

import (
	"strings"
	"testing"
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

func TestValidateChoice_Invalid(t *testing.T) {
	err := validateChoice("framework", "beego", []string{"Gin", "Echo"})
	if err == nil || !strings.Contains(err.Error(), "invalid framework") {
		t.Fatalf("expected invalid framework error, got %v", err)
	}
}

func resetInitFlags() {
	flagFramework = ""
	flagDatabase = ""
	flagORM = ""
	flagNoORM = false
}
