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

func testSignedCorimCBOR(t *testing.T) []byte {
	t.Helper()

	signer, err := corim.NewSignerFromJWK(testECKey)
	require.NoError(t, err)

	var unsigned corim.UnsignedCorim
	require.NoError(t, unsigned.FromCBOR(testCorimValid))

	var meta corim.Meta
	require.NoError(t, meta.FromJSON(testMetaValid))

	s := corim.SignedCorim{
		UnsignedCorim: unsigned,
		Meta:          meta,
	}

	cbor, err := s.Sign(signer)
	require.NoError(t, err)

	return cbor
}

func Test_CorimVerifyCmd_unknown_argument(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	args := []string{"--unknown-argument=val"}
	cmd.SetArgs(args)

	err := cmd.Execute()
	assert.EqualError(t, err, "unknown flag: --unknown-argument")
}

func Test_CorimVerifyCmd_mandatory_args_missing_corim_file(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	args := []string{
		"--key=ignored.jwk",
	}
	cmd.SetArgs(args)

	err := cmd.Execute()
	assert.EqualError(t, err, "no CoRIM supplied")
}

func Test_CorimVerifyCmd_non_existent_signed_corim_file(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	args := []string{
		"--file=nonexistent.cbor",
		"--key=ignored.jwk",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()

	err := cmd.Execute()
	assert.EqualError(t, err, "error loading signed CoRIM from nonexistent.cbor: open nonexistent.cbor: file does not exist")
}

func Test_CorimVerifyCmd_bad_signed_corim(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	args := []string{
		"--file=bad.txt",
		"--key=ignored.jwk",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "bad.txt", []byte("hello!"), 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.EqualError(t, err, "error decoding signed CoRIM from bad.txt: failed CBOR decoding for COSE-Sign1 signed CoRIM: cbor: invalid COSE_Sign1_Tagged object")
}

func Test_CorimVerifyCmd_non_existent_key_file(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	args := []string{
		"--file=ok.cbor",
		"--key=nonexistent.jwk",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testSignedCorimCBOR(t), 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.EqualError(t, err, "error loading verifying key from nonexistent.jwk: open nonexistent.jwk: file does not exist")
}

func Test_CorimVerifyCmd_invalid_key_file(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	args := []string{
		"--file=ok.cbor",
		"--key=invalid.jwk",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testSignedCorimCBOR(t), 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "invalid.jwk", []byte("{}"), 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.EqualError(t, err, "error loading verifying key from invalid.jwk: invalid key type from JSON ()")
}

func Test_CorimVerifyCmd_ok(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	args := []string{
		"--file=ok.cbor",
		"--key=ok.jwk",
	}
	cmd.SetArgs(args)

	fs = afero.NewMemMapFs()
	err := afero.WriteFile(fs, "ok.cbor", testSignedCorimCBOR(t), 0644)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "ok.jwk", testECKey, 0644)
	require.NoError(t, err)

	err = cmd.Execute()
	assert.NoError(t, err)
}

func Test_CorimVerifyCmd_key_with_crl_policy_fails(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "ok.cbor", testSignedCorimCBOR(t), 0644))
	require.NoError(t, afero.WriteFile(fs, "ok.jwk", testECKey, 0644))

	cmd.SetArgs([]string{
		"--file=ok.cbor",
		"--key=ok.jwk",
		"--crl-policy=permissive",
	})

	err := cmd.Execute()
	assert.EqualError(t, err, "cannot use --crl-policy without --crl")
}

func Test_CorimVerifyCmd_key_with_crl_policy_permissive_and_crl_fails(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fixture := buildX5chainPKIFixture(t)

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "ok.cbor", testSignedCorimCBOR(t), 0644))
	require.NoError(t, afero.WriteFile(fs, "ok.jwk", testECKey, 0644))
	require.NoError(t, afero.WriteFile(fs, "issuer.crl", fixture.emptyCRL(t), 0644))

	cmd.SetArgs([]string{
		"--file=ok.cbor",
		"--key=ok.jwk",
		"--crl=issuer.crl",
		"--crl-policy=permissive",
	})

	err := cmd.Execute()
	assert.EqualError(t, err, "cannot use --crl with --key")
}

func Test_parseCrlPolicy_emptyString(t *testing.T) {
	p, err := parseCrlPolicy("")
	assert.NoError(t, err)
	assert.Equal(t, corim.CrlPolicyStrict, p)
}
