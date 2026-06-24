package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Set at build time via -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the layrgo version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("layrgo %s (commit: %s, built: %s)\n", Version, Commit, Date)
	},
}
