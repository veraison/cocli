// Copyright 2026 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	_ "embed"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testcases/example-spdm-toc.json
var testSpdmTocTemplate []byte

//go:embed testcases/example-concise-evidence.json
var testCoevTemplate []byte

func Test_CoevCreateCmd_unknown_argument(t *testing.T) {
	cmd := NewCoevCreateCmd()
	cmd.SetArgs([]string{"--unknown-argument=val"})
	err := cmd.Execute()
	assert.EqualError(t, err, "unknown flag: --unknown-argument")
}

func Test_CoevCreateCmd_no_templates(t *testing.T) {
	cmd := NewCoevCreateCmd()
	err := cmd.Execute()
	assert.EqualError(t, err, "no templates supplied")
}

func Test_CoevCreateCmd_no_files_found(t *testing.T) {
	cmd := NewCoevCreateCmd()
	cmd.SetArgs([]string{"--template=missing.json", "--template-dir=nodir"})
	err := cmd.Execute()
	assert.EqualError(t, err, "no files found")
}

func Test_CoevCreateCmd_spdm_toc_from_file(t *testing.T) {
	cmd := NewCoevCreateCmd()
	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "spdm-toc.json", testSpdmTocTemplate, 0644))

	cmd.SetArgs([]string{"--template=spdm-toc.json"})
	err := cmd.Execute()
	assert.NoError(t, err)

	_, err = fs.Stat("spdm-toc.cbor")
	assert.NoError(t, err)
}

func Test_CoevCreateCmd_standalone_ce_from_file(t *testing.T) {
	cmd := NewCoevCreateCmd()
	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "ce.json", testCoevTemplate, 0644))

	cmd.SetArgs([]string{"--template=ce.json"})
	err := cmd.Execute()
	assert.NoError(t, err)

	_, err = fs.Stat("ce.cbor")
	assert.NoError(t, err)
}

func Test_CoevCreateCmd_output_dir(t *testing.T) {
	cmd := NewCoevCreateCmd()
	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "spdm-toc.json", testSpdmTocTemplate, 0644))
	require.NoError(t, fs.MkdirAll("out", 0755))

	cmd.SetArgs([]string{"--template=spdm-toc.json", "--output-dir=out"})
	err := cmd.Execute()
	assert.NoError(t, err)

	_, err = fs.Stat("out/spdm-toc.cbor")
	assert.NoError(t, err)
}

func Test_CoevCreateCmd_from_dir(t *testing.T) {
	cmd := NewCoevCreateCmd()
	fs = afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("templates", 0755))
	require.NoError(t, afero.WriteFile(fs, "templates/t1.json", testSpdmTocTemplate, 0644))
	require.NoError(t, afero.WriteFile(fs, "templates/t2.json", testCoevTemplate, 0644))

	cmd.SetArgs([]string{"--template-dir=templates"})
	err := cmd.Execute()
	assert.NoError(t, err)

	_, err = fs.Stat("t1.cbor")
	assert.NoError(t, err)
	_, err = fs.Stat("t2.cbor")
	assert.NoError(t, err)
}

func Test_CoevCreateCmd_invalid_json(t *testing.T) {
	cmd := NewCoevCreateCmd()
	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "bad.json", []byte("{"), 0644))

	cmd.SetArgs([]string{"--template=bad.json"})
	err := cmd.Execute()
	assert.EqualError(t, err, "1/1 creation(s) failed")
}

func Test_CoevCreateCmd_invalid_spdm_toc_content(t *testing.T) {
	cmd := NewCoevCreateCmd()
	fs = afero.NewMemMapFs()
	// tagged-evidence present but CE is empty — validation will fail
	require.NoError(t, afero.WriteFile(fs, "bad.json",
		[]byte(`{"tagged-evidence":[{}]}`), 0644))

	cmd.SetArgs([]string{"--template=bad.json"})
	err := cmd.Execute()
	assert.EqualError(t, err, "1/1 creation(s) failed")
}

func Test_CoevCreateCmd_invalid_standalone_ce_content(t *testing.T) {
	cmd := NewCoevCreateCmd()
	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "bad.json", []byte("{}"), 0644))

	cmd.SetArgs([]string{"--template=bad.json"})
	err := cmd.Execute()
	assert.EqualError(t, err, "1/1 creation(s) failed")
}
