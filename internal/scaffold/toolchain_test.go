package scaffold

import "testing"

func TestCheckFrameworkToolchain(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		framework string
		goVersion string
		wantErr   bool
	}{
		// net/http method patterns land in Go 1.22.
		{"stdlib on go1.21 is refused", "stdlib", "go1.21.13", true},
		{"stdlib on go1.20 is refused", "stdlib", "go1.20", true},
		{"stdlib on go1.22 is allowed", "stdlib", "go1.22.0", false},
		{"stdlib on go1.23 is allowed", "stdlib", "go1.23.4", false},
		{"stdlib on go1.25 is allowed", "stdlib", "go1.25.1", false},
		{"stdlib is matched case-insensitively", "StdLib", "go1.21.0", true},

		// Other frameworks bring their own router, so the Go version doesn't matter.
		{"chi on go1.21 is allowed", "Chi", "go1.21.0", false},
		{"gin on go1.20 is allowed", "Gin", "go1.20", false},

		// Unparseable versions are let through.
		{"devel toolchain is not blocked", "stdlib", "devel +abc1234", false},
		{"empty version is not blocked", "stdlib", "", false},
		{"go2 is not blocked", "stdlib", "go2.0.0", false},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := checkFrameworkToolchain(tc.framework, tc.goVersion)
			if tc.wantErr && err == nil {
				t.Errorf("checkFrameworkToolchain(%q, %q) = nil, want an error",
					tc.framework, tc.goVersion)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("checkFrameworkToolchain(%q, %q) = %v, want nil",
					tc.framework, tc.goVersion, err)
			}
		})
	}
}

func TestGoMinor(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		in     string
		want   int
		wantOK bool
	}{
		{"go1.23.4", 23, true},
		{"go1.22", 22, true},
		{"go1.21.13", 21, true},
		{"go1.25rc1", 25, true},
		{"go2.0.0", 0, false},
		{"devel +abc", 0, false},
		{"", 0, false},
		{"go1.", 0, false},
	} {
		got, ok := goMinor(tc.in)
		if ok != tc.wantOK || got != tc.want {
			t.Errorf("goMinor(%q) = (%d, %t), want (%d, %t)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}
