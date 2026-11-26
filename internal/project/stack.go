// Package project reads what Layrgo knows about an existing generated project:
// its module path and its stack (framework, database and ORM), recorded in
// .layrgo.json or, for projects made before v1.5.0, detected from go.mod.
package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// StackFile is the file at the project root that records the stack.
const StackFile = ".layrgo.json"

var (
	frameworks = []string{"fiber", "gin", "echo", "chi", "iris", "stdlib"}
	databases  = []string{"postgres", "mysql", "sqlite"}
	orms       = []string{"none", "gorm", "xorm", "ent"}
)

// Stack is the content of .layrgo.json. Values are lowercase.
type Stack struct {
	Version   int    `json:"version"`
	Framework string `json:"framework"`
	Database  string `json:"database"`
	ORM       string `json:"orm"`
}

// NewStack builds a Stack from any casing ("Gin", "Postgres", "GORM"). An
// empty orm means "none".
func NewStack(framework, database, orm string) Stack {
	if orm == "" {
		orm = "none"
	}
	return Stack{
		Version:   1,
		Framework: strings.ToLower(framework),
		Database:  strings.ToLower(database),
		ORM:       strings.ToLower(orm),
	}
}

// Marshal renders the stack as the content of .layrgo.json.
func (s Stack) Marshal() string {
	b, _ := json.MarshalIndent(s, "", "  ")
	return string(b) + "\n"
}

// Validate reports values Layrgo does not support.
func (s Stack) Validate() error {
	var bad []string
	if !slices.Contains(frameworks, s.Framework) {
		bad = append(bad, fmt.Sprintf("framework %q", s.Framework))
	}
	if !slices.Contains(databases, s.Database) {
		bad = append(bad, fmt.Sprintf("database %q", s.Database))
	}
	if !slices.Contains(orms, s.ORM) {
		bad = append(bad, fmt.Sprintf("orm %q", s.ORM))
	}
	if len(bad) > 0 {
		return fmt.Errorf("unsupported %s", strings.Join(bad, ", "))
	}
	return nil
}

// LoadStack reads the stack from .layrgo.json in root, or detects it from
// gomod when the file is missing. detected reports the second case, so the
// caller can write the file for next time.
func LoadStack(root string, gomod []byte) (stack Stack, detected bool, err error) {
	b, err := os.ReadFile(filepath.Join(root, StackFile))
	if errors.Is(err, os.ErrNotExist) {
		stack, err := DetectStack(gomod)
		return stack, true, err
	}
	if err != nil {
		return Stack{}, false, fmt.Errorf("read %s: %w", StackFile, err)
	}

	var raw Stack
	if err := json.Unmarshal(b, &raw); err != nil {
		return Stack{}, false, fmt.Errorf("read %s: %w", StackFile, err)
	}
	stack = NewStack(raw.Framework, raw.Database, raw.ORM)
	stack.Version = raw.Version
	if err := stack.Validate(); err != nil {
		return Stack{}, false, fmt.Errorf("%s: %w", StackFile, err)
	}
	return stack, false, nil
}

// DetectStack works out the stack from the modules go.mod requires.
func DetectStack(gomod []byte) (Stack, error) {
	mods := requiredModules(gomod)
	requires := func(path string) bool {
		return slices.ContainsFunc(mods, func(m string) bool {
			return m == path || strings.HasPrefix(m, path+"/")
		})
	}

	s := NewStack("stdlib", "", "none")
	switch {
	case requires("github.com/gin-gonic/gin"):
		s.Framework = "gin"
	case requires("github.com/labstack/echo"):
		s.Framework = "echo"
	case requires("github.com/gofiber/fiber"):
		s.Framework = "fiber"
	case requires("github.com/go-chi/chi"):
		s.Framework = "chi"
	case requires("github.com/kataras/iris"):
		s.Framework = "iris"
	}

	switch {
	case requires("github.com/lib/pq"), requires("gorm.io/driver/postgres"):
		s.Database = "postgres"
	case requires("github.com/go-sql-driver/mysql"), requires("gorm.io/driver/mysql"):
		s.Database = "mysql"
	case requires("github.com/mattn/go-sqlite3"), requires("gorm.io/driver/sqlite"):
		s.Database = "sqlite"
	default:
		return s, fmt.Errorf("could not tell the database from go.mod (no lib/pq, go-sql-driver/mysql or go-sqlite3); create %s, e.g. %s",
			StackFile, strings.TrimSpace(NewStack("gin", "postgres", "none").Marshal()))
	}

	switch {
	case requires("gorm.io/gorm"):
		s.ORM = "gorm"
	case requires("xorm.io/xorm"):
		s.ORM = "xorm"
	case requires("entgo.io/ent"):
		s.ORM = "ent"
	}
	return s, nil
}

// ModulePath returns the module path declared in go.mod, or "".
func ModulePath(gomod []byte) string {
	for _, line := range strings.Split(string(gomod), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(stripComment(line)), "module "); ok {
			return strings.Trim(strings.TrimSpace(rest), `"`)
		}
	}
	return ""
}

// requiredModules lists the module paths in go.mod's require directives.
func requiredModules(gomod []byte) []string {
	var mods []string
	inBlock := false
	for _, line := range strings.Split(string(gomod), "\n") {
		line = strings.TrimSpace(stripComment(line))
		switch {
		case line == "":
		case line == "require (":
			inBlock = true
		case inBlock && line == ")":
			inBlock = false
		case inBlock:
			mods = append(mods, strings.Fields(line)[0])
		case strings.HasPrefix(line, "require "):
			if f := strings.Fields(strings.TrimPrefix(line, "require ")); len(f) > 0 {
				mods = append(mods, f[0])
			}
		}
	}
	return mods
}

func stripComment(s string) string {
	if i := strings.Index(s, "//"); i >= 0 {
		return s[:i]
	}
	return s
}
