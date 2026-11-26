// Package feature adds a complete feature (model, store, service, handler,
// routes and a test) to an existing Layrgo project, wires it into
// cmd/main.go, and checks that the project still builds.
package feature

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/samznd/layrgo/internal/templates"
)

// Names is everything derived from a feature's name.
type Names struct {
	Package string // "order": the package, folder and variable prefix
	Type    string // "Order"
	Plural  string // "orders": the route and the table
}

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

// reservedNames can't be a feature: Go keywords and predeclared identifiers,
// the project's own packages, packages cmd/main.go or the generated files
// import (the import would be shadowed), and names whose type would clash with
// the generated Store, Service or Handler.
var reservedNames = setOf(
	// Go keywords.
	"break", "case", "chan", "const", "continue", "default", "defer", "else",
	"fallthrough", "for", "func", "go", "goto", "if", "import", "interface",
	"map", "package", "range", "return", "select", "struct", "switch", "type", "var",
	// Predeclared identifiers.
	"any", "bool", "byte", "comparable", "complex64", "complex128", "error",
	"float32", "float64", "int", "int8", "int16", "int32", "int64", "rune",
	"string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
	"true", "false", "iota", "nil", "append", "cap", "clear", "close", "complex",
	"copy", "delete", "imag", "len", "make", "max", "min", "new", "panic",
	"print", "println", "real", "recover",
	// Project packages and folders.
	"main", "config", "utils", "middleware", "internal", "cmd", "pkg",
	// Imports of cmd/main.go and the generated files.
	"context", "log", "http", "time", "fmt", "errors", "strings", "strconv",
	"json", "sql", "testing", "os", "gin", "echo", "fiber", "chi", "iris",
	"gorm", "xorm", "ent",
	// Would clash with the generated types.
	"store", "service", "handler",
)

// NewNames validates name and derives the rest. plural overrides the derived
// plural when it is not empty.
func NewNames(name, plural string) (Names, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Names{}, fmt.Errorf("feature name is required")
	}
	if !nameRe.MatchString(name) {
		return Names{}, fmt.Errorf("invalid feature name %q: use lowercase letters and digits, starting with a letter (e.g. order)", name)
	}
	if reservedNames[name] {
		return Names{}, fmt.Errorf("invalid feature name %q: it is reserved in Go or in a Layrgo project; pick another", name)
	}

	plural = strings.TrimSpace(plural)
	if plural == "" {
		plural = Pluralize(name)
	} else if !nameRe.MatchString(plural) {
		return Names{}, fmt.Errorf("invalid --plural %q: use lowercase letters and digits, starting with a letter", plural)
	}
	if plural == name {
		return Names{}, fmt.Errorf("the plural of %q must differ from it; pass --plural", name)
	}
	if sqlReserved[plural] {
		return Names{}, fmt.Errorf("the table name %q is a reserved word in SQL; pass a different --plural", plural)
	}

	return Names{
		Package: name,
		Type:    templates.GoName(name),
		Plural:  plural,
	}, nil
}

// Pluralize adds "s", or "es" after s, x, z, ch and sh, and turns a final
// consonant + "y" into "ies". Irregular plurals need --plural.
func Pluralize(s string) string {
	switch {
	case strings.HasSuffix(s, "s"), strings.HasSuffix(s, "x"), strings.HasSuffix(s, "z"),
		strings.HasSuffix(s, "ch"), strings.HasSuffix(s, "sh"):
		return s + "es"
	case len(s) > 1 && s[len(s)-1] == 'y' && !strings.ContainsRune("aeiou", rune(s[len(s)-2])):
		return s[:len(s)-1] + "ies"
	default:
		return s + "s"
	}
}

func setOf(items ...string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, it := range items {
		m[it] = true
	}
	return m
}
