package scaffold

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const commandTimeout = 5 * time.Minute

func installDependencies(cfg Config) error {
	logStep("📦 Initializing Go module...")
	if err := runGo(cfg.ProjectPath, "mod", "init", cfg.ProjectName); err != nil {
		return err
	}

	logStep("📦 Installing dependencies...")

	pkgs := []string{
		"github.com/joho/godotenv@v1.5.1",
	}

	switch strings.ToLower(cfg.Framework) {
	case "fiber":
		pkgs = append(pkgs, "github.com/gofiber/fiber/v3@v3.0.0-beta.4")
	case "gin":
		pkgs = append(pkgs, "github.com/gin-gonic/gin@v1.10.1")
	case "echo":
		pkgs = append(pkgs, "github.com/labstack/echo/v4@v4.13.3")
	case "chi":
		pkgs = append(pkgs, "github.com/go-chi/chi/v5@v5.2.1")
	case "iris":
		pkgs = append(pkgs, "github.com/kataras/iris/v12@v12.2.11")
	case "stdlib":
		// net/http needs no router.
	default:
		return fmt.Errorf("invalid framework: %s", cfg.Framework)
	}

	switch strings.ToLower(cfg.Database) {
	case "postgres":
		pkgs = append(pkgs, "github.com/lib/pq@v1.10.9")
	case "mysql":
		pkgs = append(pkgs, "github.com/go-sql-driver/mysql@v1.8.1")
	case "sqlite":
		pkgs = append(pkgs, "github.com/mattn/go-sqlite3@v1.14.24")
	default:
		return fmt.Errorf("invalid database: %s", cfg.Database)
	}

	switch strings.ToLower(cfg.ORM) {
	case "gorm":
		pkgs = append(pkgs, "gorm.io/gorm@v1.25.12")
		switch strings.ToLower(cfg.Database) {
		case "postgres":
			pkgs = append(pkgs, "gorm.io/driver/postgres@v1.5.11")
		case "mysql":
			pkgs = append(pkgs, "gorm.io/driver/mysql@v1.5.7")
		case "sqlite":
			pkgs = append(pkgs, "gorm.io/driver/sqlite@v1.5.7")
		}
	case "xorm":
		pkgs = append(pkgs, "xorm.io/xorm@v1.3.9")
	case "ent":
		pkgs = append(pkgs, "entgo.io/ent@v0.14.1", "entgo.io/ent/cmd/ent@v0.14.1")
	}

	// Install the pinned versions. "go get" records them in go.mod.
	if err := runGo(cfg.ProjectPath, append([]string{"get"}, pkgs...)...); err != nil {
		return err
	}

	// tidy moves the modules we import out of "// indirect". It is allowed to
	// fail: it re-resolves everything to the newest release and can pick a module
	// that needs a newer Go than the user has (gorm.io/driver/postgres v1.6.3
	// wants Go 1.25). We pin again right after, so a failed tidy only leaves
	// "// indirect" markers behind.
	if err := runGo(cfg.ProjectPath, "mod", "tidy"); err != nil {
		logStep("ℹ️  skipped go mod tidy (%v); pinned versions are already recorded", err)
	}

	// Pin again in case tidy upgraded something.
	if err := runGo(cfg.ProjectPath, append([]string{"get"}, pkgs...)...); err != nil {
		return err
	}

	if err := runGo(cfg.ProjectPath, "mod", "download"); err != nil {
		return err
	}

	logStep("✅ Dependencies installed successfully!")
	return nil
}

func runGo(dir string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Stdout = progressOut
	cmd.Stderr = progressOut

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go %s: %w", strings.Join(args, " "), err)
	}
	return nil
}
