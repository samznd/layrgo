package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/samznd/layrgo/internal/agents"
	"github.com/spf13/cobra"
)

var (
	flagUpdateDryRun bool
	flagUpdateJSON   bool
)

// UpdateCmd groups the "update" subcommands.
var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Bring files in an existing Layrgo project up to date",
}

// UpdateAgentsCmd rewrites the instruction files for AI coding agents.
var UpdateAgentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Rewrite AGENTS.md and the files that point agents to it, for this Layrgo version",
	Long: `Rewrite the instructions for AI coding agents in the Layrgo project in the
current directory: AGENTS.md, CLAUDE.md, GEMINI.md and
.github/copilot-instructions.md, written for the project's framework, database
and ORM.

Run it after upgrading Layrgo, so agents learn about new commands such as
"layrgo add feature". Each file it changes is backed up first (AGENTS.md.bak);
copy anything you added by hand from the backup, then delete it.

Examples:
  layrgo update agents --dry-run   # show the diff, write nothing
  layrgo update agents
  layrgo update agents --json`,
	Args:          cobra.NoArgs,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		res, err := agents.Update(agents.Options{Root: ".", DryRun: flagUpdateDryRun})
		if err != nil {
			if flagUpdateJSON {
				_ = writeJSON(out, map[string]any{"error": err.Error(), "exit_code": 1})
			} else {
				fmt.Fprintf(out, "❌ %v\n", err)
			}
			return exitError{code: 1}
		}

		if flagUpdateJSON {
			return writeJSON(out, res)
		}
		printUpdateResult(out, res)
		return nil
	},
}

func init() {
	f := UpdateAgentsCmd.Flags()
	f.BoolVar(&flagUpdateDryRun, "dry-run", false, "print what would change, with diffs; write nothing")
	f.BoolVar(&flagUpdateJSON, "json", false, "print the result as JSON")

	UpdateCmd.AddCommand(UpdateAgentsCmd)
}

func printUpdateResult(out io.Writer, res *agents.Result) {
	if !res.Changed() {
		fmt.Fprintln(out, "✓ The agent files are already up to date.")
		return
	}

	if res.DryRun {
		fmt.Fprintln(out, "Dry run: nothing was written.")
		for _, f := range res.Files {
			switch f.Status {
			case agents.Created:
				fmt.Fprintf(out, "\nWould create %s\n", f.Path)
			case agents.Updated:
				fmt.Fprintf(out, "\nWould update %s (backing up the current one):\n\n%s", f.Path, f.Diff)
			}
		}
		return
	}

	var backups []string
	for _, f := range res.Files {
		switch f.Status {
		case agents.Created:
			fmt.Fprintf(out, "✓ Created %s\n", f.Path)
		case agents.Updated:
			fmt.Fprintf(out, "✓ Updated %s (the old one is in %s)\n", f.Path, f.Backup)
			backups = append(backups, f.Backup)
		case agents.Unchanged:
			fmt.Fprintf(out, "✓ %s is up to date\n", f.Path)
		}
	}
	if res.StackFileWritten {
		fmt.Fprintln(out, "✓ Recorded the stack in .layrgo.json (commit it)")
	}
	if len(backups) > 0 {
		fmt.Fprintf(out, "\nNext: copy anything you added by hand from %s, then delete the backup.\n", strings.Join(backups, ", "))
	}
}
