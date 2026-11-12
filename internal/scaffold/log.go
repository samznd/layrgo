package scaffold

import (
	"fmt"
	"io"
)

// progressOut is where per-file progress goes. Tests send it to io.Discard.
var progressOut io.Writer = stdout{}

type stdout struct{}

func (stdout) Write(p []byte) (int, error) { return fmt.Print(string(p)) }

// SetProgressOutput redirects progress output and returns a function restoring
// the previous destination.
func SetProgressOutput(w io.Writer) func() {
	prev := progressOut
	progressOut = w
	return func() { progressOut = prev }
}

func logCreated(rel string) {
	fmt.Fprintf(progressOut, "✅ Created %s\n", rel)
}

func logStep(format string, args ...any) {
	fmt.Fprintf(progressOut, format+"\n", args...)
}
