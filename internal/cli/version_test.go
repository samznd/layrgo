package cli

import (
	"runtime/debug"
	"testing"
)

// buildInfo builds a *debug.BuildInfo with the main module version and the VCS
// settings the Go toolchain stamps into binaries produced by `go install`.
func buildInfo(mainVersion, revision, time string) *debug.BuildInfo {
	bi := &debug.BuildInfo{}
	bi.Main.Version = mainVersion
	if revision != "" {
		bi.Settings = append(bi.Settings, debug.BuildSetting{Key: "vcs.revision", Value: revision})
	}
	if time != "" {
		bi.Settings = append(bi.Settings, debug.BuildSetting{Key: "vcs.time", Value: time})
	}
	return bi
}

// moduleZipBuildInfo is what `go install <module>@<version>` produces: a module
// version and no vcs settings.
func moduleZipBuildInfo(mainVersion string) *debug.BuildInfo {
	bi := &debug.BuildInfo{}
	bi.Main.Version = mainVersion
	return bi
}

func TestResolveVersion_LdflagsWin(t *testing.T) {
	t.Parallel()

	// Makefile and GoReleaser builds: ldflags win and build info is ignored.
	bi := buildInfo("v1.1.0", "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "2026-01-01T00:00:00Z")

	version, commit, date := resolveVersion("v1.2.0-3-gabc1234", "abc1234", "2026-11-09T17:19:53Z", bi)

	if version != "v1.2.0-3-gabc1234" {
		t.Errorf("version = %q, want the linker value v1.2.0-3-gabc1234", version)
	}
	if commit != "abc1234" {
		t.Errorf("commit = %q, want the linker value abc1234", commit)
	}
	if date != "2026-11-09T17:19:53Z" {
		t.Errorf("date = %q, want the linker value 2026-11-09T17:19:53Z", date)
	}
}

func TestResolveVersion_GoInstallModuleZip(t *testing.T) {
	t.Parallel()

	// go install ...@v1.1.0: no ldflags and no vcs info, only the module version.
	version, commit, date := resolveVersion("dev", "none", "unknown", moduleZipBuildInfo("v1.1.0"))

	if version != "v1.1.0" {
		t.Errorf("version = %q, want v1.1.0 from the module version", version)
	}
	if commit != "none" || date != "unknown" {
		t.Errorf("got commit %q and date %q, want the sentinels: a module zip records no vcs data", commit, date)
	}
}

func TestResolveVersion_GoInstallVCSFallback(t *testing.T) {
	t.Parallel()

	// Both signals present: every field comes from build info.
	bi := buildInfo("v1.1.0", "dee514b0000000000000000000000000000000ab", "2026-11-09T17:19:53Z")

	version, commit, date := resolveVersion("dev", "none", "unknown", bi)

	if version != "v1.1.0" {
		t.Errorf("version = %q, want v1.1.0 from the module version", version)
	}
	if commit != "dee514b0000000000000000000000000000000ab" {
		t.Errorf("commit = %q, want the vcs.revision build setting", commit)
	}
	if date != "2026-11-09T17:19:53Z" {
		t.Errorf("date = %q, want the vcs.time build setting", date)
	}
}

func TestResolveVersion_MarksDirtyWorkingTree(t *testing.T) {
	t.Parallel()

	bi := buildInfo("(devel)", "cafebabecafebabecafebabecafebabecafebabe", "2026-11-09T12:00:00Z")
	bi.Settings = append(bi.Settings, debug.BuildSetting{Key: "vcs.modified", Value: "true"})

	_, commit, _ := resolveVersion("dev", "none", "unknown", bi)

	if commit != "cafebabecafebabecafebabecafebabecafebabe-dirty" {
		t.Errorf("commit = %q, want the revision marked -dirty for a modified tree", commit)
	}
}

func TestResolveVersion_LocalDevelBuildKeepsDev(t *testing.T) {
	t.Parallel()

	// go build from a checkout: version is "(devel)", commit and date are real.
	bi := buildInfo("(devel)", "cafebabecafebabecafebabecafebabecafebabe", "2026-11-09T12:00:00Z")

	version, commit, date := resolveVersion("dev", "none", "unknown", bi)

	if version != "dev" {
		t.Errorf("version = %q, want dev for a (devel) build", version)
	}
	if commit != "cafebabecafebabecafebabecafebabecafebabe" {
		t.Errorf("commit = %q, want the vcs.revision build setting", commit)
	}
	if date != "2026-11-09T12:00:00Z" {
		t.Errorf("date = %q, want the vcs.time build setting", date)
	}
}

func TestResolveVersion_NoBuildInfoKeepsSentinels(t *testing.T) {
	t.Parallel()

	version, commit, date := resolveVersion("dev", "none", "unknown", nil)

	if version != "dev" || commit != "none" || date != "unknown" {
		t.Errorf("got (%q, %q, %q), want the sentinels (dev, none, unknown)", version, commit, date)
	}
}

func TestResolveVersion_MissingVCSSettingsKeepSentinels(t *testing.T) {
	t.Parallel()

	// Build info without VCS data, e.g. built outside git or with -buildvcs=false.
	version, commit, date := resolveVersion("dev", "none", "unknown", buildInfo("v1.1.0", "", ""))

	if version != "v1.1.0" {
		t.Errorf("version = %q, want v1.1.0 from the module version", version)
	}
	if commit != "none" {
		t.Errorf("commit = %q, want the sentinel none when vcs.revision is absent", commit)
	}
	if date != "unknown" {
		t.Errorf("date = %q, want the sentinel unknown when vcs.time is absent", date)
	}
}

func TestResolveVersion_NormalizesGoReleaserVersion(t *testing.T) {
	t.Parallel()

	// GoReleaser stamps the version without the leading "v".
	version, _, _ := resolveVersion("1.1.0", "dee514b", "2026-11-09T17:19:53Z", nil)

	if version != "v1.1.0" {
		t.Errorf("version = %q, want the GoReleaser 1.1.0 normalized to v1.1.0", version)
	}
}

func TestNormalizeVersion(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ in, want string }{
		{"1.1.0", "v1.1.0"},
		{"v1.1.0", "v1.1.0"},
		{"1.2.0-3-gabc1234", "v1.2.0-3-gabc1234"},
		{"v1.2.0-3-gabc1234-dirty", "v1.2.0-3-gabc1234-dirty"},
		{"dev", "dev"},
		{"", "dev"},
	} {
		if got := normalizeVersion(tc.in); got != tc.want {
			t.Errorf("normalizeVersion(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatVersion(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name                  string
		version, commit, date string
		want                  string
	}{
		{
			name:    "full detail from ldflags",
			version: "v1.1.0", commit: "dee514b", date: "2026-11-09T17:19:53Z",
			want: "layrgo v1.1.0 (commit: dee514b, built: 2026-11-09T17:19:53Z)",
		},
		{
			// go install build: a version and nothing else.
			name:    "module version only omits the parenthetical",
			version: "v1.1.0", commit: "none", date: "unknown",
			want: "layrgo v1.1.0",
		},
		{
			name:    "local build has vcs data but no version",
			version: "dev", commit: "cafebabe", date: "2026-11-09T12:00:00Z",
			want: "layrgo dev (commit: cafebabe, built: 2026-11-09T12:00:00Z)",
		},
		{
			name:    "commit without a date",
			version: "dev", commit: "cafebabe", date: "unknown",
			want: "layrgo dev (commit: cafebabe)",
		},
		{
			name:    "date without a commit",
			version: "dev", commit: "none", date: "2026-11-09T12:00:00Z",
			want: "layrgo dev (built: 2026-11-09T12:00:00Z)",
		},
		{
			name:    "nothing known at all",
			version: "dev", commit: "none", date: "unknown",
			want: "layrgo dev",
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := formatVersion(tc.version, tc.commit, tc.date); got != tc.want {
				t.Errorf("formatVersion(%q, %q, %q) = %q, want %q",
					tc.version, tc.commit, tc.date, got, tc.want)
			}
		})
	}
}
