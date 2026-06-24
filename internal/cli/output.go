package cli

import (
	"fmt"
	"strings"

	"github.com/samznd/layrgo/internal/scaffold"
)

func printSuccess(cfg scaffold.Config) {
	orm := cfg.ORM
	if strings.EqualFold(orm, "none") {
		orm = "database/sql"
	}

	fmt.Println()
	fmt.Printf("✅ Project %q is ready (%s + %s + %s)\n\n", cfg.ProjectName, cfg.Framework, cfg.Database, orm)
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", cfg.ProjectName)
	fmt.Println("  go run ./cmd")
	fmt.Println("  curl http://localhost:3000/api/v1/message")
	fmt.Println()
	fmt.Println("Docker:")
	fmt.Println("  docker compose up --build")
	fmt.Println()
}
