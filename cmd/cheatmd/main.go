// Command cheatmd is an executable Markdown cheatsheet tool. It parses Markdown
// cheatsheets, lets the user browse and fill in command templates interactively,
// and prints, copies, or executes the result. See package internal/ui for the
// interactive flow and pkg/parser for the cheatsheet format.
package main

import (
	"os"
	"runtime/debug"
)

// version is the build version. Release binaries set it via
// -ldflags "-X main.version=<tag>". When unset, resolveVersion falls back to
// the module version embedded in the binary: a clean tag for
// `go install module@version`, or a VCS pseudo-version (e.g.
// v1.0.0-rc.2.0.<timestamp>-<hash>+dirty) for local builds from a checkout.
var version = "dev"

func main() {
	rootCmd.Version = resolveVersion()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// resolveVersion returns the ldflag-injected version when present, otherwise
// the module version embedded in the binary (a clean tag for
// `go install module@version`, or a VCS pseudo-version for local builds),
// falling back to "dev" when no build info is available.
func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v := bi.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return version
}
