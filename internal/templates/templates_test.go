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

// The xorm store must check the row count and return both errors.
func TestStoreFile_XORM_ErrorPaths(t *testing.T) {
	t.Parallel()

	content, err := templates.StoreFile("myapp", "xorm")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(content, `return "", fmt.Errorf("get message: %w", err)`) {
		t.Error("xorm repository should propagate the query error")
	}
	if !strings.Contains(content, "if !has {") {
		t.Error("xorm repository should guard against a missing row")
	}
	if !strings.Contains(content, `return "", fmt.Errorf("get message: no rows")`) {
		t.Error("xorm repository should return an error for a missing row")
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

	for _, fw := range []string{"fiber", "gin", "echo", "chi", "iris", "stdlib"} {
		content, err := templates.MainFile("myapp", fw)
		if err != nil {
			t.Fatalf("%s: %v", fw, err)
		}
		for _, want := range []string{
			"messageStore := message.NewStore(config.DB)",
			"if err := messageStore.Migrate(context.Background()); err != nil",
			"messageService := message.NewService(messageStore)",
			"messageHandler := message.NewHandler(messageService)",
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

// stdlib projects must run on net/http alone.
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

	// Without the "GET " prefix the route would match every method.
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

	// The generated server must set timeouts; http.ListenAndServe has none.
	if strings.Contains(content, "http.ListenAndServe(") {
		t.Errorf("stdlib main should use an explicit http.Server, not http.ListenAndServe:\n%s", content)
	}
	for _, field := range []string{"ReadHeaderTimeout", "ReadTimeout", "WriteTimeout", "IdleTimeout"} {
		if !strings.Contains(content, field) {
			t.Errorf("stdlib main should set %s:\n%s", field, content)
		}
	}
}

// Every generated file must be gofmt-clean Go.
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

// One package per feature, a Store, and no by-layer packages.
func TestFeaturePackage_SinglePackage(t *testing.T) {
	t.Parallel()

	for _, fw := range []string{"fiber", "gin", "echo", "chi", "iris", "stdlib"} {
		mainFile, err := templates.MainFile("myapp", fw)
		if err != nil {
			t.Fatalf("%s: %v", fw, err)
		}
		for _, want := range []string{
			`"myapp/internal/message"`,
			"messageStore := message.NewStore(config.DB)",
			"messageService := message.NewService(messageStore)",
			"messageHandler := message.NewHandler(messageService)",
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

// AGENTS.md must render for every stack, with nothing left unrendered.
func TestAgentsFile_AllStacks(t *testing.T) {
	t.Parallel()

	for _, fw := range []string{"Fiber", "Gin", "Echo", "Chi", "Iris", "stdlib"} {
		for _, db := range []string{"Postgres", "MySQL", "SQLite"} {
			for _, orm := range []string{"none", "GORM", "XORM", "Ent", ""} {
				content, err := templates.AgentsFile("myapp", fw, db, orm)
				if err != nil {
					t.Fatalf("%s/%s/%s: %v", fw, db, orm, err)
				}
				for _, bad := range []string{"{{", "}}", "<no value>"} {
					if strings.Contains(content, bad) {
						t.Errorf("%s/%s/%s: AGENTS.md contains %q", fw, db, orm, bad)
					}
				}
				for _, want := range []string{"Module: `myapp`", "go test ./...", "bookmarkStore := bookmark.NewStore(config.DB)"} {
					if !strings.Contains(content, want) {
						t.Errorf("%s/%s/%s: AGENTS.md missing %q", fw, db, orm, want)
					}
				}
			}
		}
	}
}

// The instructions an agent most often gets wrong differ by stack.
func TestAgentsFile_StackSpecificGuidance(t *testing.T) {
	t.Parallel()

	cases := []struct {
		fw, db, orm string
		want        []string
		notWant     []string
	}{
		{"chi", "postgres", "none", []string{"Do not** call `r.Route(\"/api/v1\"", "bookmark.SetupRoutes(r, bookmarkHandler)", "`$1, $2, ...`", "sql.ErrNoRows"}, []string{"AutoMigrate"}},
		{"gin", "sqlite", "gorm", []string{"**without** that prefix", "bookmark.SetupRoutes(api, bookmarkHandler)", "AutoMigrate", "gorm.ErrRecordNotFound"}, []string{"`$1"}},
		{"stdlib", "mysql", "xorm", []string{"r.PathValue(\"id\")", "bookmark.SetupRoutes(mux, bookmarkHandler)", "s.db.Sync"}, []string{"AutoMigrate"}},
		{"fiber", "mysql", "none", []string{"Fiber **v3**", "Placeholders for MySQL", "`?`"}, nil},
		{"iris", "sqlite", "ent", []string{"app.Party(\"/api/v1/bookmarks\")", "Ent was selected", "ent new"}, nil},
	}
	for _, c := range cases {
		content, err := templates.AgentsFile("myapp", c.fw, c.db, c.orm)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range c.want {
			if !strings.Contains(content, want) {
				t.Errorf("%s/%s/%s: missing %q", c.fw, c.db, c.orm, want)
			}
		}
		for _, bad := range c.notWant {
			if strings.Contains(content, bad) {
				t.Errorf("%s/%s/%s: should not contain %q", c.fw, c.db, c.orm, bad)
			}
		}
	}
}

func TestClaudeFile_ImportsAgents(t *testing.T) {
	t.Parallel()

	if got := templates.ClaudeFile(); got != "@AGENTS.md\n" {
		t.Errorf("CLAUDE.md = %q, want it to import AGENTS.md", got)
	}
}

// Tools that look for their own file are pointed back to AGENTS.md.
func TestAgentPointerFiles(t *testing.T) {
	t.Parallel()

	if got := templates.GeminiFile(); got != "@./AGENTS.md\n" {
		t.Errorf("GEMINI.md = %q, want it to import AGENTS.md", got)
	}
	copilot := templates.CopilotFile()
	for _, want := range []string{"[AGENTS.md](../AGENTS.md)", "handler → service → store", "go test ./..."} {
		if !strings.Contains(copilot, want) {
			t.Errorf("copilot-instructions.md missing %q", want)
		}
	}
}

// Agents copy the sample feature, so it ships with a test for them to copy.
func TestServiceTestFile(t *testing.T) {
	t.Parallel()

	content, err := templates.ServiceTestFile("myapp")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"package message", "type fakeStore struct", "func TestService_GetMessage(", "NewService(fakeStore{"} {
		if !strings.Contains(content, want) {
			t.Errorf("service_test.go missing %q", want)
		}
	}
}

// Agents copy the sample store, so every variant shows how a feature creates
// its own table at startup.
func TestStoreFile_HasMigrate(t *testing.T) {
	t.Parallel()

	for _, orm := range []string{"none", "gorm", "xorm", "ent"} {
		content, err := templates.StoreFile("myapp", orm)
		if err != nil {
			t.Fatalf("%s: %v", orm, err)
		}
		for _, want := range []string{"func (s *Store) Migrate(ctx context.Context) error", "data from the store"} {
			if !strings.Contains(content, want) {
				t.Errorf("%s store missing %q", orm, want)
			}
		}
	}
}
