package main

import (
	"fmt"
	"os"

	"github.com/samznd/layrgo/internal/cli"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "layrgo",
		Short: "Scaffold production-ready Go web APIs in seconds",
		Long: `Layrgo generates Go web projects with clean architecture,
wired handlers/services/repositories, Docker support, and your choice
of framework, database, and ORM.`,
	}

	root.SetHelpCommand(&cobra.Command{Hidden: true})
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		fmt.Printf("❌ %v\n\n", err)
		_ = cmd.Usage()
		os.Exit(1)
		return nil
	})

	root.AddCommand(cli.InitCmd, cli.VersionCmd)

	if err := root.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
