// Copyright 2021-2024 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim/corim"
)

func Test_CorimSignCmd_unknown_argument(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{"--unknown-argument=val"}
	cmd.SetArgs(args)

	err := cmd.Execute()
	assert.EqualError(t, err, "unknown flag: --unknown-argument")
}

func Test_CorimSignCmd_mandatory_args_missing_corim_file(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--key=ignored.jwk",
		"--meta=ignored.json",
	}
	cmd.SetArgs(args)

	err := cmd.Execute()
	assert.EqualError(t, err, "no CoRIM supplied")
}

func Test_CorimSignCmd_mandatory_args_missing_meta_file(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=ignored.cbor",
		"--key=ignored.jwk",
	}
	cmd.SetArgs(args)

	err := cmd.Execute()
	assert.EqualError(t, err, "no CoRIM Meta supplied")
}

func Test_CorimSignCmd_mandatory_args_missing_key_file(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=ignored.cbor",
		"--meta=ignored.json",
	}
	cmd.SetArgs(args)

	err := cmd.Execute()
	assert.EqualError(t, err, "no key supplied")
}

func Test_CorimSignCmd_non_existent_unsigned_corim_file(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=nonexistent.cbor",
		"--key=ignored.jwk",
		"--meta=ignored.json",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()

	err := cmd.Execute()
	assert.EqualError(t, err, "error loading unsigned CoRIM from nonexistent.cbor: open nonexistent.cbor: file does not exist")
}

func Test_CorimSignCmd_bad_unsigned_corim(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=bad.txt",
		"--key=ignored.jwk",
		"--meta=ignored.json",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "bad.txt", []byte("hello!"), 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.EqualError(t, err, "error decoding unsigned CoRIM from bad.txt: did not see unsigned CoRIM tag")
}

func Test_CorimSignCmd_invalid_unsigned_corim(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=invalid.cbor",
		"--key=ignored.jwk",
		"--meta=ignored.json",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "invalid.cbor", testCorimInvalid, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.EqualError(t, err, `error decoding unsigned CoRIM from invalid.cbor: missing mandatory field "Tags" (1)`)
}

func Test_CorimSignCmd_non_existent_meta_file(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=ok.cbor",
		"--key=ignored.jwk",
		"--meta=nonexistent.json",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testCorimValid, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.EqualError(t, err, "error loading CoRIM Meta from nonexistent.json: open nonexistent.json: file does not exist")
}

func Test_CorimSignCmd_bad_meta_file(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=ok.cbor",
		"--key=ignored.jwk",
		"--meta=bad.json",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testCorimValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "bad.json", []byte("{"), 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.EqualError(t, err, "error decoding CoRIM Meta from bad.json: unexpected end of JSON input")
}

func Test_CorimSignCmd_invalid_meta_file(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=ok.cbor",
		"--key=ignored.jwk",
		"--meta=invalid.json",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testCorimValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "invalid.json", testMetaInvalid, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.EqualError(t, err, "error validating CoRIM Meta: invalid signer: empty name")
}

func Test_CorimSignCmd_non_existent_key_file(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=ok.cbor",
		"--key=nonexistent.jwk",
		"--meta=ok.json",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testCorimValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.json", testMetaValid, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.EqualError(t, err, "error loading signing key from nonexistent.jwk: open nonexistent.jwk: file does not exist")
}

func Test_CorimSignCmd_invalid_key_file(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=ok.cbor",
		"--key=invalid.jwk",
		"--meta=ok.json",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testCorimValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.json", testMetaValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "invalid.jwk", []byte("{}"), 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.EqualError(t, err, "error loading signing key from invalid.jwk: invalid key type from JSON ()")
}

func Test_CorimSignCmd_ok_with_default_output_file(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=ok.cbor",
		"--key=ok.jwk",
		"--meta=ok.json",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testCorimValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.json", testMetaValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.jwk", testECKey, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.NoError(t, err)

	_, err = fs.Stat("signed-ok.cbor")
	assert.NoError(t, err)

	bytes, err := afero.ReadFile(fs, "signed-ok.cbor")
	assert.NoError(t, err)

	signed, err := corim.UnmarshalAndValidateSignedCorimFromCBOR(bytes)
	assert.NoError(t, err)
	assert.Equal(t, []byte("1"), signed.KeyID)
}

