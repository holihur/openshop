// Package version carries build metadata injected at release time via ldflags
// (see .goreleaser.yaml). Defaults are used for local `go build` runs.
package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the semantic version or git tag of the build.
	Version = "dev"
	// Commit is the short git SHA the binary was built from.
	Commit = "none"
	// Date is the build timestamp.
	Date = "unknown"
)

// String renders a single-line description for logs and `--version`.
func String() string {
	return fmt.Sprintf("openshop %s (commit %s, built %s, %s/%s)",
		Version, Commit, Date, runtime.GOOS, runtime.GOARCH)
}
