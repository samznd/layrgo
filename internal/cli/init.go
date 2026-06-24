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
  layrgo init myapi --framework chi --database SQLite --no-orm`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := resolveConfig(args)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(cfg.ProjectPath, 0o755); err != nil {
			return fmt.Errorf("create project directory: %w", err)
		}

		if err := scaffold.Scaffold(cfg); err != nil {
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

	if cfg.Framework == "" || cfg.Database == "" {
		return cfg, fmt.Errorf("--framework and --database are required for non-interactive init")
	}

	if flagNoORM && flagORM != "" {
		return cfg, fmt.Errorf("use either --no-orm or --orm, not both")
	}

	if flagORM != "" {
		cfg.ORM = flagORM
	}

	if err := validateChoice("framework", cfg.Framework, []string{"Fiber", "Gin", "Echo", "Chi", "Iris"}); err != nil {
		return cfg, err
	}
	if err := validateChoice("database", cfg.Database, []string{"Postgres", "MySQL", "SQLite"}); err != nil {
		return cfg, err
	}
	if cfg.ORM != "none" {
		if err := validateChoice("orm", cfg.ORM, []string{"GORM", "XORM", "Ent"}); err != nil {
			return cfg, err
		}
	}

	return cfg, nil
}

func validateChoice(name, value string, allowed []string) error {
	for _, a := range allowed {
		if strings.EqualFold(value, a) {
			return nil
		}
	}
	return fmt.Errorf("invalid %s %q (allowed: %s)", name, value, strings.Join(allowed, ", "))
}

func promptConfig() (scaffold.Config, error) {
	var cfg scaffold.Config

	if err := survey.AskOne(&survey.Input{Message: "What is your project name?"}, &cfg.ProjectName); err != nil {
		return cfg, cancelErr(err)
	}

	if err := survey.AskOne(&survey.Select{
		Message: "Choose your web framework:",
		Options: []string{"Fiber", "Gin", "Echo", "Chi", "Iris"},
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

	cfg.ProjectPath = filepath.Join(".", cfg.ProjectName)
	return cfg, nil
}

func cancelErr(err error) error {
	fmt.Println("\nOperation canceled by user.")
	return err
}
