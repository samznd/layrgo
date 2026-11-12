package templates_test

import (
	"go/format"
	"strings"
	"testing"

	"github.com/samznd/layrgo/internal/templates"
)

func TestStoreFile_RawSQL(t *testing.T) {
	t.Parallel()

	content, err := templates.StoreFile("myapp", "none")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"func NewStore(db *sql.DB)",
		"GetMessage(ctx context.Context) (string, error)",
		"QueryRowContext(ctx,",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("missing %q in:\n%s", want, content)
		}
	}
}

// Previously this test could never fail. It now asserts the specific behavior:
// the xorm repository must check the row count and propagate both error paths.
func TestStoreFile_XORM_ErrorPaths(t *testing.T) {
	t.Parallel()

	content, err := templates.StoreFile("myapp", "xorm")
	if err != nil {
		t.Fatal(err)
	}

	if strings.Count(content, `return "", err`) < 1 {
		t.Error("xorm repository should propagate the query error")
	}
	if !strings.Contains(content, "len(rows) == 0") {
		t.Error("xorm repository should guard against an empty result set")
	}
	if !strings.Contains(content, `return "", fmt.Errorf("no rows returned")`) {
		t.Error("xorm repository should return an error for an empty result set")
	}
}

func TestStoreFile_RejectsUnknownORM(t *testing.T) {
	t.Parallel()

	if _, err := templates.StoreFile("myapp", "sqlboiler"); err == nil {
		t.Fatal("expected an error for an unsupported ORM")
	}
}

func TestRoutesFile_NoDoubleAPIPrefix(t *testing.T) {
	t.Parallel()

	// Frameworks whose main.go already groups under /api/v1 must not repeat it.
	for _, fw := range []string{"fiber", "gin", "echo"} {
		content, err := templates.RoutesFile("myapp", fw)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(content, "/api/v1") {
			t.Errorf("%s routes should not re-add the /api/v1 prefix:\n%s", fw, content)
		}
	}

	// Chi, Iris and stdlib receive the bare router/mux, so they own the prefix.
	for _, fw := range []string{"chi", "iris", "stdlib"} {
		content, err := templates.RoutesFile("myapp", fw)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(content, "/api/v1") {
			t.Errorf("%s routes should declare the /api/v1 prefix:\n%s", fw, content)
		}
	}
}

func TestRoutesFile_EchoAcceptsGroup(t *testing.T) {
	t.Parallel()

	content, err := templates.RoutesFile("myapp", "echo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "func SetupRoutes(api *echo.Group") {
		t.Error("echo routes should accept *echo.Group")
	}
}

func TestHandlerFile_AllFrameworks(t *testing.T) {
	t.Parallel()

	for _, fw := range []string{"fiber", "gin", "echo", "chi", "iris"} {
		content, err := templates.HandlerFile("myapp", fw)
		if err != nil {
			t.Fatalf("%s: %v", fw, err)
		}
		if !strings.Contains(content, "NewHandler(s messageService)") {
			t.Errorf("%s handler missing NewHandler constructor", fw)
		}
		// Handlers must not leak internal error detail to clients.
		if strings.Contains(content, "err.Error()") {
			t.Errorf("%s handler should not echo raw error text to the client", fw)
		}
		if !strings.Contains(content, "GetMessage(") {
			t.Errorf("%s handler should call the service", fw)
		}
	}
}

func TestHandlerFile_RejectsUnknownFramework(t *testing.T) {
	t.Parallel()

	if _, err := templates.HandlerFile("myapp", "beego"); err == nil {
		t.Fatal("expected an error for an unsupported framework")
	}
}

func TestMainFile_InjectsDependencies(t *testing.T) {
	t.Parallel()

	for _, fw := range []string{"fiber", "gin", "echo", "chi", "iris"} {
		content, err := templates.MainFile("myapp", fw)
		if err != nil {
			t.Fatalf("%s: %v", fw, err)
		}
		for _, want := range []string{
			"store := message.NewStore(config.DB)",
			"service := message.NewService(store)",
			"handler := message.NewHandler(service)",
			"if err := config.Connect(); err != nil",
			"defer config.Close()",
		} {
			if !strings.Contains(content, want) {
				t.Errorf("%s main.go missing %q", fw, want)
			}
		}
		if strings.Contains(content, "{{") {
			t.Errorf("%s main.go has an unrendered template action:\n%s", fw, content)
		}
	}
}

func TestDatabaseFile_PoolingAndErrorReturn(t *testing.T) {
	t.Parallel()

	for _, db := range []string{"postgres", "mysql", "sqlite"} {
		for _, orm := range []string{"none", "gorm", "xorm", "ent"} {
			content, err := templates.DatabaseFile(db, orm)
			if err != nil {
				t.Fatalf("%s/%s: %v", db, orm, err)
			}
			for _, want := range []string{
				"func Connect() error",
				"func Close()",
				"SetMaxOpenConns(",
				"SetConnMaxLifetime(",
			} {
				if !strings.Contains(content, want) {
					t.Errorf("%s/%s missing %q", db, orm, want)
				}
			}
			if strings.Contains(content, "log.Fatal") {
				t.Errorf("%s/%s should return errors rather than calling log.Fatal", db, orm)
			}
		}
	}
}

func TestDatabaseFile_RejectsUnknownDatabase(t *testing.T) {
	t.Parallel()

	if _, err := templates.DatabaseFile("mongodb", "none"); err == nil {
		t.Fatal("expected an error for an unsupported database")
	}
}

// The stdlib framework exists so a generated project can run on net/http alone.
// These assertions pin the two properties that make that true.
func TestStdlibTemplates_NoThirdPartyRouter(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		render func() (string, error)
	}{
		{"main", func() (string, error) { return templates.MainFile("myapp", "stdlib") }},
		{"routes", func() (string, error) { return templates.RoutesFile("myapp", "stdlib") }},
		{"handler", func() (string, error) { return templates.HandlerFile("myapp", "stdlib") }},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			content, err := tc.render()
			if err != nil {
				t.Fatal(err)
			}
			for _, router := range []string{
				"github.com/go-chi/chi",
				"github.com/gin-gonic/gin",
				"github.com/labstack/echo",
				"github.com/gofiber/fiber",
				"github.com/kataras/iris",
			} {
				if strings.Contains(content, router) {
					t.Errorf("stdlib %s imports %s; it must depend on net/http only:\n%s",
						tc.name, router, content)
				}
			}
			if !strings.Contains(content, "net/http") {
				t.Errorf("stdlib %s should use net/http:\n%s", tc.name, content)
			}
		})
	}
}

