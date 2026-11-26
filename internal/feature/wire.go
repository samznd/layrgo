package feature

import (
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"regexp"
	"strings"

	"github.com/samznd/layrgo/internal/templates"
)

// Markers that "layrgo init" writes into cmd/main.go from v1.5.0.
const (
	WireMarker   = "// layrgo:wire"
	RoutesMarker = "// layrgo:routes"
)

// ErrCannotWire means cmd/main.go has no place to insert the feature. The
// caller prints Wiring.Manual instead and leaves the file alone.
var ErrCannotWire = errors.New("could not find where to wire the feature in cmd/main.go")

var (
	handlerLineRe = regexp.MustCompile(`^\s*\w+Handler\s*:?=\s*\w+\.NewHandler\(`)
	routesLineRe  = regexp.MustCompile(`^\s*\w+\.SetupRoutes\(`)
	routerVarRe   = regexp.MustCompile(`(?m)^\s*\w+\.SetupRoutes\(\s*(\w+)\s*,`)
)

// Wiring is the code a feature adds to cmd/main.go.
type Wiring struct {
	Import string   // the import path
	Setup  []string // store, Migrate, service, handler; unindented
	Routes string   // the SetupRoutes call; unindented
}

// Manual is the text shown when the wiring has to be pasted by hand.
func (w Wiring) Manual() string {
	var b strings.Builder
	b.WriteString("Add the import to cmd/main.go:\n\n\t\"" + w.Import + "\"\n\n")
	b.WriteString("Wire the feature next to the others:\n\n")
	for _, l := range w.Setup {
		b.WriteString("\t" + l + "\n")
	}
	b.WriteString("\nRegister its routes after the other SetupRoutes calls:\n\n\t" + w.Routes + "\n")
	return b.String()
}

func newWiring(src []byte, module, framework string, n Names) Wiring {
	routerVar := templates.RouterVar(framework)
	if m := routerVarRe.FindSubmatch(src); m != nil {
		routerVar = string(m[1])
	}
	p := n.Package
	return Wiring{
		Import: module + "/internal/" + p,
		Setup: []string{
			fmt.Sprintf("%sStore := %s.NewStore(config.DB)", p, p),
			fmt.Sprintf("if err := %sStore.Migrate(context.Background()); err != nil {", p),
			fmt.Sprintf("\tlog.Fatalf(\"migrate %s: %%v\", err)", n.Plural),
			"}",
			fmt.Sprintf("%sService := %s.NewService(%sStore)", p, p, p),
			fmt.Sprintf("%sHandler := %s.NewHandler(%sService)", p, p, p),
		},
		Routes: fmt.Sprintf("%s.SetupRoutes(%s, %sHandler)", p, routerVar, p),
	}
}

// WireResult is the outcome of Wire.
type WireResult struct {
	Content      []byte // the new cmd/main.go; equal to the input when nothing changed
	AlreadyWired bool   // the feature was wired before (e.g. a --force rerun)
	Wiring       Wiring
}

