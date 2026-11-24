// Copyright 2021-2026 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/veraison/corim/profiles/cca"
	"github.com/veraison/corim/profiles/tdx"
)

func Test_ComidValidateCmd_unknown_argument(t *testing.T) {
	cmd := NewComidValidateCmd()

	args := []string{"--unknown-argument=val"}
	cmd.SetArgs(args)

	err := cmd.Execute()
	assert.EqualError(t, err, "unknown flag: --unknown-argument")
}

func Test_ComidValidateCmd_no_files(t *testing.T) {
	cmd := NewComidValidateCmd()

	// no args

	err := cmd.Execute()
	assert.EqualError(t, err, "no files supplied")
}

func Test_ComidValidateCmd_no_files_found(t *testing.T) {
	cmd := NewComidValidateCmd()

	args := []string{
		"--file=unknown",
		"--dir=unsure",
	}
	cmd.SetArgs(args)

	err := cmd.Execute()
	assert.EqualError(t, err, "no files found")
}

func Test_ComidValidateCmd_file_with_invalid_cbor(t *testing.T) {
	var err error

	cmd := NewComidValidateCmd()

	fs = afero.NewMemMapFs()
	err = afero.WriteFile(fs, "invalid.cbor", []byte{0xff, 0xff}, 0400)
	require.NoError(t, err)

	args := []string{
		"--file=invalid.cbor",
	}
	cmd.SetArgs(args)

	err = cmd.Execute()
	assert.EqualError(t, err, "1/1 validation(s) failed")
}

func Test_ComidValidateCmd_file_with_invalid_comid(t *testing.T) {
	var err error

	cmd := NewComidValidateCmd()

	fs = afero.NewMemMapFs()
	err = afero.WriteFile(fs, "bad-comid.cbor", []byte{0xa0}, 0400)
	require.NoError(t, err)

	args := []string{
		"--file=bad-comid.cbor",
	}
	cmd.SetArgs(args)

	err = cmd.Execute()
	assert.EqualError(t, err, "1/1 validation(s) failed")
}

func Test_ComidValidateCmd_file_with_valid_comid(t *testing.T) {
	var err error

	cmd := NewComidValidateCmd()

	fs = afero.NewMemMapFs()
	err = afero.WriteFile(fs, "ok.cbor", PSARefValCBOR, 0400)
	require.NoError(t, err)

	args := []string{
		"--file=ok.cbor",
	}
	cmd.SetArgs(args)

	err = cmd.Execute()
	assert.NoError(t, err)
}

func Test_ComidValidateCmd_file_with_valid_comid_from_dir(t *testing.T) {
	var err error

	cmd := NewComidValidateCmd()

	fs = afero.NewMemMapFs()
	err = afero.WriteFile(fs, "testdir/ok.cbor", PSARefValCBOR, 0400)
	require.NoError(t, err)

	args := []string{
		"--dir=testdir",
	}
	cmd.SetArgs(args)

	err = cmd.Execute()
	assert.NoError(t, err)
}

func Test_ComidValidateCmd_with_valid_comid(t *testing.T) {
	var err error
	profile := "--profile=" + testProfile
	cmd := NewComidValidateCmd()

	fs = afero.NewMemMapFs()
	err = afero.WriteFile(fs, "ok.cbor", tdx.ComidSeamRefVal, 0644)
	require.NoError(t, err)

	args := []string{
		"--file=ok.cbor",
		profile,
	}
	cmd.SetArgs(args)

	fmt.Printf("%x\n", tdx.ComidSeamRefVal)

	err = cmd.Execute()
	assert.NoError(t, err)
}

func Test_ComidValidateCmd_with_valid_cca_platform_comid(t *testing.T) {
	var err error
	profile := "--profile=tag:arm.com,2025:cca_platform#1.0.0"
	cmd := NewComidValidateCmd()

	fs = afero.NewMemMapFs()
	err = afero.WriteFile(fs, "cca-platform.cbor", CCAPlatformRefValCBOR, 0644)
	require.NoError(t, err)

	args := []string{
		"--file=cca-platform.cbor",
		profile,
	}
	cmd.SetArgs(args)

	err = cmd.Execute()
	assert.NoError(t, err)
}

func Test_ComidValidateCmd_with_valid_cca_realm_comid(t *testing.T) {
	var err error
	profile := "--profile=tag:arm.com,2025:cca_realm#1.0.0"
	cmd := NewComidValidateCmd()

	fs = afero.NewMemMapFs()
	err = afero.WriteFile(fs, "cca-realm.cbor", CCARealmRefValCBOR, 0644)
	require.NoError(t, err)

	args := []string{
		"--file=cca-realm.cbor",
		profile,
	}
	cmd.SetArgs(args)

	err = cmd.Execute()
	assert.NoError(t, err)
}
