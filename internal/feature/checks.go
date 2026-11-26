package feature

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const checkTimeout = 5 * time.Minute

// Check is the result of one command run after writing the feature.
type Check struct {
	Name   string // gofmt, build, vet, test
	OK     bool
	Output string
}

// RunChecks runs gofmt, go build, go vet and the new feature's tests from the
// project root. It runs all four, even after a failure, so the report is
// complete.
func RunChecks(root, pkg string) []Check {
	gofmt := gofmtPath()
	return []Check{
		run(root, "gofmt", gofmt, true, "-l", "."),
		run(root, "build", "go", false, "build", "./..."),
		run(root, "vet", "go", false, "vet", "./..."),
		run(root, "test", "go", false, "test", "./internal/"+pkg+"/"),
	}
}

// run executes a command. With mustBeQuiet, any output is a failure too: that
// is how "gofmt -l" reports unformatted files.
func run(dir, name, bin string, mustBeQuiet bool, args ...string) Check {
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	ok := err == nil && (!mustBeQuiet || output == "")
	if err != nil && output == "" {
		output = err.Error()
	}
	if mustBeQuiet && err == nil && output != "" {
		output = "not gofmt-clean:\n" + output
	}
	return Check{Name: name, OK: ok, Output: output}
}

// gofmtPath finds gofmt on PATH, else next to the go command.
func gofmtPath() string {
	if p, err := exec.LookPath("gofmt"); err == nil {
		return p
	}
	if out, err := exec.Command("go", "env", "GOROOT").Output(); err == nil {
		name := "gofmt"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		return filepath.Join(strings.TrimSpace(string(out)), "bin", name)
	}
	return "gofmt"
}
