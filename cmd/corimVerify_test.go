// Copyright 2021-2024 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim/corim"
	cose "github.com/veraison/go-cose"
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
	assert.Equal(t, cose.CRLPolicyStrict, p)
}

func Test_CorimVerifyCmd_new_flags_invalid(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"invalid mode", []string{"--revocation-mode=invalid"}, "invalid revocation mode"},
		{"empty mode", []string{"--revocation-mode="}, "invalid revocation mode"},
		{"full chain needs CRL", []string{"--revocation-mode=full-chain"}, "requires --crl"},
		{"leaf needs CRL", []string{"--revocation-mode=leaf-only"}, "requires --crl"},
		{"key with mode", []string{"--key=key.jwk", "--revocation-mode=disabled"}, "cannot use --revocation-mode with --key"},
		{"key with intermediates", []string{"--key=key.jwk", "--intermediate=issuer.der"},
			"cannot use --intermediate with --key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewCorimVerifyCmd()
			cmd.SetArgs(append([]string{"--file=signed.cbor"}, tc.args...))
			assert.ErrorContains(t, cmd.Execute(), tc.want)
		})
	}
}

func Test_CorimVerifyCmd_revocation_modes(t *testing.T) {
	fixture, intermediateDER := buildX5chainIntermediateFixture(t)
	_, unrelatedCRL := buildUnrelatedCRL(t)
	validCRL := fixture.emptyCRL(t)
	revokedCRL := fixture.crlWithRevokedLeaf(t)
	for _, tc := range []struct {
		name   string
		mode   string
		policy string
		crl    []byte
		want   error
	}{
		{"full chain needs root CRL", "full-chain", "", validCRL, corim.ErrX5ChainCRLMissing},
		{"leaf only needs issuer CRL", "leaf-only", "", validCRL, nil},
		{"leaf only rejects revoked leaf", "leaf-only", "", revokedCRL, corim.ErrX5ChainRevoked},
		{"disabled ignores revoked leaf", "disabled", "", revokedCRL, nil},
		{"leaf permissive missing issuer", "leaf-only", "permissive", unrelatedCRL, nil},
		{"leaf permissive revoked", "leaf-only", "permissive", revokedCRL, corim.ErrX5ChainRevoked},
		{"disabled strict revoked", "disabled", "strict", revokedCRL, nil},
		{"disabled permissive revoked", "disabled", "permissive", revokedCRL, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewCorimVerifyCmd()
			var stderr bytes.Buffer
			cmd.SetErr(&stderr)
			fs = afero.NewMemMapFs()
			require.NoError(t, afero.WriteFile(fs, "signed.cbor", fixture.signedCBOR, 0644))
			require.NoError(t, afero.WriteFile(fs, "root.der", fixture.anchorDER, 0644))
			require.NoError(t, afero.WriteFile(fs, "issuer.der", intermediateDER, 0644))
			require.NoError(t, afero.WriteFile(fs, "issuer.crl", tc.crl, 0644))
			args := []string{"--file=signed.cbor", "--trust-anchors=root.der", "--intermediate=issuer.der",
				"--crl=issuer.crl", "--revocation-mode=" + tc.mode}
			if tc.policy != "" {
				args = append(args, "--crl-policy="+tc.policy)
			}
			cmd.SetArgs(args)
			err := cmd.Execute()
			if tc.want != nil {
				assert.ErrorIs(t, err, tc.want)
			} else {
				assert.NoError(t, err)
			}
			warning := "warning: --revocation-mode=disabled; supplied CRLs are not used for revocation checks"
			if tc.mode == "disabled" {
				assert.Contains(t, stderr.String(), warning)
			} else {
				assert.NotContains(t, stderr.String(), warning)
			}
		})
	}
}

func Test_CorimVerifyCmd_no_crl_no_revocation_warning(t *testing.T) {
	fixture := buildX5chainPKIFixture(t)
	for _, mode := range []string{"", "disabled"} {
		name := mode
		if name == "" {
			name = "default"
		}
		t.Run(name, func(t *testing.T) {
			cmd := NewCorimVerifyCmd()
			var stderr bytes.Buffer
			cmd.SetErr(&stderr)
			fs = afero.NewMemMapFs()
			require.NoError(t, afero.WriteFile(fs, "signed.cbor", fixture.signedCBOR, 0644))
			require.NoError(t, afero.WriteFile(fs, "root.der", fixture.anchorDER, 0644))
			args := []string{"--file=signed.cbor", "--trust-anchors=root.der"}
			if mode != "" {
				args = append(args, "--revocation-mode="+mode)
			}
			cmd.SetArgs(args)
			require.NoError(t, cmd.Execute())
			assert.Empty(t, stderr.String())
		})
	}
}

