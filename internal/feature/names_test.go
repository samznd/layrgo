package feature

import (
	"strings"
	"testing"
)

func TestPluralize(t *testing.T) {
	t.Parallel()

	for in, want := range map[string]string{
		"order":   "orders",
		"bus":     "buses",
		"box":     "boxes",
		"quiz":    "quizes",
		"match":   "matches",
		"wish":    "wishes",
		"city":    "cities",
		"day":     "days",
		"key":     "keys",
		"invoice": "invoices",
		"y":       "ys",
	} {
		if got := Pluralize(in); got != want {
			t.Errorf("Pluralize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNewNames(t *testing.T) {
	t.Parallel()

	n, err := NewNames("order", "")
	if err != nil {
		t.Fatal(err)
	}
	if n != (Names{Package: "order", Type: "Order", Plural: "orders"}) {
		t.Errorf("NewNames(order) = %+v", n)
	}

	n, err = NewNames("person", "people")
	if err != nil {
		t.Fatal(err)
	}
	if n.Plural != "people" {
		t.Errorf("--plural ignored: %+v", n)
	}
}

func TestNewNames_Rejects(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ name, plural, wantErr string }{
		{"", "", "required"},
		{"Order", "", "lowercase"},
		{"order_item", "", "lowercase"},
		{"1order", "", "lowercase"},
		{"type", "", "reserved"},
		{"string", "", "reserved"},
		{"config", "", "reserved"},
		{"http", "", "reserved"},
		{"handler", "", "reserved"},
		{"order", "Orders", "--plural"},
		{"sheep", "sheep", "must differ"},
		{"group", "", "reserved word in SQL"},
	} {
		_, err := NewNames(tc.name, tc.plural)
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("NewNames(%q, %q) error = %v, want it to mention %q", tc.name, tc.plural, err, tc.wantErr)
		}
	}
}