func Test_CorimSignCmd_ok_with_custom_output_file(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=ok.cbor",
		"--key=ok.jwk",
		"--meta=ok.json",
		"--output=my-signed-corim.cbor",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testCorimValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.json", testMetaValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.jwk", testECKey, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.NoError(t, err)

	_, err = fs.Stat("my-signed-corim.cbor")
	assert.NoError(t, err)
}

func Test_CorimSignCmd_with_signing_cert(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=ok.cbor",
		"--key=ok.jwk",
		"--meta=ok.json",
		"--cert=cert.der",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testCorimValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.json", testMetaValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.jwk", testECKey, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "cert.der", testSigningCertificate, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.NoError(t, err)

	_, err = fs.Stat("signed-ok.cbor")
	assert.NoError(t, err)
}

func Test_CorimSignCmd_with_signing_cert_and_intermediates(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=ok.cbor",
		"--key=ok.jwk",
		"--meta=ok.json",
		"--cert=cert.der",
		"--intermediates=intermediates.der",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testCorimValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.json", testMetaValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.jwk", testECKey, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "cert.der", testSigningCertificate, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "intermediates.der", testIntermediateCerts, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.NoError(t, err)

	_, err = fs.Stat("signed-ok.cbor")
	assert.NoError(t, err)
}

func Test_CorimSignCmd_invalid_cert_file(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=ok.cbor",
		"--key=ok.jwk",
		"--meta=ok.json",
		"--cert=invalid-cert.der",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testCorimValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.json", testMetaValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.jwk", testECKey, 0644)
	require.NoError(t, err)
	// Write an invalid certificate (just some random bytes that aren't a valid DER certificate)
	err = afero.WriteFile(fs, "invalid-cert.der", []byte{0x30, 0x03, 0x02, 0x01, 0x01}, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.ErrorContains(t, err, "error adding signing certificate")
}

func Test_CorimSignCmd_invalid_intermediates_file(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=ok.cbor",
		"--key=ok.jwk",
		"--meta=ok.json",
		"--cert=cert.der",
		"--intermediates=invalid-intermediates.der",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testCorimValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.json", testMetaValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.jwk", testECKey, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "cert.der", testSigningCertificate, 0644)
	require.NoError(t, err)
	// Write invalid intermediate certificates
	err = afero.WriteFile(fs, "invalid-intermediates.der", []byte{0x30, 0x03, 0x02, 0x01, 0x02}, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.ErrorContains(t, err, "error adding intermediate certificates")
}

func Test_CorimSignCmd_intermediates_without_signing_cert(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=ok.cbor",
		"--key=ok.jwk",
		"--meta=ok.json",
		"--intermediates=intermediates.der",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testCorimValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.json", testMetaValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.jwk", testECKey, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "intermediates.der", testIntermediateCerts, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.EqualError(t, err, "cannot add intermediate certificates without a signing certificate")
}

func Test_CorimSignCmd_nonexistent_cert_file(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=ok.cbor",
		"--key=ok.jwk",
		"--meta=ok.json",
		"--cert=nonexistent.der",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testCorimValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.json", testMetaValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.jwk", testECKey, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.EqualError(t, err, "error loading signing certificate from nonexistent.der: open nonexistent.der: file does not exist")
}

func Test_CorimSignCmd_nonexistent_intermediates_file(t *testing.T) {
	cmd := NewCorimSignCmd()

	args := []string{
		"--file=ok.cbor",
		"--key=ok.jwk",
		"--meta=ok.json",
		"--cert=cert.der",
		"--intermediates=nonexistent.der",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testCorimValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.json", testMetaValid, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.jwk", testECKey, 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "cert.der", testSigningCertificate, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.EqualError(t, err, "error loading intermediate certificates from nonexistent.der: open nonexistent.der: file does not exist")
}
