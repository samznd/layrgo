package feature

import (
	"errors"
	"strings"
	"testing"

	"github.com/samznd/layrgo/internal/templates"
)

var orderNames = Names{Package: "order", Type: "Order", Plural: "orders"}

// renderMain renders a v1.5.0 cmd/main.go, with markers.
func renderMain(t *testing.T, framework string) string {
	t.Helper()
	src, err := templates.MainFile("shop", framework)
	if err != nil {
		t.Fatal(err)
	}
	return src
}

// withoutMarkers turns a v1.5.0 cmd/main.go into a v1.4.0 one.
func withoutMarkers(src string) string {
	var keep []string
	for _, l := range strings.Split(src, "\n") {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, WireMarker) || strings.HasPrefix(t, RoutesMarker) {
			continue
		}
		keep = append(keep, l)
	}
	return strings.Join(keep, "\n")
}

func TestWire_AllFrameworks(t *testing.T) {
	t.Parallel()

	for _, fw := range []string{"fiber", "gin", "echo", "chi", "iris", "stdlib"} {
		for _, version := range []string{"markers", "v1.4.0"} {
			src := renderMain(t, fw)
			if version == "v1.4.0" {
				src = withoutMarkers(src)
			}

			res, err := Wire([]byte(src), "shop", fw, orderNames)
			if err != nil {
				t.Fatalf("%s/%s: %v", fw, version, err)
			}
			out := string(res.Content)

			routerVar := templates.RouterVar(fw)
			for _, want := range []string{
				`"shop/internal/order"`,
				"orderStore := order.NewStore(config.DB)",
				"orderStore.Migrate(context.Background())",
				`log.Fatalf("migrate orders: %v", err)`,
				"orderHandler := order.NewHandler(orderService)",
				"order.SetupRoutes(" + routerVar + ", orderHandler)",
			} {
				if !strings.Contains(out, want) {
					t.Errorf("%s/%s: missing %q in:\n%s", fw, version, want, out)
				}
			}

			// The feature goes after message, and its routes after message's.
			if strings.Index(out, "orderStore :=") < strings.Index(out, "messageHandler :=") {
				t.Errorf("%s/%s: order is wired before message", fw, version)
			}
			if strings.Index(out, "order.SetupRoutes") < strings.Index(out, "message.SetupRoutes") {
				t.Errorf("%s/%s: order's routes come before message's", fw, version)
			}

			// Wiring a second feature keeps both.
			res2, err := Wire(res.Content, "shop", fw, Names{Package: "invoice", Type: "Invoice", Plural: "invoices"})
			if err != nil {
				t.Fatalf("%s/%s second feature: %v", fw, version, err)
			}
			if !strings.Contains(string(res2.Content), "invoice.SetupRoutes(") ||
				!strings.Contains(string(res2.Content), "order.SetupRoutes(") {
				t.Errorf("%s/%s: second feature lost a feature:\n%s", fw, version, res2.Content)
			}
		}
	}
}

func TestWire_AlreadyWired(t *testing.T) {
	t.Parallel()

	first, err := Wire([]byte(renderMain(t, "gin")), "shop", "gin", orderNames)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Wire(first.Content, "shop", "gin", orderNames)
	if err != nil {
		t.Fatal(err)
	}
	if !again.AlreadyWired || string(again.Content) != string(first.Content) {
		t.Errorf("rewiring changed the file or was not detected: AlreadyWired=%v", again.AlreadyWired)
	}
}

func TestWire_NothingToFollow(t *testing.T) {
	t.Parallel()

	src := `package main

import (
	"log"
)

func main() {
	log.Println("hand-written")
}
`
	res, err := Wire([]byte(src), "shop", "gin", orderNames)
	if !errors.Is(err, ErrCannotWire) {
		t.Fatalf("err = %v, want ErrCannotWire", err)
	}
	if string(res.Content) != src {
		t.Error("the file was changed")
	}
	manual := res.Wiring.Manual()
	for _, want := range []string{`"shop/internal/order"`, "orderHandler := order.NewHandler(orderService)", "order.SetupRoutes(api, orderHandler)"} {
		if !strings.Contains(manual, want) {
			t.Errorf("manual instructions miss %q:\n%s", want, manual)
		}
	}
}

func TestWire_DoesNotParse(t *testing.T) {
	t.Parallel()

	if _, err := Wire([]byte("package main\nfunc main( {"), "shop", "gin", orderNames); !errors.Is(err, ErrCannotWire) {
		t.Fatalf("err = %v, want ErrCannotWire", err)
	}
}

func TestUnifiedDiff(t *testing.T) {
	t.Parallel()

	a := "a\nb\nc\nd\ne\nf\ng\nh\n"
	b := "a\nb\nc\nd\nNEW\ne\nf\ng\nh\n"
	got := unifiedDiff("x.go", a, b)
	want := "--- a/x.go\n+++ b/x.go\n@@ -2,6 +2,7 @@\n b\n c\n d\n+NEW\n e\n f\n g\n"
	if got != want {
		t.Errorf("diff =\n%s\nwant\n%s", got, want)
	}
	if unifiedDiff("x.go", a, a) != "" {
		t.Error("equal inputs should give an empty diff")
	}
}
