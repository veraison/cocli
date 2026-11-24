// Copyright 2026 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"runtime/debug"
)

// Version can be set at link time to override the automatically derived
// version, e.g.:
//
//	go build -ldflags "-X github.com/veraison/cocli/cmd.Version=1.2.3"
var Version string

// version returns the version of the running binary.  In order of preference:
//
//  1. the value of Version, if it has been set at link time;
//  2. the version of the main module recorded by the Go toolchain.  This is the
//     tag, if the binary was built from a tagged commit (or installed with
//     "go install github.com/veraison/cocli@<tag>"), and a pseudo-version
//     derived from the latest tag and the commit otherwise, with a "+dirty"
//     suffix if the working tree had uncommitted changes;
//  3. "v0.0.0-rev.<revision>", if the toolchain recorded the vcs revision but
//     could not derive a version from it (with a "+dirty" suffix if modified);
//  4. "v0.0.0-unknown", e.g., when built outside a git checkout or under "go
//     test".
//
// The fallbacks are valid semver prereleases of v0.0.0, so they sort below any
// real release and can be parsed by tools that expect semver.
func version() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return versionFrom(Version, nil)
	}

	return versionFrom(Version, bi)
}

const unknownVersion = "v0.0.0-unknown"

func versionFrom(override string, bi *debug.BuildInfo) string {
	if override != "" {
		return override
	}

	if bi == nil {
		return unknownVersion
	}

	if v := bi.Main.Version; v != "" && v != "(devel)" {
		return v
	}

	var (
		rev      string
		modified bool
	)

	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}

	if rev == "" {
		return unknownVersion
	}

	if len(rev) > 12 {
		rev = rev[:12]
	}

	// the "rev." prefix guarantees a valid semver prerelease identifier even
	// when the abbreviated hash is all digits with a leading zero
	v := "v0.0.0-rev." + rev

	if modified {
		v += "+dirty"
	}

	return v
}
