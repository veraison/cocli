// Copyright 2026 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	_ "embed"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim/coev"
)

//go:embed testcases/example-spdm-toc.cbor
var testSpdmTocCBOR []byte

//go:embed testcases/example-concise-evidence.cbor
var testCoevCBOR []byte

func Test_CoevDisplayCmd_unknown_argument(t *testing.T) {
	cmd := NewCoevDisplayCmd()
	cmd.SetArgs([]string{"--unknown-argument=val"})
	err := cmd.Execute()
	assert.EqualError(t, err, "unknown flag: --unknown-argument")
}

func Test_CoevDisplayCmd_no_files(t *testing.T) {
	cmd := NewCoevDisplayCmd()
	err := cmd.Execute()
	assert.EqualError(t, err, "no files supplied")
}

func Test_CoevDisplayCmd_no_files_found(t *testing.T) {
	cmd := NewCoevDisplayCmd()
	cmd.SetArgs([]string{"--file=missing.cbor", "--dir=nodir"})
	err := cmd.Execute()
	assert.EqualError(t, err, "no files found")
}

func Test_CoevDisplayCmd_spdm_toc_file(t *testing.T) {
	cmd := NewCoevDisplayCmd()
	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "spdm-toc.cbor", testSpdmTocCBOR, 0644))

	cmd.SetArgs([]string{"--file=spdm-toc.cbor"})
	err := cmd.Execute()
	assert.NoError(t, err)
}

func Test_CoevDisplayCmd_tagged_ce_file(t *testing.T) {
	// testCoevCBOR is a tagged-concise-evidence (tag 571)
	cmd := NewCoevDisplayCmd()
	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "ce.cbor", testCoevCBOR, 0644))

	cmd.SetArgs([]string{"--file=ce.cbor"})
	err := cmd.Execute()
	assert.NoError(t, err)
}

func Test_CoevDisplayCmd_untagged_ce_file(t *testing.T) {
	// Build an untagged ConciseEvidence CBOR from the JSON template
	var ce coev.ConciseEvidence
	require.NoError(t, ce.RegisterExtensions(coev.SpdmExtensionMap()))
	require.NoError(t, ce.FromJSON(testCoevTemplate))
	untagged, err := ce.ToCBOR()
	require.NoError(t, err)

	cmd := NewCoevDisplayCmd()
	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "untagged.cbor", untagged, 0644))

	cmd.SetArgs([]string{"--file=untagged.cbor"})
	err = cmd.Execute()
	assert.NoError(t, err)
}

func Test_CoevDisplayCmd_from_dir(t *testing.T) {
	cmd := NewCoevDisplayCmd()
	fs = afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("coevs", 0755))
	require.NoError(t, afero.WriteFile(fs, "coevs/toc.cbor", testSpdmTocCBOR, 0644))

	cmd.SetArgs([]string{"--dir=coevs"})
	err := cmd.Execute()
	assert.NoError(t, err)
}

func Test_CoevDisplayCmd_invalid_cbor(t *testing.T) {
	cmd := NewCoevDisplayCmd()
	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "bad.cbor", []byte{0xff, 0xff}, 0644))

	cmd.SetArgs([]string{"--file=bad.cbor"})
	err := cmd.Execute()
	assert.EqualError(t, err, "1/1 display(s) failed")
}

func Test_CoevDisplayCmd_multiple_files_partial_failure(t *testing.T) {
	cmd := NewCoevDisplayCmd()
	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "good.cbor", testSpdmTocCBOR, 0644))
	require.NoError(t, afero.WriteFile(fs, "bad.cbor", []byte{0xff, 0xff}, 0644))

	cmd.SetArgs([]string{"--file=good.cbor", "--file=bad.cbor"})
	err := cmd.Execute()
	assert.EqualError(t, err, "1/2 display(s) failed")
}
