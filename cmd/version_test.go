// Copyright 2026 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_versionFrom(t *testing.T) {
	const rev = "499dc19e9dc1fb68e1e38ecf8ca8a30bc0fc0353"

	info := func(mainVersion string, settings ...debug.BuildSetting) *debug.BuildInfo {
		return &debug.BuildInfo{
			Main:     debug.Module{Version: mainVersion},
			Settings: settings,
		}
	}

	tests := []struct {
		name     string
		override string
		bi       *debug.BuildInfo
		want     string
	}{
		{"override", "9.9.9", info("v0.2.0"), "9.9.9"},
		{"override without build info", "9.9.9", nil, "9.9.9"},
		{"no build info", "", nil, "v0.0.0-unknown"},
		{"tag", "", info("v0.2.0"), "v0.2.0"},
		{"pseudo-version", "", info(
			"v0.2.1-0.20261123120000-499dc19e9dc1+dirty"),
			"v0.2.1-0.20261123120000-499dc19e9dc1+dirty"},
		{"devel without vcs", "", info("(devel)"), "v0.0.0-unknown"},
		{"devel with revision", "", info(
			"(devel)", debug.BuildSetting{Key: "vcs.revision", Value: rev}),
			"v0.0.0-rev.499dc19e9dc1"},
		{"devel with modified revision", "", info("",
			debug.BuildSetting{Key: "vcs.revision", Value: rev},
			debug.BuildSetting{Key: "vcs.modified", Value: "true"}),
			"v0.0.0-rev.499dc19e9dc1+dirty"},
		{"short revision", "", info(
			"", debug.BuildSetting{Key: "vcs.revision", Value: "abc"}),
			"v0.0.0-rev.abc"},
		{"all-digit revision", "", info(
			"", debug.BuildSetting{Key: "vcs.revision", Value: "0123456789012345"}),
			"v0.0.0-rev.012345678901"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, versionFrom(tt.override, tt.bi), tt.name)
	}
}

func Test_version_not_empty(t *testing.T) {
	assert.NotEmpty(t, version())
}
