package textdiff

import "testing"

func TestUnified(t *testing.T) {
	t.Parallel()

	a := "a\nb\nc\nd\ne\nf\ng\nh\n"
	b := "a\nb\nc\nd\nNEW\ne\nf\ng\nh\n"
	got := Unified("x.go", a, b)
	want := "--- a/x.go\n+++ b/x.go\n@@ -2,6 +2,7 @@\n b\n c\n d\n+NEW\n e\n f\n g\n"
	if got != want {
		t.Errorf("diff =\n%s\nwant\n%s", got, want)
	}
	if Unified("x.go", a, a) != "" {
		t.Error("equal inputs should give an empty diff")
	}
}

// A changed line shows the old line first, as diff(1) does.
func TestUnified_ChangedLineOldFirst(t *testing.T) {
	t.Parallel()

	got := Unified("x.go", "a\nold\nb\n", "a\nnew\nb\n")
	want := "--- a/x.go\n+++ b/x.go\n@@ -1,3 +1,3 @@\n a\n-old\n+new\n b\n"
	if got != want {
		t.Errorf("diff =\n%s\nwant\n%s", got, want)
	}
}
