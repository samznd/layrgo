package feature

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/samznd/layrgo/internal/fileutil"
	"github.com/samznd/layrgo/internal/project"
	"github.com/samznd/layrgo/internal/templates"
)

// Exit codes of "layrgo add feature". 0 means created, wired and checked.
const (
	ExitInvalid    = 1 // invalid input, or not a Layrgo project; nothing written
	ExitExists     = 2 // internal/<name> exists and --force was not given; nothing written
	ExitNotWired   = 3 // files written, cmd/main.go left alone; Result.ManualWiring says what to paste
	ExitCheckFails = 4 // files written and wired, but a check failed
)

// Error is a failure with the exit code the command should end with. Result
// is set when files were written, so the caller can still report them.
type Error struct {
	Code   int
	Err    error
	Result *Result
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

// Options are the inputs of Add.
type Options struct {
	Root   string // the project root, the directory with go.mod
	Name   string
	Plural string // overrides the derived plural when set
	Fields string // the --fields value
	DryRun bool
	Force  bool

	// SkipChecks skips gofmt, build, vet and test. Tests use it.
	SkipChecks bool
}

// Result is what Add did. Its JSON form is the --json output.
type Result struct {
	Feature          string        `json:"feature"`
	Files            []string      `json:"files"`
	Wired            bool          `json:"wired"`
	AlreadyWired     bool          `json:"already_wired,omitempty"`
	Routes           []string      `json:"routes"`
	Checks           *CheckSummary `json:"checks,omitempty"`
	Todos            []string      `json:"todos"`
	Example          string        `json:"example"`
	DryRun           bool          `json:"dry_run,omitempty"`
	Diff             string        `json:"diff,omitempty"`
	ManualWiring     string        `json:"manual_wiring,omitempty"`
	StackFileWritten bool          `json:"stack_file_written,omitempty"`

	// FailedChecks holds the output of each failed check, by name.
	FailedChecks map[string]string `json:"failed_checks,omitempty"`
}

// CheckSummary is whether each check passed.
type CheckSummary struct {
	Gofmt bool `json:"gofmt"`
	Build bool `json:"build"`
	Vet   bool `json:"vet"`
	Test  bool `json:"test"`
}

const mainFile = "cmd/main.go"

// Add generates a feature in an existing project, wires it into cmd/main.go
// and runs the checks. A non-nil error is always an *Error.
func Add(opts Options) (*Result, error) {
	plan, err := newPlan(opts)
	if err != nil {
		return nil, err
	}
	res := plan.result()
	if opts.DryRun {
		return res, nil
	}

	if err := plan.write(); err != nil {
		return nil, &Error{Code: ExitInvalid, Err: err}
	}
	res.StackFileWritten = plan.writeStack

	if !opts.SkipChecks {
		res.Checks, res.FailedChecks = summarize(RunChecks(opts.Root, plan.names.Package))
	}

	switch {
	case plan.wireErr != nil:
		return res, &Error{Code: ExitNotWired, Err: plan.wireErr, Result: res}
	case len(res.FailedChecks) > 0:
		return res, &Error{Code: ExitCheckFails, Err: errors.New("the feature was added, but a check failed"), Result: res}
	}
	return res, nil
}

// plan is everything Add will write, worked out before anything is written.
type plan struct {
	opts       Options
	names      Names
	fields     []templates.Field
	stack      project.Stack
	writeStack bool // .layrgo.json was missing; write the detected stack
	files      []templates.GeneratedFile
	mainBefore []byte
	wire       WireResult
	wireErr    error // ErrCannotWire: write the files, leave cmd/main.go alone
}

func newPlan(opts Options) (*plan, error) {
	invalid := func(err error) error { return &Error{Code: ExitInvalid, Err: err} }

	gomod, err := os.ReadFile(filepath.Join(opts.Root, "go.mod"))
	if err != nil {
		return nil, invalid(fmt.Errorf("no go.mod here; run this from the root of a Layrgo project"))
	}
	module := project.ModulePath(gomod)
	if module == "" {
		return nil, invalid(fmt.Errorf("go.mod declares no module"))
	}
	mainBefore, err := os.ReadFile(filepath.Join(opts.Root, mainFile))
	if err != nil {
		return nil, invalid(fmt.Errorf("no %s here; run this from the root of a Layrgo project", mainFile))
	}

	p := &plan{opts: opts, mainBefore: mainBefore}
	if p.stack, p.writeStack, err = project.LoadStack(opts.Root, gomod); err != nil {
		return nil, invalid(err)
	}
	if p.names, err = NewNames(opts.Name, opts.Plural); err != nil {
		return nil, invalid(err)
	}
	if p.fields, err = ParseFields(opts.Fields); err != nil {
		return nil, invalid(err)
	}

	dir := filepath.Join(opts.Root, "internal", p.names.Package)
	if _, err := os.Stat(dir); err == nil && !opts.Force {
		return nil, &Error{Code: ExitExists, Err: fmt.Errorf("internal/%s already exists (use --force to replace it)", p.names.Package)}
	}

	p.files, err = templates.FeatureFiles(templates.FeatureSpec{
		Module:    module,
		Package:   p.names.Package,
		Type:      p.names.Type,
		Plural:    p.names.Plural,
		Framework: p.stack.Framework,
		Database:  p.stack.Database,
		ORM:       p.stack.ORM,
		Fields:    p.fields,
	})
	if err != nil {
		return nil, invalid(err)
	}

	p.wire, p.wireErr = Wire(mainBefore, module, p.stack.Framework, p.names)
	return p, nil
}

func (p *plan) result() *Result {
	res := &Result{
		Feature:      p.names.Package,
		Wired:        p.wireErr == nil,
		AlreadyWired: p.wire.AlreadyWired,
		DryRun:       p.opts.DryRun,
		Routes: []string{
			"POST /api/v1/" + p.names.Plural,
			"GET /api/v1/" + p.names.Plural,
			"GET /api/v1/" + p.names.Plural + "/{id}",
		},
		Example: fmt.Sprintf("curl -X POST localhost:3000/api/v1/%s -H 'Content-Type: application/json' -d '%s'",
			p.names.Plural, templates.ExampleJSON(p.fields)),
	}
	for _, f := range p.files {
		res.Files = append(res.Files, f.Rel)
		res.Todos = append(res.Todos, todos(f)...)
	}
	if p.wireErr != nil {
		res.ManualWiring = p.wire.Wiring.Manual()
	}
	if p.opts.DryRun {
		res.Diff = unifiedDiff(mainFile, string(p.mainBefore), string(p.wire.Content))
	}
	return res
}

// write replaces internal/<name> with the generated files, then updates
// cmd/main.go and .layrgo.json.
func (p *plan) write() error {
	dir := filepath.Join(p.opts.Root, "internal", p.names.Package)
	if p.opts.Force {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("remove internal/%s: %w", p.names.Package, err)
		}
	}
	for _, f := range p.files {
		if err := fileutil.CreateFile(filepath.Join(p.opts.Root, filepath.FromSlash(f.Rel)), f.Content); err != nil {
			return err
		}
	}
	if p.wireErr == nil && !p.wire.AlreadyWired {
		if err := os.WriteFile(filepath.Join(p.opts.Root, mainFile), p.wire.Content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", mainFile, err)
		}
	}
	if p.writeStack {
		if err := fileutil.CreateFile(filepath.Join(p.opts.Root, project.StackFile), p.stack.Marshal()); err != nil {
			return err
		}
	}
	return nil
}

// todos lists "file:line" for each TODO comment in a generated file.
func todos(f templates.GeneratedFile) []string {
	var out []string
	sc := bufio.NewScanner(strings.NewReader(f.Content))
	for n := 1; sc.Scan(); n++ {
		if strings.HasPrefix(strings.TrimSpace(sc.Text()), "// TODO") {
			out = append(out, fmt.Sprintf("%s:%d", f.Rel, n))
		}
	}
	return out
}

func summarize(checks []Check) (*CheckSummary, map[string]string) {
	sum := &CheckSummary{}
	failed := map[string]string{}
	for _, c := range checks {
		switch c.Name {
		case "gofmt":
			sum.Gofmt = c.OK
		case "build":
			sum.Build = c.OK
		case "vet":
			sum.Vet = c.OK
		case "test":
			sum.Test = c.OK
		}
		if !c.OK {
			failed[c.Name] = c.Output
		}
	}
	if len(failed) == 0 {
		failed = nil
	}
	return sum, failed
}