// Wire inserts the feature into cmd/main.go: above the layrgo markers when
// they are there, else after the last feature's handler and SetupRoutes lines.
// It returns ErrCannotWire, with the input unchanged, when it finds neither or
// the file does not parse.
func Wire(src []byte, module, framework string, n Names) (WireResult, error) {
	res := WireResult{Content: src, Wiring: newWiring(src, module, framework, n)}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", src, parser.ParseComments)
	if err != nil {
		return res, fmt.Errorf("%w: it does not parse: %v", ErrCannotWire, err)
	}

	hasImport := false
	for _, imp := range file.Imports {
		if strings.Trim(imp.Path.Value, `"`) == res.Wiring.Import {
			hasImport = true
		}
	}
	if hasImport && strings.Contains(string(src), n.Package+".NewHandler(") {
		res.AlreadyWired = true
		return res, nil
	}

	importLine, err := importInsertLine(fset, file, module)
	if err != nil {
		return res, err
	}

	lines := strings.Split(string(src), "\n")

	setupAt, setupAbove := findInsert(lines, WireMarker, handlerLineRe)
	routesAt, routesAbove := findInsert(lines, RoutesMarker, routesLineRe)
	if setupAt < 0 || routesAt < 0 {
		return res, fmt.Errorf("%w: no %q or %q markers and no existing feature to follow", ErrCannotWire, WireMarker, RoutesMarker)
	}
	if setupAt > routesAt {
		return res, fmt.Errorf("%w: the features are wired after their routes", ErrCannotWire)
	}

	// Insert bottom-up so earlier line numbers stay valid.
	routes := []string{res.Wiring.Routes}
	lines = insertLines(lines, routesAt, routesAbove, routes)

	setup := res.Wiring.Setup
	if setupAbove {
		setup = append(append([]string{}, setup...), "")
	} else {
		setup = append([]string{""}, setup...)
	}
	lines = insertLines(lines, setupAt, setupAbove, setup)

	if !hasImport {
		lines = append(lines[:importLine], append([]string{"\t\"" + res.Wiring.Import + "\""}, lines[importLine:]...)...)
	}

	out, err := format.Source([]byte(strings.Join(lines, "\n")))
	if err != nil {
		return WireResult{Content: src, Wiring: res.Wiring}, fmt.Errorf("%w: the edited file does not parse: %v", ErrCannotWire, err)
	}
	res.Content = out
	return res, nil
}

// importInsertLine returns the 0-based line before which the new import goes:
// after the project's own imports, or after the last import.
func importInsertLine(fset *token.FileSet, file *ast.File, module string) (int, error) {
	var block *ast.GenDecl
	for _, d := range file.Decls {
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			block = gd
		}
	}
	if block == nil || !block.Lparen.IsValid() {
		return 0, fmt.Errorf("%w: it has no import ( ... ) block", ErrCannotWire)
	}

	var last, lastOwn ast.Spec
	for _, spec := range block.Specs {
		last = spec
		if strings.HasPrefix(strings.Trim(spec.(*ast.ImportSpec).Path.Value, `"`), module+"/") {
			lastOwn = spec
		}
	}
	switch {
	case lastOwn != nil:
		return fset.Position(lastOwn.End()).Line, nil
	case last != nil:
		return fset.Position(last.End()).Line, nil
	default:
		return fset.Position(block.Rparen).Line - 1, nil
	}
}

// findInsert finds where to insert: the marker line (insert above it), or the
// line after the last statement matching re (insert below it). It returns -1
// when it finds neither.
func findInsert(lines []string, marker string, re *regexp.Regexp) (at int, above bool) {
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), marker) {
			return i, true
		}
	}
	at = -1
	for i, l := range lines {
		if re.MatchString(l) {
			at = statementEnd(lines, i)
		}
	}
	return at, false
}

// statementEnd returns the last line of the statement starting at line i, by
// balancing parentheses.
func statementEnd(lines []string, i int) int {
	depth := 0
	for j := i; j < len(lines); j++ {
		depth += strings.Count(lines[j], "(") - strings.Count(lines[j], ")")
		if depth <= 0 {
			return j
		}
	}
	return i
}

// insertLines inserts block above lines[at] or below it, indented like it.
func insertLines(lines []string, at int, above bool, block []string) []string {
	ind := leadingSpace(lines[at])
	indented := make([]string, len(block))
	for i, l := range block {
		if l != "" {
			indented[i] = ind + l
		}
	}
	pos := at
	if !above {
		pos = at + 1
	}
	out := make([]string, 0, len(lines)+len(block))
	out = append(out, lines[:pos]...)
	out = append(out, indented...)
	return append(out, lines[pos:]...)
}

func leadingSpace(s string) string {
	return s[:len(s)-len(strings.TrimLeft(s, " \t"))]
}
