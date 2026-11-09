package templates

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed files
var files embed.FS

// Data is the input every template renders against. Named fields replace the
// repeated positional %s arguments the generator used previously, so a template
// referencing an unknown field fails loudly instead of silently misordering.
type Data struct {
	Module    string
	Framework string
	ORM       string
	Database  string

	// Database-specific pieces, set by the config templates' caller.
	DriverImport  string
	DriverName    string
	DSNBlock      string
	GormDialector string
}

var (
	frameworks = map[string]bool{"fiber": true, "gin": true, "echo": true, "chi": true, "iris": true, "stdlib": true}
	orms       = map[string]bool{"none": true, "gorm": true, "xorm": true, "ent": true}
)

// render executes the named embedded template. Missing keys are an error rather
// than an empty string, which is what lets a bad stack fail at generation time.
func render(name string, data Data, extra ...string) (string, error) {
	patterns := append([]string{"files/" + name}, extra...)

	tmpl := template.New(baseName(name)).Option("missingkey=error")
	tmpl, err := tmpl.ParseFS(files, patterns...)
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, baseName(name), data); err != nil {
		return "", fmt.Errorf("render template %s: %w", name, err)
	}
	return buf.String(), nil
}

func baseName(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}

// MainFile renders cmd/main.go for the given framework.
func MainFile(module, framework string) (string, error) {
	framework = strings.ToLower(framework)
	if !frameworks[framework] {
		return "", fmt.Errorf("unsupported framework %q", framework)
	}
	return render("main/"+framework+".go.tmpl", Data{Module: module, Framework: framework}, "files/main/di.tmpl")
}

// HandlerFile renders internal/handlers/handler.go.
func HandlerFile(module, framework string) (string, error) {
	framework = strings.ToLower(framework)
	if !frameworks[framework] {
		return "", fmt.Errorf("unsupported framework %q", framework)
	}
	return render("handler/"+framework+".go.tmpl", Data{Module: module, Framework: framework})
}

// RoutesFile renders internal/routes/routes.go.
func RoutesFile(module, framework string) (string, error) {
	framework = strings.ToLower(framework)
	if !frameworks[framework] {
		return "", fmt.Errorf("unsupported framework %q", framework)
	}
	return render("routes/"+framework+".go.tmpl", Data{Module: module, Framework: framework})
}

// ServiceFile renders internal/services/service.go.
func ServiceFile(module string) (string, error) {
	return render("service/service.go.tmpl", Data{Module: module})
}

// RepositoryFile renders internal/repositories/repository.go for the given ORM.
func RepositoryFile(module, orm string) (string, error) {
	orm = strings.ToLower(orm)
	if !orms[orm] {
		return "", fmt.Errorf("unsupported orm %q", orm)
	}

	// "none" and Ent both sit on database/sql; Ent's file carries an extra note.
	name := map[string]string{
		"none": "sql",
		"gorm": "gorm",
		"xorm": "xorm",
		"ent":  "ent",
	}[orm]

	return render("repository/"+name+".go.tmpl", Data{Module: module, ORM: orm})
}