func TestStdlibRoutes_UsesMethodPattern(t *testing.T) {
	t.Parallel()

	content, err := templates.RoutesFile("myapp", "stdlib")
	if err != nil {
		t.Fatal(err)
	}

	// Go 1.22 method patterns are the whole reason no router is needed. Without
	// the "GET " prefix the route would match every method.
	if !strings.Contains(content, `"GET /api/v1/message"`) {
		t.Errorf("stdlib routes should register a GET method pattern:\n%s", content)
	}
	if !strings.Contains(content, "*http.ServeMux") {
		t.Errorf("stdlib routes should take an *http.ServeMux:\n%s", content)
	}
}

func TestStdlibMain_SetsServerTimeouts(t *testing.T) {
	t.Parallel()

	content, err := templates.MainFile("myapp", "stdlib")
	if err != nil {
		t.Fatal(err)
	}

	// http.ListenAndServe has no timeouts, so an idle client can hold a
	// connection open indefinitely. The generated server must set them.
	if strings.Contains(content, "http.ListenAndServe(") {
		t.Errorf("stdlib main should use an explicit http.Server, not http.ListenAndServe:\n%s", content)
	}
	for _, field := range []string{"ReadHeaderTimeout", "ReadTimeout", "WriteTimeout", "IdleTimeout"} {
		if !strings.Contains(content, field) {
			t.Errorf("stdlib main should set %s:\n%s", field, content)
		}
	}
}

// TestTemplates_RenderGofmtCleanGo renders the feature package for every
// framework and ORM and requires each file to be gofmt-clean Go, so a stray
// blank line or dangling import in a template fails here rather than in a
// user's editor.
func TestTemplates_RenderGofmtCleanGo(t *testing.T) {
	t.Parallel()

	for _, fw := range []string{"fiber", "gin", "echo", "chi", "iris", "stdlib"} {
		files := map[string]func() (string, error){
			"handler": func() (string, error) { return templates.HandlerFile("myapp", fw) },
			"routes":  func() (string, error) { return templates.RoutesFile("myapp", fw) },
			"service": func() (string, error) { return templates.ServiceFile("myapp") },
		}
		for _, orm := range []string{"none", "gorm", "xorm", "ent"} {
			files["store-"+orm] = func() (string, error) { return templates.StoreFile("myapp", orm) }
		}
		for name, renderFile := range files {
			content, err := renderFile()
			if err != nil {
				t.Fatalf("%s/%s: %v", fw, name, err)
			}
			formatted, err := format.Source([]byte(content))
			if err != nil {
				t.Errorf("%s/%s is not valid Go: %v\n%s", fw, name, err, content)
				continue
			}
			if string(formatted) != content {
				t.Errorf("%s/%s is not gofmt-clean:\n%s", fw, name, content)
			}
		}
	}
}

// TestFeaturePackage_SinglePackage pins the shape of the generated code: one
// package per feature, a Store for data access, and no by-layer packages.
func TestFeaturePackage_SinglePackage(t *testing.T) {
	t.Parallel()

	for _, fw := range []string{"fiber", "gin", "echo", "chi", "iris", "stdlib"} {
		mainFile, err := templates.MainFile("myapp", fw)
		if err != nil {
			t.Fatalf("%s: %v", fw, err)
		}
		for _, want := range []string{
			`"myapp/internal/message"`,
			"store := message.NewStore(config.DB)",
			"service := message.NewService(store)",
			"handler := message.NewHandler(service)",
			"message.SetupRoutes(",
		} {
			if !strings.Contains(mainFile, want) {
				t.Errorf("%s main.go missing %q", fw, want)
			}
		}
		for _, unwanted := range []string{"internal/handlers", "internal/services", "internal/repositories", "internal/routes"} {
			if strings.Contains(mainFile, unwanted) {
				t.Errorf("%s main.go should not import %q", fw, unwanted)
			}
		}

		routes, err := templates.RoutesFile("myapp", fw)
		if err != nil {
			t.Fatalf("%s: %v", fw, err)
		}
		if !strings.HasPrefix(routes, "package message\n") || strings.Contains(routes, "handlers.") {
			t.Errorf("%s routes.go should be in package message with no handlers import:\n%s", fw, routes)
		}
	}

	for _, orm := range []string{"none", "gorm", "xorm", "ent"} {
		store, err := templates.StoreFile("myapp", orm)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(store, "type Store struct") || strings.Contains(store, "Repository") {
			t.Errorf("%s: expected a Store and no Repository:\n%s", orm, store)
		}
	}
}
