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

// Data is what the templates render against.
type Data struct {
	Module    string
	Framework string
	ORM       string
	Database  string

	// Set for the config templates only.
	DriverImport  string
	DriverName    string
	DSNBlock      string
	GormDialector string
}

// FeaturePackage is the package the sample endpoint goes into. A feature's
// handler, service, store and routes all live in internal/<feature>.
const FeaturePackage = "message"

var (
	frameworks = map[string]bool{"fiber": true, "gin": true, "echo": true, "chi": true, "iris": true, "stdlib": true}
	orms       = map[string]bool{"none": true, "gorm": true, "xorm": true, "ent": true}
)

// render executes an embedded template. Missing keys are errors.
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

// HandlerFile renders the handler file.
func HandlerFile(module, framework string) (string, error) {
	framework = strings.ToLower(framework)
	if !frameworks[framework] {
		return "", fmt.Errorf("unsupported framework %q", framework)
	}
	return render("handler/"+framework+".go.tmpl", Data{Module: module, Framework: framework})
}

// RoutesFile renders the routes file.
func RoutesFile(module, framework string) (string, error) {
	framework = strings.ToLower(framework)
	if !frameworks[framework] {
		return "", fmt.Errorf("unsupported framework %q", framework)
	}
	return render("routes/"+framework+".go.tmpl", Data{Module: module, Framework: framework})
}

// ServiceFile renders the service file.
func ServiceFile(module string) (string, error) {
	return render("service/service.go.tmpl", Data{Module: module})
}

// StoreFile renders the data-access file for the given ORM.
func StoreFile(module, orm string) (string, error) {
	orm = strings.ToLower(orm)
	if !orms[orm] {
		return "", fmt.Errorf("unsupported orm %q", orm)
	}

	// Ent uses database/sql too; its file just has an extra note.
	name := map[string]string{
		"none": "sql",
		"gorm": "gorm",
		"xorm": "xorm",
		"ent":  "ent",
	}[orm]

	return render("store/"+name+".go.tmpl", Data{Module: module, ORM: orm})
}
