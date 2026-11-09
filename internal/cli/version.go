package cli

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

// Set at build time via -ldflags by the Makefile and GoReleaser. When they are
// left at these defaults the values are recovered from the build info that the
// toolchain stamps into every binary, so a `go install` build — which passes no
// linker flags — still reports something true.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Sentinel values meaning "the linker did not set this".
const (
	unsetVersion = "dev"
	unsetCommit  = "none"
	unsetDate    = "unknown"
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the layrgo version",
	Run: func(cmd *cobra.Command, args []string) {
		bi, _ := debug.ReadBuildInfo()
		version, commit, date := resolveVersion(Version, Commit, Date, bi)
		fmt.Println(formatVersion(version, commit, date))
	},
}

// resolveVersion decides what `layrgo version` reports. Linker-provided values
// always win, so the Makefile and GoReleaser behavior is unchanged; each field
// then falls back independently to build info, and keeps its sentinel when
// neither source has an answer.
//
// The two fallback sources are disjoint in practice. A module-zip build
// (`go install <module>@<version>`) knows its module version but records no
// vcs.* settings, because a module zip contains no repository. A build from a
// working tree (`go build`) has the reverse: real vcs.* settings, but the
// module version "(devel)". Each field is therefore resolved on its own rather
// than treating build info as all-or-nothing.
func resolveVersion(ldVersion, ldCommit, ldDate string, bi *debug.BuildInfo) (version, commit, date string) {
	version, commit, date = ldVersion, ldCommit, ldDate

	if bi == nil {
		return normalizeVersion(version), commit, date
	}

	if version == unsetVersion {
		// "(devel)" is what the toolchain reports for a build from a working
		// tree rather than a tagged module version; it says no more than "dev",
		// so keep the sentinel.
		if v := bi.Main.Version; v != "" && v != "(devel)" {
			version = v
		}
	}

	settings := make(map[string]string, len(bi.Settings))
	for _, s := range bi.Settings {
		settings[s.Key] = s.Value
	}

	if commit == unsetCommit {
		if rev := settings["vcs.revision"]; rev != "" {
			commit = rev
			// A working-tree build can differ from the commit it names.
			if settings["vcs.modified"] == "true" {
				commit += "-dirty"
			}
		}
	}
	if date == unsetDate {
		if t := settings["vcs.time"]; t != "" {
			date = t
		}
	}

	return normalizeVersion(version), commit, date
}

// formatVersion renders the version line, omitting the parenthetical detail
// when neither the commit nor the build date is known. A module-zip install
// resolves a real version but no vcs.* data, and printing
// "(commit: none, built: unknown)" next to a correct version is noise.
func formatVersion(version, commit, date string) string {
	line := "layrgo " + version

	var parts []string
	if commit != unsetCommit {
		parts = append(parts, "commit: "+commit)
	}
	if date != unsetDate {
		parts = append(parts, "built: "+date)
	}
	if len(parts) == 0 {
		return line
	}
	return line + " (" + strings.Join(parts, ", ") + ")"
}

// normalizeVersion gives both build paths the same shape. GoReleaser stamps
// "1.1.0" while module versions and git tags are "v1.1.0", so settle on the
// v-prefixed form and leave non-version strings such as "dev" alone.
func normalizeVersion(version string) string {
	if version == "" {
		return unsetVersion
	}
	if strings.HasPrefix(version, "v") || version[0] < '0' || version[0] > '9' {
		return version
	}
	return "v" + version
}
