package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/samznd/layrgo/internal/feature"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	flagAddFields string
	flagAddPlural string
	flagAddDryRun bool
	flagAddForce  bool
	flagAddJSON   bool
	flagAddYes    bool
)

// AddCmd groups the "add" subcommands.
var AddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add code to an existing Layrgo project",
}

// AddFeatureCmd adds a complete feature to an existing project.
var AddFeatureCmd = &cobra.Command{
	Use:   "feature [name]",
	Short: "Add a feature: model, store, service, handler, routes and a test, wired into cmd/main.go",
	Long: `Add a complete feature to the Layrgo project in the current directory.

It writes internal/<name>/ (model, store, service, a service test, handler and
routes) for the project's framework, database and ORM, wires it into
cmd/main.go, then runs gofmt, go build, go vet and the feature's tests.

Field types: string, text, int, float, bool, time. Add ? for an optional
(nullable) field. Every feature gets an int64 id.

Examples:
  layrgo add feature order --fields "customer:string,total:int,paid:bool"
  layrgo add feature person --plural people --fields "name:string,bio:text?"
  layrgo add feature order --fields "customer:string" --dry-run
  layrgo add feature order --fields "customer:string" --json --yes

Exit codes:
  0  created, wired and all checks passed
  1  invalid input, or not a Layrgo project; nothing written
  2  internal/<name> already exists (without --force); nothing written
  3  files created, but cmd/main.go could not be wired (lines printed to paste)
  4  files created and wired, but a check failed`,
	Args:          cobra.MaximumNArgs(1),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		opts := feature.Options{
			Root:   ".",
			Plural: flagAddPlural,
			Fields: flagAddFields,
			DryRun: flagAddDryRun,
			Force:  flagAddForce,
		}
		if len(args) == 1 {
			opts.Name = args[0]
		}

		if err := promptMissing(&opts); err != nil {
			return reportAddError(out, &feature.Error{Code: feature.ExitInvalid, Err: err})
		}

		res, err := feature.Add(opts)
		if err != nil {
			return reportAddError(out, err)
		}
		if flagAddJSON {
			return writeJSON(out, res)
		}
		printAddResult(out, res)
		return nil
	},
}

func init() {
	f := AddFeatureCmd.Flags()
	f.StringVar(&flagAddFields, "fields", "", `comma-separated name:type list, e.g. "title:string,note:text?"`)
	f.StringVar(&flagAddPlural, "plural", "", "override the derived plural, e.g. people for person")
	f.BoolVar(&flagAddDryRun, "dry-run", false, "print the files and the cmd/main.go diff; write nothing")
	f.BoolVar(&flagAddForce, "force", false, "replace an existing internal/<name> package")
	f.BoolVar(&flagAddJSON, "json", false, "print the result as JSON")
	f.BoolVar(&flagAddYes, "yes", false, "never prompt; fail on anything missing instead")

	AddCmd.AddCommand(AddFeatureCmd)
}

// promptMissing asks for the name and fields when they are missing and a
// person is at the terminal. It never prompts otherwise: an agent can't answer
// a prompt, and a hung agent is the worst failure.
func promptMissing(opts *feature.Options) error {
	interactive := !flagAddYes && !flagAddJSON && stdinIsTerminal()

	if opts.Name == "" {
		if !interactive {
			return errors.New("feature name is required, e.g. layrgo add feature order --fields \"customer:string\"")
		}
		if err := survey.AskOne(&survey.Input{Message: "Feature name:"}, &opts.Name); err != nil {
			return err
		}
	}
	if opts.Fields == "" {
		if !interactive {
			return errors.New("--fields is required, e.g. --fields \"customer:string,total:int,paid:bool\"")
		}
		if err := survey.AskOne(&survey.Input{
			Message: "Fields (name:type, comma-separated):",
			Help:    "Types: string, text, int, float, bool, time. Add ? for optional, e.g. note:text?",
		}, &opts.Fields); err != nil {
			return err
		}
	}
	return nil
}

// stdinIsTerminal is false for pipes, files and /dev/null, which agents and
// CI use.
func stdinIsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// reportAddError prints err, and any partial result it carries, and returns
// an exitError so main exits with the matching code.
func reportAddError(out io.Writer, err error) error {
	var ferr *feature.Error
	if !errors.As(err, &ferr) {
		ferr = &feature.Error{Code: feature.ExitInvalid, Err: err}
	}

	if flagAddJSON {
		// A nil embedded *Result adds no fields, so input errors print just
		// the error and the code.
		_ = writeJSON(out, struct {
			*feature.Result
			Error    string `json:"error"`
			ExitCode int    `json:"exit_code"`
		}{ferr.Result, ferr.Error(), ferr.Code})
		return exitError{code: ferr.Code}
	}

	if ferr.Result != nil {
		printAddResult(out, ferr.Result)
	}
	fmt.Fprintf(out, "❌ %v\n", ferr)
	return exitError{code: ferr.Code}
}

func printAddResult(out io.Writer, res *feature.Result) {
	dir := "internal/" + res.Feature + "/"
	names := make([]string, len(res.Files))
	for i, f := range res.Files {
		names[i] = strings.TrimPrefix(f, dir)
	}

	if res.DryRun {
		fmt.Fprintf(out, "Dry run: nothing was written.\n\nWould create %s %s\n", dir, strings.Join(names, " "))
		switch {
		case !res.Wired:
			fmt.Fprintf(out, "\nWould not wire cmd/main.go.\n%s", res.ManualWiring)
		case res.Diff != "":
			fmt.Fprintf(out, "\nWould change cmd/main.go:\n\n%s", res.Diff)
		default:
			fmt.Fprintln(out, "\ncmd/main.go already wires it.")
		}
		return
	}

	fmt.Fprintf(out, "✓ Created %s %s\n", dir, strings.Join(names, " "))
	switch {
	case res.AlreadyWired:
		fmt.Fprintf(out, "✓ %s was already wired in cmd/main.go\n", res.Feature)
	case res.Wired:
		fmt.Fprintf(out, "✓ Wired %s in cmd/main.go (store, Migrate, service, handler, routes)\n", res.Feature)
	default:
		fmt.Fprintf(out, "✗ Could not wire cmd/main.go; nothing in it was changed.\n\n%s\n", res.ManualWiring)
	}
	if res.StackFileWritten {
		fmt.Fprintln(out, "✓ Recorded the stack in .layrgo.json (commit it)")
	}
	if res.Checks != nil {
		if len(res.FailedChecks) == 0 {
			fmt.Fprintln(out, "✓ gofmt, go build, go vet and go test passed")
		} else {
			for _, name := range []string{"gofmt", "build", "vet", "test"} {
				if output, failed := res.FailedChecks[name]; failed {
					fmt.Fprintf(out, "✗ %s failed:\n%s\n", name, output)
				}
			}
		}
	}

	fmt.Fprintf(out, "\nNext: add your business rules in %sservice.go (search for TODO)\n", dir)
	fmt.Fprintf(out, "Try:  %s\n", res.Example)
}

func writeJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// exitError ends the program with code. Its message was already printed.
type exitError struct{ code int }

func (e exitError) Error() string { return fmt.Sprintf("exit status %d", e.code) }

// ExitCode reports the exit code a command asked for. ok is false for an
// ordinary error, which main prints itself.
func ExitCode(err error) (code int, ok bool) {
	var e exitError
	if errors.As(err, &e) {
		return e.code, true
	}
	return 0, false
}
