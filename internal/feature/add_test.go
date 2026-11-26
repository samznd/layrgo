package feature_test

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samznd/layrgo/internal/feature"
	"github.com/samznd/layrgo/internal/project"
	"github.com/samznd/layrgo/internal/scaffold"
)

func TestMain(m *testing.M) {
	restore := scaffold.SetProgressOutput(io.Discard)
	code := m.Run()
	restore()
	os.Exit(code)
}

// newProject writes a Layrgo project without installing dependencies, plus a
// go.mod like the one "layrgo init" would leave.
func newProject(t *testing.T, framework, database, orm string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "shop")
	if err := scaffold.WriteFiles(scaffold.Config{
		ProjectName: "shop", ProjectPath: root,
		Framework: framework, Database: database, ORM: orm,
	}); err != nil {
		t.Fatal(err)
	}
	gomod := "module shop\n\ngo 1.23\n\nrequire (\n\tgithub.com/gin-gonic/gin v1.10.1\n\tgithub.com/lib/pq v1.10.9\n)\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func add(t *testing.T, root string, mod func(*feature.Options)) (*feature.Result, error) {
	t.Helper()
	opts := feature.Options{Root: root, Name: "order", Fields: "customer:string,total:int", SkipChecks: true}
	if mod != nil {
		mod(&opts)
	}
	return feature.Add(opts)
}

func exitCode(err error) int {
	var ferr *feature.Error
	if errors.As(err, &ferr) {
		return ferr.Code
	}
	return -1
}

func TestAdd_WritesAndWires(t *testing.T) {
	t.Parallel()
	root := newProject(t, "Gin", "Postgres", "none")

	res, err := add(t, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Wired || len(res.Files) != 6 {
		t.Errorf("result = %+v", res)
	}
	for _, f := range res.Files {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Errorf("missing %s", f)
		}
	}
	main, _ := os.ReadFile(filepath.Join(root, "cmd", "main.go"))
	if !strings.Contains(string(main), "order.SetupRoutes(api, orderHandler)") {
		t.Errorf("main.go not wired:\n%s", main)
	}
	if len(res.Todos) == 0 || !strings.HasPrefix(res.Todos[0], "internal/order/service.go:") {
		t.Errorf("todos = %v", res.Todos)
	}
	if res.StackFileWritten {
		t.Error(".layrgo.json exists, so it should not be rewritten")
	}
}

func TestAdd_ExistingFeature(t *testing.T) {
	t.Parallel()
	root := newProject(t, "Chi", "SQLite", "none")

	if _, err := add(t, root, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := add(t, root, nil); exitCode(err) != feature.ExitExists {
		t.Fatalf("second add: err = %v, want exit %d", err, feature.ExitExists)
	}

	// --force replaces the package and does not wire it twice.
	res, err := add(t, root, func(o *feature.Options) { o.Force = true; o.Fields = "customer:string" })
	if err != nil {
		t.Fatal(err)
	}
	if !res.AlreadyWired {
		t.Error("--force rerun should report the feature as already wired")
	}
	main, _ := os.ReadFile(filepath.Join(root, "cmd", "main.go"))
	if n := strings.Count(string(main), "order.SetupRoutes("); n != 1 {
		t.Errorf("order.SetupRoutes appears %d times", n)
	}
	model, _ := os.ReadFile(filepath.Join(root, "internal", "order", "model.go"))
	if strings.Contains(string(model), "Total") {
		t.Error("--force kept the old model")
	}
}

func TestAdd_DryRunWritesNothing(t *testing.T) {
	t.Parallel()
	root := newProject(t, "Echo", "MySQL", "GORM")
	before, _ := os.ReadFile(filepath.Join(root, "cmd", "main.go"))

	res, err := add(t, root, func(o *feature.Options) { o.DryRun = true })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Diff, "+\torder.SetupRoutes(api, orderHandler)") {
		t.Errorf("diff misses the routes line:\n%s", res.Diff)
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "order")); !os.IsNotExist(err) {
		t.Error("dry run created internal/order")
	}
	after, _ := os.ReadFile(filepath.Join(root, "cmd", "main.go"))
	if string(before) != string(after) {
		t.Error("dry run changed cmd/main.go")
	}
}

func TestAdd_InvalidInputWritesNothing(t *testing.T) {
	t.Parallel()
	root := newProject(t, "Gin", "Postgres", "none")

	for _, mod := range []func(*feature.Options){
		func(o *feature.Options) { o.Name = "type" },
		func(o *feature.Options) { o.Fields = "total:money" },
		func(o *feature.Options) { o.Root = t.TempDir() }, // not a project
	} {
		if _, err := add(t, root, mod); exitCode(err) != feature.ExitInvalid {
			t.Errorf("err = %v, want exit %d", err, feature.ExitInvalid)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "order")); !os.IsNotExist(err) {
		t.Error("invalid input created internal/order")
	}
}

// Projects made before v1.5.0 have no .layrgo.json: the stack comes from
// go.mod and is recorded for next time.
func TestAdd_DetectsStackFromGoMod(t *testing.T) {
	t.Parallel()
	root := newProject(t, "Gin", "Postgres", "none")
	if err := os.Remove(filepath.Join(root, project.StackFile)); err != nil {
		t.Fatal(err)
	}

	res, err := add(t, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.StackFileWritten {
		t.Error("the detected stack was not recorded")
	}
	b, err := os.ReadFile(filepath.Join(root, project.StackFile))
	if err != nil || !strings.Contains(string(b), `"framework": "gin"`) {
		t.Errorf(".layrgo.json = %s, %v", b, err)
	}
}

func TestAdd_UnwirableMainExits3(t *testing.T) {
	t.Parallel()
	root := newProject(t, "Gin", "Postgres", "none")
	hand := "package main\n\nimport (\n\t\"log\"\n)\n\nfunc main() {\n\tlog.Println(\"mine\")\n}\n"
	mainPath := filepath.Join(root, "cmd", "main.go")
	if err := os.WriteFile(mainPath, []byte(hand), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := add(t, root, nil)
	if exitCode(err) != feature.ExitNotWired {
		t.Fatalf("err = %v, want exit %d", err, feature.ExitNotWired)
	}
	if res.ManualWiring == "" {
		t.Error("no lines to paste")
	}
	if after, _ := os.ReadFile(mainPath); string(after) != hand {
		t.Error("cmd/main.go was changed")
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "order", "store.go")); err != nil {
		t.Error("the feature files should still be written")
	}
}

// The stdlib + database/sql feature imports only the standard library, so it
// can be compiled and its generated tests run without downloading anything.
func TestAdd_GeneratedPackageCompilesAndPasses(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	t.Parallel()

	for _, db := range []string{"Postgres", "MySQL", "SQLite"} {
		root := newProject(t, "stdlib", db, "none")
		if _, err := add(t, root, func(o *feature.Options) {
			o.Fields = "customer:string,note:text?,total:int,price:float,paid:bool,due_at:time,shipped_at:time?"
		}); err != nil {
			t.Fatal(err)
		}
		// go.mod lists gin and pq, which are not downloaded; test the package alone.
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module shop\n\ngo 1.23\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		for _, args := range [][]string{{"vet", "./internal/order/"}, {"test", "./internal/order/"}} {
			cmd := exec.Command("go", args...)
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOPROXY=off")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("%s: go %s: %v\n%s", db, strings.Join(args, " "), err, out)
			}
		}
	}
}
