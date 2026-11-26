package templates

import (
	"fmt"
	"go/format"
	"path/filepath"
	"strings"
)

// FeatureSpec describes a feature to generate.
type FeatureSpec struct {
	Module    string
	Package   string // "order"
	Type      string // "Order"
	Plural    string // "orders": the route and the table
	Framework string // lowercase, as in .layrgo.json
	Database  string
	ORM       string
	Fields    []Field
}

// GeneratedFile is one file of a generated feature.
type GeneratedFile struct {
	Rel     string // relative to the project root, with forward slashes
	Content string
}

// featureData is what the feature templates render against.
type featureData struct {
	Module, Package, Type, Plural, Table string
	Framework, Database, ORM             string
	Article                              string // "a" or "an", before Package

	IDTag  string
	Fields []fieldData

	ModelImportsTime bool
	TestImportsTime  bool
	HasRequired      bool // any required string/text field: the service validates

	// Raw SQL, for database/sql stores.
	CreateTableSQL string
	InsertSQL      string
	InsertArgs     string
	SelectCols     string
	ScanArgs       string
	IDPlaceholder  string
}

type fieldData struct {
	Name, GoName, GoType, Tag string
	Optional                  bool
	Required                  bool // a required string or text: must not be blank
	TestValue                 string
}

// FeatureFiles renders every file of a new feature, gofmt'ed.
func FeatureFiles(f FeatureSpec) ([]GeneratedFile, error) {
	data, err := newFeatureData(f)
	if err != nil {
		return nil, err
	}

	store := "feature/store_sql.go.tmpl"
	switch data.ORM {
	case "gorm":
		store = "feature/store_gorm.go.tmpl"
	case "xorm":
		store = "feature/store_xorm.go.tmpl"
	}

	parts := []struct{ file, tmpl string }{
		{"model.go", "feature/model.go.tmpl"},
		{"store.go", store},
		{"service.go", "feature/service.go.tmpl"},
		{"service_test.go", "feature/service_test.go.tmpl"},
		{"handler.go", "feature/handler.go.tmpl"},
		{"routes.go", "feature/routes.go.tmpl"},
	}

	out := make([]GeneratedFile, 0, len(parts))
	for _, p := range parts {
		src, err := render(p.tmpl, data)
		if err != nil {
			return nil, err
		}
		formatted, err := format.Source([]byte(src))
		if err != nil {
			return nil, fmt.Errorf("format generated %s: %w\n%s", p.file, err, src)
		}
		out = append(out, GeneratedFile{
			Rel:     filepath.ToSlash(filepath.Join("internal", f.Package, p.file)),
			Content: string(formatted),
		})
	}
	return out, nil
}

func newFeatureData(f FeatureSpec) (featureData, error) {
	d := featureData{
		Module:    f.Module,
		Package:   f.Package,
		Type:      f.Type,
		Plural:    f.Plural,
		Table:     f.Plural,
		Framework: strings.ToLower(f.Framework),
		Database:  strings.ToLower(f.Database),
		ORM:       strings.ToLower(f.ORM),
		Article:   "a",
	}
	if d.ORM == "" {
		d.ORM = "none"
	}
	if !frameworks[d.Framework] {
		return d, fmt.Errorf("unsupported framework %q", f.Framework)
	}
	if _, ok := dbSpecs[d.Database]; !ok {
		return d, fmt.Errorf("unsupported database %q", f.Database)
	}
	if !orms[d.ORM] {
		return d, fmt.Errorf("unsupported orm %q", f.ORM)
	}
	if len(f.Fields) == 0 {
		return d, fmt.Errorf("a feature needs at least one field")
	}
	if strings.ContainsRune("aeiou", rune(f.Package[0])) {
		d.Article = "an"
	}

	switch d.ORM {
	case "gorm":
		d.IDTag = `json:"id" gorm:"primaryKey"`
	case "xorm":
		d.IDTag = `json:"id" xorm:"pk autoincr 'id'"`
	default:
		d.IDTag = `json:"id"`
	}

	for _, field := range f.Fields {
		k, ok := fieldKinds[field.Kind]
		if !ok {
			return d, fmt.Errorf("unsupported field type %q", field.Kind)
		}
		fd := fieldData{
			Name:      field.Name,
			GoName:    GoName(field.Name),
			GoType:    k.goType,
			Optional:  field.Optional,
			Required:  !field.Optional && k.goType == "string",
			TestValue: k.testValue,
		}
		if field.Optional {
			fd.GoType = "*" + k.goType
		}
		fd.Tag = fieldTag(d.ORM, field, k)
		d.Fields = append(d.Fields, fd)

		if k.needsTime {
			d.ModelImportsTime = true
			if !field.Optional {
				d.TestImportsTime = true
			}
		}
		if fd.Required {
			d.HasRequired = true
		}
	}

	d.buildSQL(f.Fields)
	return d, nil
}

