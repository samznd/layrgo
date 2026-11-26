package feature

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/samznd/layrgo/internal/templates"
)

var fieldNameRe = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// sqlReserved are words that can't be a column or table name without quoting
// in at least one of Postgres, MySQL and SQLite. The generated SQL doesn't
// quote names, so they are rejected up front.
var sqlReserved = setOf(
	"all", "alter", "and", "any", "as", "asc", "between", "by", "case", "check",
	"column", "constraint", "create", "cross", "current_date", "current_time",
	"current_timestamp", "current_user", "default", "delete", "desc", "distinct",
	"drop", "else", "end", "exists", "foreign", "from", "grant", "group", "groups",
	"having", "in", "index", "inner", "insert", "interval", "into", "is", "join",
	"key", "keys", "left", "like", "limit", "match", "natural", "not", "null",
	"offset", "on", "or", "order", "outer", "primary", "range", "rank",
	"references", "right", "row", "rows", "select", "set", "table", "then", "to",
	"union", "unique", "update", "user", "using", "values", "when", "where",
	"window", "with",
)

// ParseFields parses a --fields value such as
// "customer:string, note:string?, total:int".
func ParseFields(spec string) ([]templates.Field, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, fmt.Errorf("--fields needs at least one field, e.g. --fields \"title:string,done:bool\"")
	}

	var fields []templates.Field
	seen := map[string]bool{}
	goNames := map[string]string{}
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, kind, ok := strings.Cut(part, ":")
		name, kind = strings.TrimSpace(name), strings.TrimSpace(kind)
		if !ok || name == "" || kind == "" {
			return nil, fmt.Errorf("invalid field %q: use name:type, e.g. title:string", part)
		}

		optional := strings.HasSuffix(kind, "?")
		kind = strings.ToLower(strings.TrimSuffix(kind, "?"))

		if !fieldNameRe.MatchString(name) {
			return nil, fmt.Errorf("invalid field name %q: use snake_case (lowercase letters, digits and single underscores, starting with a letter)", name)
		}
		if name == "id" {
			return nil, fmt.Errorf("field %q is added automatically; don't declare it", name)
		}
		if name == "table_name" {
			return nil, fmt.Errorf("field %q would clash with the generated TableName method", name)
		}
		if sqlReserved[name] {
			return nil, fmt.Errorf("field name %q is a reserved word in SQL; pick another, e.g. %s_value", name, name)
		}
		if !templates.IsFieldKind(kind) {
			return nil, fmt.Errorf("invalid type %q for field %q (allowed: %s; add ? for optional)",
				kind, name, strings.Join(templates.FieldKinds(), ", "))
		}
		if seen[name] {
			return nil, fmt.Errorf("field %q is declared twice", name)
		}
		goName := templates.GoName(name)
		if other, dup := goNames[goName]; dup {
			return nil, fmt.Errorf("fields %q and %q both become %s in Go", other, name, goName)
		}
		seen[name] = true
		goNames[goName] = name

		fields = append(fields, templates.Field{Name: name, Kind: kind, Optional: optional})
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("--fields needs at least one field, e.g. --fields \"title:string,done:bool\"")
	}
	return fields, nil
}
