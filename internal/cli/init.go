package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/samznd/layrgo/internal/scaffold"
	"github.com/spf13/cobra"
)

var (
	flagFramework string
	flagDatabase  string
	flagORM       string
	flagNoORM     bool
	flagForce     bool
)

// InitCmd scaffolds a new Go web application.
var InitCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new Go web application",
	Long: `Initialize a new Go web application with clean architecture.

Run without flags for an interactive wizard, or pass a project name with
flags to scaffold non-interactively (great for scripts and demos).

Examples:
  layrgo init
  layrgo init myapi --framework gin --database Postgres --orm GORM
  layrgo init myapi --framework chi --database SQLite --no-orm

Scaffolding refuses to write into a non-empty directory; pass --force to override.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := resolveConfig(args)
		if err != nil {
			return err
		}

		if err := ensureTargetDir(cfg.ProjectPath, flagForce); err != nil {
			return err
		}

		// Only a directory we created ourselves may be cleaned up on failure;
		// never remove something that was already there.
		_, statErr := os.Stat(cfg.ProjectPath)
		weCreatedDir := os.IsNotExist(statErr)

		if err := os.MkdirAll(cfg.ProjectPath, 0o755); err != nil {
			return fmt.Errorf("create project directory: %w", err)
		}

		if err := scaffold.Scaffold(cfg); err != nil {
			if weCreatedDir {
				if rmErr := os.RemoveAll(cfg.ProjectPath); rmErr != nil {
					fmt.Printf("⚠️  could not clean up %s: %v\n", cfg.ProjectPath, rmErr)
				} else {
					fmt.Printf("🧹 Removed partially created %s\n", cfg.ProjectPath)
				}
			}
			return err
		}

		printSuccess(cfg)
		return nil
	},
}

func init() {
	InitCmd.Flags().StringVar(&flagFramework, "framework", "", "web framework: Fiber, Gin, Echo, Chi, or Iris")
	InitCmd.Flags().StringVar(&flagDatabase, "database", "", "database: Postgres, MySQL, or SQLite")
	InitCmd.Flags().StringVar(&flagORM, "orm", "", "ORM: GORM, XORM, or Ent")
	InitCmd.Flags().BoolVar(&flagNoORM, "no-orm", false, "use database/sql instead of an ORM")
	InitCmd.Flags().BoolVar(&flagForce, "force", false, "scaffold into a non-empty directory, overwriting existing files")
}

func resolveConfig(args []string) (scaffold.Config, error) {
	if len(args) == 1 {
		return configFromFlags(args[0])
	}
	if hasNonInteractiveFlags() {
		return scaffold.Config{}, fmt.Errorf("project name is required when using --framework, --database, or --orm flags")
	}
	return promptConfig()
}

func hasNonInteractiveFlags() bool {
	return flagFramework != "" || flagDatabase != "" || flagORM != "" || flagNoORM
}

func configFromFlags(name string) (scaffold.Config, error) {
	cfg := scaffold.Config{
		ProjectName: name,
		ProjectPath: filepath.Join(".", name),
		Framework:   flagFramework,
		Database:    flagDatabase,
		ORM:         "none",
	}

	if err := validateProjectName(name); err != nil {
		return cfg, err
	}

	if cfg.Framework == "" || cfg.Database == "" {
		return cfg, fmt.Errorf("--framework and --database are required for non-interactive init")
	}

	if flagNoORM && flagORM != "" {
		return cfg, fmt.Errorf("use either --no-orm or --orm, not both")
	}

	if flagORM != "" {
		cfg.ORM = flagORM
	}

	framework, err := canonicalChoice("framework", cfg.Framework, []string{"Fiber", "Gin", "Echo", "Chi", "Iris", "stdlib"})
	if err != nil {
		return cfg, err
	}
	cfg.Framework = framework

	database, err := canonicalChoice("database", cfg.Database, []string{"Postgres", "MySQL", "SQLite"})
	if err != nil {
		return cfg, err
	}
	cfg.Database = database

	if cfg.ORM != "none" {
		orm, err := canonicalChoice("orm", cfg.ORM, []string{"GORM", "XORM", "Ent"})
		if err != nil {
			return cfg, err
		}
		cfg.ORM = orm
	}

	return cfg, nil
}

// canonicalChoice validates value case-insensitively and returns the canonical
// spelling from allowed. Normalizing here, at the input boundary, means everything
// downstream can compare against one known form instead of re-lowering per call site.
func canonicalChoice(name, value string, allowed []string) (string, error) {
	for _, a := range allowed {
		if strings.EqualFold(value, a) {
			return a, nil
		}
	}
	return "", fmt.Errorf("invalid %s %q (allowed: %s)", name, value, strings.Join(allowed, ", "))
}

func promptConfig() (scaffold.Config, error) {
	var cfg scaffold.Config

	if err := survey.AskOne(&survey.Input{Message: "What is your project name?"}, &cfg.ProjectName); err != nil {
		return cfg, cancelErr(err)
	}

	if err := survey.AskOne(&survey.Select{
		Message: "Choose your web framework:",
		Options: []string{"Fiber", "Gin", "Echo", "Chi", "Iris", "stdlib"},
	}, &cfg.Framework); err != nil {
		return cfg, cancelErr(err)
	}

	if err := survey.AskOne(&survey.Select{
		Message: "Choose your database system:",
		Options: []string{"Postgres", "MySQL", "SQLite"},
	}, &cfg.Database); err != nil {
		return cfg, cancelErr(err)
	}

	cfg.ORM = "none"

	var useORM bool
	if err := survey.AskOne(&survey.Confirm{Message: "Would you like to use an ORM?"}, &useORM); err != nil {
		return cfg, cancelErr(err)
	}

	if useORM {
		if err := survey.AskOne(&survey.Select{
			Message: "Choose your ORM framework:",
			Options: []string{"GORM", "XORM", "Ent"},
		}, &cfg.ORM); err != nil {
			return cfg, cancelErr(err)
		}
	}

	if err := validateProjectName(cfg.ProjectName); err != nil {
		return cfg, err
	}

	cfg.ProjectPath = filepath.Join(".", cfg.ProjectName)
	return cfg, nil
}

func cancelErr(err error) error {
	fmt.Println("\nOperation canceled by user.")
	return err
}

// ensureTargetDir refuses to scaffold into a directory that already has contents,
// so a mistyped name can never clobber an existing project. A missing directory is
// fine (we create it next); force skips the check.
func ensureTargetDir(path string, force bool) error {
	if force {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspect project directory: %w", err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("directory %s is not empty (use --force to scaffold into it anyway)", path)
	}
	return nil
}

// validateProjectName rejects names that are not usable as a Go module path or that
// would place the project outside the current directory. Checked before any
// filesystem work so a bad name cannot leave a half-built project behind.
func validateProjectName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("project name is required")
	}
	if name != filepath.Clean(name) || filepath.IsAbs(name) {
		return fmt.Errorf("invalid project name %q: must be a plain directory name", name)
	}
	if strings.ContainsAny(name, `/\`) || name == ".." || name == "." {
		return fmt.Errorf("invalid project name %q: must not contain path separators", name)
	}
	if strings.ContainsAny(name, " \t\n\"'`!$&*()[]{}<>|;:?") {
		return fmt.Errorf("invalid project name %q: must be a valid Go module path (letters, digits, '.', '-', '_')", name)
	}
	return nil
}
