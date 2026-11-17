package templates

import (
	"fmt"
	"strings"
)

// frameworkInfo is what AGENTS.md says about each framework.
var frameworkInfo = map[string]struct{ name, routerVar string }{
	"fiber":  {"Fiber v3", "api"},
	"gin":    {"Gin", "api"},
	"echo":   {"Echo", "api"},
	"chi":    {"Chi", "r"},
	"iris":   {"Iris", "app"},
	"stdlib": {"net/http (standard library)", "mux"},
}

var (
	databaseNames = map[string]string{"postgres": "Postgres", "mysql": "MySQL", "sqlite": "SQLite"}
	ormNames      = map[string]string{
		"none": "database/sql",
		"gorm": "GORM",
		"xorm": "XORM",
		"ent":  "database/sql (Ent is installed; see below)",
	}
)

// AgentsFile renders AGENTS.md: instructions for AI coding agents, specific to
// the project's framework, database and ORM. An empty orm means "none".
func AgentsFile(module, framework, database, orm string) (string, error) {
	framework = strings.ToLower(framework)
	database = strings.ToLower(database)
	orm = strings.ToLower(orm)
	if orm == "" {
		orm = "none"
	}

	fw, ok := frameworkInfo[framework]
	if !ok {
		return "", fmt.Errorf("unsupported framework %q", framework)
	}
	dbName, ok := databaseNames[database]
	if !ok {
		return "", fmt.Errorf("unsupported database %q", database)
	}
	ormName, ok := ormNames[orm]
	if !ok {
		return "", fmt.Errorf("unsupported orm %q", orm)
	}

	return render("agents/AGENTS.md.tmpl", Data{
		Module:        module,
		Framework:     framework,
		Database:      database,
		ORM:           orm,
		FrameworkName: fw.name,
		DatabaseName:  dbName,
		ORMName:       ormName,
		RouterVar:     fw.routerVar,
	})
}

// AGENTS.md is the single source of instructions. Codex, Cursor, Windsurf,
// Zed, Amp and others read it directly; the files below point the tools that
// look for their own filename back to it, so nothing is duplicated.

// ClaudeFile renders CLAUDE.md for Claude Code, which imports AGENTS.md.
func ClaudeFile() string {
	return "@AGENTS.md\n"
}

// GeminiFile renders GEMINI.md for Gemini CLI, which imports AGENTS.md.
func GeminiFile() string {
	return "@./AGENTS.md\n"
}

// CopilotFile renders .github/copilot-instructions.md for GitHub Copilot.
// Copilot cannot import files, so it repeats the core rules and points to
// AGENTS.md for the rest.
func CopilotFile() string {
	return `# Copilot instructions

Read and follow [AGENTS.md](../AGENTS.md) at the repository root before making
changes. It has the commands, the layout and a full example of adding a feature
for this project's framework and database.

The core rules:

- One package per feature in ` + "`internal/<feature>/`" + `, with ` + "`handler.go`" + `, ` + "`service.go`" + `,
  ` + "`store.go`" + ` and ` + "`routes.go`" + `. Dependencies point one way: handler → service → store.
- Each layer depends on a small unexported interface it declares itself.
- Only ` + "`cmd/main.go`" + ` reads ` + "`config.DB`" + `; wire new features there.
- Pass the request's ` + "`context.Context`" + ` down to every database call.
- Don't add or upgrade dependencies unless the task needs it.
- Before finishing: ` + "`gofmt -l .`" + ` prints nothing, and ` + "`go vet ./...`" + `,
  ` + "`go build ./...`" + ` and ` + "`go test ./...`" + ` pass.
`
}