func fieldTag(orm string, f Field, k kindInfo) string {
	tag := fmt.Sprintf(`json:"%s"`, f.Name)
	switch orm {
	case "gorm":
		settings := []string{"column:" + f.Name}
		if !f.Optional {
			settings = append(settings, "not null")
		}
		if k.gormTag != "" {
			settings = append(settings, k.gormTag)
		}
		tag += fmt.Sprintf(` gorm:"%s"`, strings.Join(settings, ";"))
	case "xorm":
		settings := []string{"'" + f.Name + "'"}
		if k.xormTag != "" {
			settings = append(settings, k.xormTag)
		}
		if f.Optional {
			settings = append(settings, "null")
		} else {
			settings = append(settings, "notnull")
		}
		tag += fmt.Sprintf(` xorm:"%s"`, strings.Join(settings, " "))
	}
	return tag
}

func (d *featureData) buildSQL(fields []Field) {
	idCol := map[string]string{
		"postgres": "BIGSERIAL PRIMARY KEY",
		"mysql":    "BIGINT AUTO_INCREMENT PRIMARY KEY",
		"sqlite":   "INTEGER PRIMARY KEY AUTOINCREMENT",
	}[d.Database]

	width := len("id")
	for _, f := range fields {
		width = max(width, len(f.Name))
	}

	cols := []string{fmt.Sprintf("%-*s %s", width, "id", idCol)}
	names := []string{"id"}
	var insertCols, placeholders, insertArgs []string
	scanArgs := []string{"&item.ID"}
	for i, f := range fields {
		null := " NOT NULL"
		if f.Optional {
			null = ""
		}
		cols = append(cols, fmt.Sprintf("%-*s %s%s", width, f.Name, fieldKinds[f.Kind].sqlTypes[d.Database], null))
		names = append(names, f.Name)
		insertCols = append(insertCols, f.Name)
		placeholders = append(placeholders, d.placeholder(i+1))
		insertArgs = append(insertArgs, "item."+GoName(f.Name))
		scanArgs = append(scanArgs, "&item."+GoName(f.Name))
	}

	d.CreateTableSQL = "CREATE TABLE IF NOT EXISTS " + d.Table + " (\n\t\t" +
		strings.Join(cols, ",\n\t\t") + "\n\t)"
	d.InsertSQL = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		d.Table, strings.Join(insertCols, ", "), strings.Join(placeholders, ", "))
	d.InsertArgs = strings.Join(insertArgs, ", ")
	d.SelectCols = strings.Join(names, ", ")
	d.ScanArgs = strings.Join(scanArgs, ", ")
	d.IDPlaceholder = d.placeholder(1)
}

func (d *featureData) placeholder(n int) string {
	if d.Database == "postgres" {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

// RouterVar is the variable cmd/main.go passes to SetupRoutes for a framework.
func RouterVar(framework string) string {
	return frameworkInfo[strings.ToLower(framework)].routerVar
}