func Test_CorimVerifyCmd_additional_intermediates(t *testing.T) {
	fixture, intermediateDER := buildX5chainIntermediateFixture(t)
	pemBundle := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: intermediateDER})
	pemBundle = append(pemBundle, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: fixture.anchorDER})...)
	for _, tc := range []struct {
		name  string
		data  []byte
		extra []string
		want  string
	}{
		{"missing chain", nil, nil, "certificate signed by unknown authority"},
		{"DER", intermediateDER, []string{"--intermediate=issuer.cert"}, ""},
		{"PEM bundle repeated", pemBundle, []string{"--intermediate=issuer.cert", "--intermediate=issuer.cert"}, ""},
		{"missing file", nil, []string{"--intermediate=missing.cert"}, "loading intermediate certificates from missing.cert"},
		{"invalid certificate", []byte("invalid"), []string{"--intermediate=issuer.cert"},
			"parsing intermediate certificates from issuer.cert"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewCorimVerifyCmd()
			fs = afero.NewMemMapFs()
			require.NoError(t, afero.WriteFile(fs, "signed.cbor", fixture.signedCBOR, 0644))
			require.NoError(t, afero.WriteFile(fs, "root.der", fixture.anchorDER, 0644))
			require.NoError(t, afero.WriteFile(fs, "issuer.cert", tc.data, 0644))
			args := []string{"--file=signed.cbor", "--trust-anchors=root.der", "--revocation-mode=disabled"}
			cmd.SetArgs(append(args, tc.extra...))
			err := cmd.Execute()
			if tc.want == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.want)
				if len(tc.extra) > 0 {
					assert.NotContains(t, err.Error(), "trust anchor")
				}
			}
		})
	}
	t.Run("intermediate does not grant trust", func(t *testing.T) {
		cmd := NewCorimVerifyCmd()
		fs = afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, "signed.cbor", fixture.signedCBOR, 0644))
		require.NoError(t, afero.WriteFile(fs, "wrong-root.der", testUnrelatedTrustAnchor(t), 0644))
		require.NoError(t, afero.WriteFile(fs, "chain.pem", pemBundle, 0644))
		cmd.SetArgs([]string{"--file=signed.cbor", "--trust-anchors=wrong-root.der", "--intermediate=chain.pem"})
		assert.ErrorIs(t, cmd.Execute(), corim.ErrX5ChainNoTrust)
	})
}

// Build a root -> intermediate -> leaf path, with only the leaf in x5chain.
// The returned fixture issues CRLs from the intermediate, leaving the root CRL absent.
func buildX5chainIntermediateFixture(t *testing.T) (x5chainPKIFixture, []byte) {
	t.Helper()
	fixture, _, intermediates := buildX5chainPathFixture(t, 1)
	return fixture, intermediates[0]
}

// Return a leaf-only message, its root issuer, and intermediates in root-to-leaf order.
func buildX5chainPathFixture(t *testing.T, intermediateCount int) (x5chainPKIFixture, x5chainPKIFixture, [][]byte) {
	t.Helper()
	root := buildX5chainPKIFixture(t)
	ca, caKey := root.ca, root.caKey
	var intermediates [][]byte
	for i := 0; i < intermediateCount; i++ {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		caTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(int64(i + 3)),
			Subject:      pkix.Name{CommonName: fmt.Sprintf("Intermediate CA %d", i+1)},
			NotBefore:    time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
			KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign, BasicConstraintsValid: true, IsCA: true,
		}
		caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, ca, &key.PublicKey, caKey)
		require.NoError(t, err)
		ca, err = x509.ParseCertificate(caDER)
		require.NoError(t, err)
		caKey = key
		intermediates = append(intermediates, caDER)
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	leaf := &x509.Certificate{
		SerialNumber: big.NewInt(100), Subject: pkix.Name{CommonName: "Leaf"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature,
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leaf, ca, &leafKey.PublicKey, caKey)
	require.NoError(t, err)
	signer, err := corim.NewSignerFromJWK(mustECPrivateKeyJWK(t, leafKey))
	require.NoError(t, err)
	var signed corim.SignedCorim
	require.NoError(t, signed.UnsignedCorim.FromCBOR(testCorimValid))
	require.NoError(t, signed.Meta.FromJSON(testMetaValid))
	require.NoError(t, signed.AddSigningCert(leafDER))
	signedCBOR, err := signed.Sign(signer)
	require.NoError(t, err)
	return x5chainPKIFixture{
		signedCBOR: signedCBOR,
		anchorDER:  root.anchorDER,
		ca:         ca,
		caKey:      caKey,
		leafSerial: leaf.SerialNumber,
	}, root, intermediates
}

