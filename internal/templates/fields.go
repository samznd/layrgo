package templates

import (
	"encoding/json"
	"strings"
)

// Field is one user-declared field of a new feature, already validated.
type Field struct {
	Name     string // snake_case, used for JSON and SQL
	Kind     string // one of FieldKinds
	Optional bool   // pointer in Go, nullable column
}

// kindInfo is everything the generator needs to know about a field type.
type kindInfo struct {
	goType    string
	sqlTypes  map[string]string // by database
	testValue string            // a valid Go literal, for the generated test
	jsonValue string            // an example JSON value, for the curl hint
	gormTag   string            // extra GORM tag settings
	xormTag   string            // extra XORM tag settings
	needsTime bool
}

var fieldKinds = map[string]kindInfo{
	"string": {
		goType:    "string",
		sqlTypes:  map[string]string{"postgres": "TEXT", "mysql": "VARCHAR(255)", "sqlite": "TEXT"},
		testValue: `"example"`,
		jsonValue: `"example"`,
		gormTag:   "size:255",
	},
	"text": {
		goType:    "string",
		sqlTypes:  map[string]string{"postgres": "TEXT", "mysql": "TEXT", "sqlite": "TEXT"},
		testValue: `"example text"`,
		jsonValue: `"example text"`,
		gormTag:   "type:text",
		xormTag:   "text",
	},
	"int": {
		goType:    "int64",
		sqlTypes:  map[string]string{"postgres": "BIGINT", "mysql": "BIGINT", "sqlite": "INTEGER"},
		testValue: "1",
		jsonValue: "1",
	},
	"float": {
		goType:    "float64",
		sqlTypes:  map[string]string{"postgres": "DOUBLE PRECISION", "mysql": "DOUBLE", "sqlite": "REAL"},
		testValue: "1.5",
		jsonValue: "1.5",
	},
	"bool": {
		goType:    "bool",
		sqlTypes:  map[string]string{"postgres": "BOOLEAN", "mysql": "BOOLEAN", "sqlite": "INTEGER"},
		testValue: "true",
		jsonValue: "true",
	},
	"time": {
		goType:    "time.Time",
		sqlTypes:  map[string]string{"postgres": "TIMESTAMPTZ", "mysql": "DATETIME", "sqlite": "TIMESTAMP"},
		testValue: "time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)",
		jsonValue: `"2026-01-01T00:00:00Z"`,
		needsTime: true,
	},
}

// FieldKinds lists the field types "layrgo add feature" accepts, in the order
// the docs show them.
func FieldKinds() []string {
	return []string{"string", "text", "int", "float", "bool", "time"}
}

// IsFieldKind reports whether kind is a supported field type.
func IsFieldKind(kind string) bool {
	_, ok := fieldKinds[kind]
	return ok
}

// GoName turns a snake_case field name into an exported Go name:
// "created_by" becomes "CreatedBy".
func GoName(snake string) string {
	var b strings.Builder
	for _, part := range strings.Split(snake, "_") {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]) + part[1:])
	}
	return b.String()
}

// ExampleJSON is a request body that passes the generated validation, for the
// curl hint printed after "add feature".
func ExampleJSON(fields []Field) string {
	var parts []string
	for _, f := range fields {
		key, _ := json.Marshal(f.Name)
		parts = append(parts, string(key)+":"+fieldKinds[f.Kind].jsonValue)
	}
	return "{" + strings.Join(parts, ",") + "}"
}
