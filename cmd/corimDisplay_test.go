// Copyright 2021-2024 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define your truncated CBOR payload
var truncatedCorim = []byte{
	0xD9, 0x01, 0xF5,   // tag(501) [unsigned-corim-map]
	0xA1,               // CBOR map with 1 key
	0x01,               // Key = 1
	0x65,               // Value: string of length 5
	'h', 'e', 'l', 'l', // Incomplete string value (missing one byte)
}

func Test_CorimDisplayCmd_unknown_argument(t *testing.T) {
	cmd := NewCorimDisplayCmd()

	args := []string{"--unknown-argument=val"}
	cmd.SetArgs(args)

	err := cmd.Execute()
	assert.EqualError(t, err, "unknown flag: --unknown-argument")
}

func Test_CorimDisplayCmd_mandatory_args_missing_corim_file(t *testing.T) {
	cmd := NewCorimDisplayCmd()

	args := []string{
		"--show-tags",
	}
	cmd.SetArgs(args)

	err := cmd.Execute()
	assert.EqualError(t, err, "no CoRIM supplied")
}

func Test_CorimDisplayCmd_non_existent_corim_file(t *testing.T) {
	cmd := NewCorimDisplayCmd()

	args := []string{
		"--file=nonexistent.cbor",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()

	err := cmd.Execute()
	assert.EqualError(t, err, "error loading CoRIM from nonexistent.cbor: open nonexistent.cbor: file does not exist")
}

func Test_CorimDisplayCmd_bad_signed_corim(t *testing.T) {
	cmd := NewCorimDisplayCmd()

	args := []string{
		"--file=bad.txt",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "bad.txt", []byte("hello!"), 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.EqualError(t, err, "error decoding CoRIM (signed or unsigned) from bad.txt: did not see unsigned CoRIM tag")
}

func Test_CorimDisplayCmd_invalid_signed_corim_Truncated(t *testing.T) {
	cmd := NewCorimDisplayCmd()

	args := []string{
		"--file=truncated.cbor",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	// Use truncatedCorim to force "unexpected EOF" error
	err := afero.WriteFile(fs, "truncated.cbor", truncatedCorim, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.EqualError(
		t,
		err,
		"error decoding CoRIM (signed or unsigned) from truncated.cbor: map item 0: could not unmarshal value: unexpected EOF",
	)
}

func Test_CorimDisplayCmd_ok_top_level_view(t *testing.T) {
	cmd := NewCorimDisplayCmd()

	args := []string{
		"--file=ok.cbor",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testSignedCorimValid, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.NoError(t, err)
}

func Test_CorimDisplayCmd_ok_nested_view(t *testing.T) {
	cmd := NewCorimDisplayCmd()

	args := []string{
		"--file=ok.cbor",
		"--show-tags",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testSignedCorimValid, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.NoError(t, err)
}

func Test_CorimDisplayCmd_ok_top_level_view_with_cots(t *testing.T) {
	cmd := NewCorimDisplayCmd()

	args := []string{
		"--file=ok.cbor",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testSignedCorimValidWithCots, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.NoError(t, err)
}

func Test_CorimDisplayCmd_ok_nested_view_with_cots(t *testing.T) {
	cmd := NewCorimDisplayCmd()

	args := []string{
		"--file=ok.cbor",
		"--show-tags",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testSignedCorimValidWithCots, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.NoError(t, err)
}

func Test_CorimDisplayCmd_unsigned_corim_top_level(t *testing.T) {
	cmd := NewCorimDisplayCmd()

	args := []string{
		"--file=unsigned.cbor",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "unsigned.cbor", testCorimValid, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.NoError(t, err)
}

func Test_CorimDisplayCmd_unsigned_corim_nested_view(t *testing.T) {
	cmd := NewCorimDisplayCmd()

	args := []string{
		"--file=unsigned.cbor",
		"--show-tags",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "unsigned.cbor", testCorimValid, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.NoError(t, err)
}