func Test_CorimVerifyCmd_full_chain_crls(t *testing.T) {
	fixture, root, intermediates := buildX5chainPathFixture(t, 1)
	for _, tc := range []struct {
		name    string
		mode    string
		revoked bool
	}{
		{"explicit full chain valid", "full-chain", false},
		{"explicit full chain revoked intermediate", "full-chain", true},
		{"default full chain valid", "", false},
		{"default full chain revoked intermediate", "", true},
		{"leaf only skips revoked intermediate", "leaf-only", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rootCRL := &x509.RevocationList{
				Number: big.NewInt(1), ThisUpdate: time.Now().Add(-time.Minute), NextUpdate: time.Now().Add(time.Hour),
			}
			if tc.revoked {
				rootCRL.RevokedCertificateEntries = []x509.RevocationListEntry{
					{SerialNumber: fixture.ca.SerialNumber, RevocationTime: time.Now().Add(-time.Minute)},
				}
			}
			rootCRLDER, err := x509.CreateRevocationList(rand.Reader, rootCRL, root.ca, root.caKey)
			require.NoError(t, err)
			cmd := NewCorimVerifyCmd()
			fs = afero.NewMemMapFs()
			require.NoError(t, afero.WriteFile(fs, "signed.cbor", fixture.signedCBOR, 0644))
			require.NoError(t, afero.WriteFile(fs, "root.der", root.anchorDER, 0644))
			require.NoError(t, afero.WriteFile(fs, "issuer.der", intermediates[0], 0644))
			require.NoError(t, afero.WriteFile(fs, "issuer.crl", fixture.emptyCRL(t), 0644))
			require.NoError(t, afero.WriteFile(fs, "root.crl", rootCRLDER, 0644))
			args := []string{"--file=signed.cbor", "--trust-anchors=root.der", "--intermediate=issuer.der",
				"--crl=issuer.crl", "--crl=root.crl"}
			if tc.mode != "" {
				args = append(args, "--revocation-mode="+tc.mode)
			}
			cmd.SetArgs(args)
			err = cmd.Execute()
			if tc.revoked && tc.mode != "leaf-only" {
				assert.ErrorIs(t, err, corim.ErrX5ChainRevoked)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_CorimVerifyCmd_multiple_intermediate_files(t *testing.T) {
	fixture, root, intermediates := buildX5chainPathFixture(t, 2)
	for _, tc := range []struct {
		name  string
		files []string
		valid bool
	}{
		{"missing upper issuer", []string{"lower.der"}, false},
		{"missing lower issuer", []string{"upper.pem"}, false},
		{"both issuers", []string{"upper.pem", "lower.der"}, true},
		{"both issuers reversed", []string{"lower.der", "upper.pem"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewCorimVerifyCmd()
			fs = afero.NewMemMapFs()
			require.NoError(t, afero.WriteFile(fs, "signed.cbor", fixture.signedCBOR, 0644))
			require.NoError(t, afero.WriteFile(fs, "root.der", root.anchorDER, 0644))
			upperPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: intermediates[0]})
			require.NoError(t, afero.WriteFile(fs, "upper.pem", upperPEM, 0644))
			require.NoError(t, afero.WriteFile(fs, "lower.der", intermediates[1], 0644))
			args := []string{"--file=signed.cbor", "--trust-anchors=root.der"}
			for _, path := range tc.files {
				args = append(args, "--intermediate="+path)
			}
			cmd.SetArgs(args)
			err := cmd.Execute()
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, corim.ErrX5ChainNoTrust)
			}
		})
	}
}

func Test_CorimVerifyCmd_system_roots(t *testing.T) {
	// Go caches system roots globally. A subprocess isolates our test root and
	// forces Go's fallback store without modifying the host's trust configuration.
	if os.Getenv("COCLI_TEST_SYSTEM_ROOTS") != "1" {
		testBinary, err := os.Executable()
		require.NoError(t, err)
		cmd := exec.Command(testBinary, "-test.run=^Test_CorimVerifyCmd_system_roots$")
		cmd.Env = append(os.Environ(), "COCLI_TEST_SYSTEM_ROOTS=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s", output)
		return
	}
	t.Setenv("GODEBUG", "x509usefallbackroots=1")
	fixture := buildX5chainPKIFixture(t)
	pool := x509.NewCertPool()
	pool.AddCert(fixture.ca)
	x509.SetFallbackRoots(pool)
	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed.cbor", fixture.signedCBOR, 0644))
	require.NoError(t, afero.WriteFile(fs, "wrong-root.der", testUnrelatedTrustAnchor(t), 0644))

	cmd := NewCorimVerifyCmd()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--file=signed.cbor"})
	require.NoError(t, cmd.Execute())
	assert.Contains(t, stderr.String(), "OS trust store")

	// Explicit trust anchors must replace the system store, even when it could
	// otherwise establish a valid path for this same message.
	cmd = NewCorimVerifyCmd()
	stderr.Reset()
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--file=signed.cbor", "--trust-anchors=wrong-root.der"})
	assert.ErrorIs(t, cmd.Execute(), corim.ErrX5ChainNoTrust)
	assert.NotContains(t, stderr.String(), "OS trust store")
}
