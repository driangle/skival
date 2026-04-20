package cmd

import "fmt"

// Version is the skival release version. Updated in-place by the release
// script before tagging (see .release.conf).
var Version = "0.0.0"

// commit is the git commit hash the binary was built from.
// Injected via -ldflags at build time; falls back to "unknown" otherwise.
var commit = "unknown"

// suffix is a build-type label (e.g. "dev", "release").
// Injected via -ldflags at build time; "dev" by default for local builds.
var suffix = "dev"

const shortCommitLen = 7

func versionString() string {
	c := commit
	if len(c) > shortCommitLen {
		c = c[:shortCommitLen]
	}
	return fmt.Sprintf("%s (commit %s, %s)", Version, c, suffix)
}

func init() {
	rootCmd.Version = versionString()
}
