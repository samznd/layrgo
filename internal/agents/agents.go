// Package agents rewrites the instruction files for AI coding agents
// (AGENTS.md, CLAUDE.md, GEMINI.md and .github/copilot-instructions.md) in an
// existing project, so projects made by an older Layrgo get the current
// instructions. It never discards a file's old content: a changed file is
// backed up next to it first.
package agents

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/samznd/layrgo/internal/fileutil"
	"github.com/samznd/layrgo/internal/project"
	"github.com/samznd/layrgo/internal/templates"
	"github.com/samznd/layrgo/internal/textdiff"
)

// What Update did, or would do, to one file.
const (
	Created   = "created"
	Updated   = "updated"
	Unchanged = "unchanged"
)

// Options are the inputs of Update.
type Options struct {
	Root   string // the project root, the directory with go.mod
	DryRun bool   // report the changes and diffs; write nothing
}

// Result is what Update did. Its JSON form is the --json output.
type Result struct {
	Files            []File `json:"files"`
	DryRun           bool   `json:"dry_run,omitempty"`
	StackFileWritten bool   `json:"stack_file_written,omitempty"`
}

// File is the outcome for one agent file.
type File struct {
	Path   string `json:"path"`
	Status string `json:"status"`           // Created, Updated or Unchanged
	Backup string `json:"backup,omitempty"` // where the old content went
	Diff   string `json:"diff,omitempty"`   // dry runs only
}

// Changed reports whether any file was, or would be, written.
func (r *Result) Changed() bool {
	for _, f := range r.Files {
		if f.Status != Unchanged {
			return true
		}
	}
	return false
}

// Update renders the agent files for the project's stack and writes the ones
// that differ, backing up the old content of each file it replaces.
func Update(opts Options) (*Result, error) {
	gomod, err := os.ReadFile(filepath.Join(opts.Root, "go.mod"))
	if err != nil {
		return nil, errors.New("no go.mod here; run this from the root of a Layrgo project")
	}
	module := project.ModulePath(gomod)
	if module == "" {
		return nil, errors.New("go.mod declares no module")
	}
	stack, detected, err := project.LoadStack(opts.Root, gomod)
	if err != nil {
		return nil, err
	}

	files, err := templates.AgentFiles(module, stack.Framework, stack.Database, stack.ORM)
	if err != nil {
		return nil, err
	}

	res := &Result{DryRun: opts.DryRun}
	for _, f := range files {
		change, err := update(opts, f)
		if err != nil {
			return nil, err
		}
		res.Files = append(res.Files, change)
	}

	if detected && !opts.DryRun {
		if err := fileutil.CreateFile(filepath.Join(opts.Root, project.StackFile), stack.Marshal()); err != nil {
			return nil, err
		}
		res.StackFileWritten = true
	}
	return res, nil
}

func update(opts Options, f templates.GeneratedFile) (File, error) {
	path := filepath.Join(opts.Root, filepath.FromSlash(f.Rel))
	change := File{Path: f.Rel}

	old, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		change.Status = Created
	case err != nil:
		return change, fmt.Errorf("read %s: %w", f.Rel, err)
	case string(old) == f.Content:
		change.Status = Unchanged
		return change, nil
	default:
		change.Status = Updated
	}

	if opts.DryRun {
		change.Diff = textdiff.Unified(f.Rel, string(old), f.Content)
		return change, nil
	}

	if change.Status == Updated {
		backup, err := freeBackupPath(path)
		if err != nil {
			return change, err
		}
		if err := os.WriteFile(backup, old, 0o644); err != nil {
			return change, fmt.Errorf("back up %s: %w", f.Rel, err)
		}
		rel, _ := filepath.Rel(opts.Root, backup)
		change.Backup = filepath.ToSlash(rel)
	}
	if err := fileutil.CreateFile(path, f.Content); err != nil {
		return change, err
	}
	return change, nil
}

// freeBackupPath returns path.bak, or path.bak.2, path.bak.3, ... when that is
// taken, so an earlier backup is never overwritten.
func freeBackupPath(path string) (string, error) {
	for n := 1; n < 100; n++ {
		candidate := path + ".bak"
		if n > 1 {
			candidate = fmt.Sprintf("%s.bak.%d", path, n)
		}
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("too many backups of %s; delete some", filepath.Base(path))
}
