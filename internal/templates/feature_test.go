package templates_test

import (
	"strings"
	"testing"

	"github.com/samznd/layrgo/internal/templates"
)

var orderFields = []templates.Field{
	{Name: "customer", Kind: "string"},
	{Name: "note", Kind: "text", Optional: true},
	{Name: "total", Kind: "int"},
	{Name: "due_at", Kind: "time", Optional: true},
}

func orderSpec(framework, database, orm string) templates.FeatureSpec {
	return templates.FeatureSpec{
		Module: "shop", Package: "order", Type: "Order", Plural: "orders",
		Framework: framework, Database: database, ORM: orm, Fields: orderFields,
	}
}

func featureFiles(t *testing.T, spec templates.FeatureSpec) map[string]string {
	t.Helper()
	files, err := templates.FeatureFiles(spec)
	if err != nil {
		t.Fatalf("%s/%s/%s: %v", spec.Framework, spec.Database, spec.ORM, err)
	}
	byName := map[string]string{}
	for _, f := range files {
		byName[strings.TrimPrefix(f.Rel, "internal/order/")] = f.Content
	}
	return byName
}

// Every stack renders six gofmt'ed files (FeatureFiles formats them, so a
// template that produces invalid Go fails here).
func TestFeatureFiles_AllStacks(t *testing.T) {
	t.Parallel()

	for _, fw := range []string{"fiber", "gin", "echo", "chi", "iris", "stdlib"} {
		for _, db := range []string{"postgres", "mysql", "sqlite"} {
			for _, orm := range []string{"none", "gorm", "xorm", "ent"} {
				files := featureFiles(t, orderSpec(fw, db, orm))
				for _, name := range []string{"model.go", "store.go", "service.go", "service_test.go", "handler.go", "routes.go"} {
					if !strings.HasPrefix(files[name], "package order\n") {
						t.Errorf("%s/%s/%s: %s missing or wrong package", fw, db, orm, name)
					}
				}
				if !strings.Contains(files["handler.go"], "func (h *Handler) List(") {
					t.Errorf("%s/%s/%s: no List handler", fw, db, orm)
				}
			}
		}
	}
}

func TestFeatureFiles_RawSQLPerDatabase(t *testing.T) {
	t.Parallel()

	pg := featureFiles(t, orderSpec("gin", "postgres", "none"))["store.go"]
	for _, want := range []string{
		"id       BIGSERIAL PRIMARY KEY",
		"customer TEXT NOT NULL",
		"note     TEXT,",
		"due_at   TIMESTAMPTZ\n",
		"VALUES ($1, $2, $3, $4) RETURNING id",
		"WHERE id = $1",
	} {
		if !strings.Contains(pg, want) {
			t.Errorf("postgres store misses %q:\n%s", want, pg)
		}
	}

	my := featureFiles(t, orderSpec("gin", "mysql", "none"))["store.go"]
	for _, want := range []string{"BIGINT AUTO_INCREMENT PRIMARY KEY", "customer VARCHAR(255) NOT NULL", "VALUES (?, ?, ?, ?)", "res.LastInsertId()"} {
		if !strings.Contains(my, want) {
			t.Errorf("mysql store misses %q:\n%s", want, my)
		}
	}
	if strings.Contains(my, "$1") {
		t.Error("mysql store uses Postgres placeholders")
	}
}

// "order" is a reserved word in SQL; ORMs must not derive the table name.
func TestFeatureFiles_ORMsPinTheTableName(t *testing.T) {
	t.Parallel()

	for _, orm := range []string{"gorm", "xorm"} {
		model := featureFiles(t, orderSpec("gin", "postgres", orm))["model.go"]
		if !strings.Contains(model, `return "orders"`) {
			t.Errorf("%s model does not pin the table name:\n%s", orm, model)
		}
	}
}

func TestFeatureFiles_ValidatesRequiredStrings(t *testing.T) {
	t.Parallel()

	files := featureFiles(t, orderSpec("chi", "sqlite", "none"))
	if !strings.Contains(files["service.go"], `fmt.Errorf("%w: customer is required", ErrInvalid)`) {
		t.Errorf("no blank check for customer:\n%s", files["service.go"])
	}
	if strings.Contains(files["service.go"], "note is required") {
		t.Error("an optional field is validated as required")
	}
	if !strings.Contains(files["service_test.go"], "func TestService_Create_RequiresCustomer(") {
		t.Error("no test for the required customer field")
	}
	if !strings.Contains(files["service.go"], "// TODO: business rules for creating an order.") {
		t.Errorf("no TODO for business rules:\n%s", files["service.go"])
	}
}

func TestExampleJSON(t *testing.T) {
	t.Parallel()

	got := templates.ExampleJSON(orderFields)
	want := `{"customer":"example","note":"example text","total":1,"due_at":"2026-01-01T00:00:00Z"}`
	if got != want {
		t.Errorf("ExampleJSON = %s, want %s", got, want)
	}
}
