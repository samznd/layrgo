package cli

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

// Set with -ldflags by the Makefile and GoReleaser. go install passes none,
// so resolveVersion falls back to the build info.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Defaults when the linker didn't set anything.
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

// resolveVersion picks what `layrgo version` prints. ldflags win. Otherwise
// each field falls back to build info on its own: a go install build knows the
// module version but has no vcs info, and a go build from a checkout is the
// other way round.
func resolveVersion(ldVersion, ldCommit, ldDate string, bi *debug.BuildInfo) (version, commit, date string) {
	version, commit, date = ldVersion, ldCommit, ldDate

	if bi == nil {
		return normalizeVersion(version), commit, date
	}

	if version == unsetVersion {
		// "(devel)" tells us nothing more than "dev" does.
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

// formatVersion leaves out the (commit, built) part when both are unknown.
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

// normalizeVersion adds the leading "v". GoReleaser stamps "1.1.0", tags are
// "v1.1.0".
func normalizeVersion(version string) string {
	if version == "" {
		return unsetVersion
	}
	if strings.HasPrefix(version, "v") || version[0] < '0' || version[0] > '9' {
		return version
	}
	return "v" + version
}
