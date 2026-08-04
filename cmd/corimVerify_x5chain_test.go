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
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim/corim"
	"github.com/veraison/corim/testdata"
)

func testEndEntityKeyJWK(t *testing.T) []byte {
	t.Helper()

	block, _ := pem.Decode(testdata.EndEntityKey)
	require.NotNil(t, block, "decoding testdata.EndEntityKey PEM")

	key, err := x509.ParseECPrivateKey(block.Bytes)
	require.NoError(t, err)

	return mustECPrivateKeyJWK(t, key)
}

func testSignedCorimWithX5ChainCBOR(t *testing.T) []byte {
	t.Helper()

	signer, err := corim.NewSignerFromJWK(testEndEntityKeyJWK(t))
	require.NoError(t, err)

	var unsigned corim.UnsignedCorim
	require.NoError(t, unsigned.FromCBOR(testCorimValid))

	var meta corim.Meta
	require.NoError(t, meta.FromJSON(testMetaValid))

	s := corim.SignedCorim{
		UnsignedCorim: unsigned,
		Meta:          meta,
	}
	require.NoError(t, s.AddSigningCert(testdata.EndEntityDer))
	require.NoError(t, s.AddIntermediateCerts(testdata.IntermediateCA))

	cbor, err := s.Sign(signer)
	require.NoError(t, err)

	return cbor
}

func Test_CorimVerifyCmd_x5chain_missing_without_key(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "unsigned.cbor", testSignedCorimCBOR(t), 0644))

	cmd.SetArgs([]string{"--file=unsigned.cbor"})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "no verification method")
}

func Test_CorimVerifyCmd_x5chain_with_trust_anchors_ok(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", testSignedCorimWithX5ChainCBOR(t), 0644))
	require.NoError(t, afero.WriteFile(fs, "anchor.der", testdata.RootCA, 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--trust-anchors=anchor.der",
	})

	assert.NoError(t, cmd.Execute())
}

func Test_CorimVerifyCmd_key_takes_precedence_over_x5chain(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", testSignedCorimWithX5ChainCBOR(t), 0644))
	require.NoError(t, afero.WriteFile(fs, "ok.jwk", testEndEntityKeyJWK(t), 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--key=ok.jwk",
	})

	assert.NoError(t, cmd.Execute())
}

func Test_CorimVerifyCmd_x5chain_key_precedence_wrong_key_fails(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", testSignedCorimWithX5ChainCBOR(t), 0644))
	require.NoError(t, afero.WriteFile(fs, "wrong.jwk", testECKey, 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--key=wrong.jwk",
	})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "error verifying signed-x5chain.cbor with key wrong.jwk")
	assert.NotContains(t, err.Error(), "using x5chain")
}

func Test_CorimVerifyCmd_key_with_trust_anchors_fails(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", testSignedCorimWithX5ChainCBOR(t), 0644))
	require.NoError(t, afero.WriteFile(fs, "ok.jwk", testEndEntityKeyJWK(t), 0644))
	require.NoError(t, afero.WriteFile(fs, "anchor.der", testdata.RootCA, 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--trust-anchors=anchor.der",
		"--key=ok.jwk",
	})

	err := cmd.Execute()
	assert.EqualError(t, err, "cannot use --trust-anchors with --key")
}

func Test_CorimVerifyCmd_key_with_crl_fails(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fixture := buildX5chainPKIFixture(t)

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", fixture.signedCBOR, 0644))
	require.NoError(t, afero.WriteFile(fs, "ok.jwk", testEndEntityKeyJWK(t), 0644))
	require.NoError(t, afero.WriteFile(fs, "issuer.crl", fixture.emptyCRL(t), 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--key=ok.jwk",
		"--crl=issuer.crl",
	})

	err := cmd.Execute()
	assert.EqualError(t, err, "cannot use --crl with --key")
}

func Test_CorimVerifyCmd_x5chain_wrong_trust_anchor_fails(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", testSignedCorimWithX5ChainCBOR(t), 0644))
	require.NoError(t, afero.WriteFile(fs, "wrong-anchor.der", testUnrelatedTrustAnchor(t), 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--trust-anchors=wrong-anchor.der",
	})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "x5chain verification failed")
}

func Test_CorimVerifyCmd_x5chain_non_existent_trust_anchor_file(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", testSignedCorimWithX5ChainCBOR(t), 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--trust-anchors=missing.der",
	})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "loading trust anchor from missing.der")
}

func Test_CorimVerifyCmd_x5chain_invalid_trust_anchor_parse(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", testSignedCorimWithX5ChainCBOR(t), 0644))
	require.NoError(t, afero.WriteFile(fs, "bad.der", []byte("not a certificate"), 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--trust-anchors=bad.der",
	})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "parsing trust anchor from bad.der")
}

func Test_CorimVerifyCmd_x5chain_with_crl_rejects_revoked(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fixture := buildX5chainPKIFixture(t)
	crlDER := fixture.crlWithRevokedLeaf(t)

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", fixture.signedCBOR, 0644))
	require.NoError(t, afero.WriteFile(fs, "anchor.der", fixture.anchorDER, 0644))
	require.NoError(t, afero.WriteFile(fs, "issuer.crl", crlDER, 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--trust-anchors=anchor.der",
		"--crl=issuer.crl",
	})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "revoked")
}

func Test_CorimVerifyCmd_x5chain_expired_crl_fails(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fixture := buildX5chainPKIFixture(t)
	crlDER := fixture.expiredCRL(t)

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", fixture.signedCBOR, 0644))
	require.NoError(t, afero.WriteFile(fs, "anchor.der", fixture.anchorDER, 0644))
	require.NoError(t, afero.WriteFile(fs, "issuer.crl", crlDER, 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--trust-anchors=anchor.der",
		"--crl=issuer.crl",
	})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "expired")
}

func Test_CorimVerifyCmd_x5chain_strict_crl_ok(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fixture := buildX5chainPKIFixture(t)
	crlDER := fixture.emptyCRL(t)

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", fixture.signedCBOR, 0644))
	require.NoError(t, afero.WriteFile(fs, "anchor.der", fixture.anchorDER, 0644))
	require.NoError(t, afero.WriteFile(fs, "issuer.crl", crlDER, 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--trust-anchors=anchor.der",
		"--crl=issuer.crl",
	})

	assert.NoError(t, cmd.Execute())
}

func Test_CorimVerifyCmd_x5chain_missing_crl_file(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", testSignedCorimWithX5ChainCBOR(t), 0644))
	require.NoError(t, afero.WriteFile(fs, "anchor.der", testdata.RootCA, 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--trust-anchors=anchor.der",
		"--crl=missing.crl",
	})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "loading CRL from missing.crl")
}

func Test_CorimVerifyCmd_x5chain_invalid_crl_parse(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", testSignedCorimWithX5ChainCBOR(t), 0644))
	require.NoError(t, afero.WriteFile(fs, "anchor.der", testdata.RootCA, 0644))
	require.NoError(t, afero.WriteFile(fs, "bad.crl", []byte("not a CRL"), 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--trust-anchors=anchor.der",
		"--crl=bad.crl",
	})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "parsing CRL from bad.crl")
}

func Test_CorimVerifyCmd_x5chain_crl_policy_without_crl_fails(t *testing.T) {
	// Include an invalid policy name to ensure the "--crl required" check
	// runs before policy parsing (otherwise "invalid CRL policy" is reported).
	for _, policy := range []string{"permissive", "strict", "whatever"} {
		t.Run(policy, func(t *testing.T) {
			cmd := NewCorimVerifyCmd()

			fs = afero.NewMemMapFs()
			require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", testSignedCorimWithX5ChainCBOR(t), 0644))
			require.NoError(t, afero.WriteFile(fs, "anchor.der", testdata.RootCA, 0644))

			cmd.SetArgs([]string{
				"--file=signed-x5chain.cbor",
				"--trust-anchors=anchor.der",
				"--crl-policy=" + policy,
			})

			err := cmd.Execute()
			assert.EqualError(t, err, "cannot use --crl-policy without --crl")
		})
	}
}

func Test_CorimVerifyCmd_x5chain_invalid_crl_policy(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fixture := buildX5chainPKIFixture(t)
	crlDER := fixture.emptyCRL(t)

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", fixture.signedCBOR, 0644))
	require.NoError(t, afero.WriteFile(fs, "anchor.der", fixture.anchorDER, 0644))
	require.NoError(t, afero.WriteFile(fs, "issuer.crl", crlDER, 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--trust-anchors=anchor.der",
		"--crl=issuer.crl",
		"--crl-policy=invalid",
	})

	err := cmd.Execute()
	assert.ErrorContains(t, err, `invalid CRL policy "invalid"`)
}

func Test_CorimVerifyCmd_x5chain_strict_crl_missing_issuer_fails(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fixture := buildX5chainPKIFixture(t)
	_, unrelatedCRLDER := buildUnrelatedCRL(t)

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", fixture.signedCBOR, 0644))
	require.NoError(t, afero.WriteFile(fs, "anchor.der", fixture.anchorDER, 0644))
	require.NoError(t, afero.WriteFile(fs, "unrelated.crl", unrelatedCRLDER, 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--trust-anchors=anchor.der",
		"--crl=unrelated.crl",
	})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "unable to get certificate CRL")
}

func Test_CorimVerifyCmd_x5chain_permissive_crl_skips_missing_issuer(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fixture := buildX5chainPKIFixture(t)
	_, unrelatedCRLDER := buildUnrelatedCRL(t)

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", fixture.signedCBOR, 0644))
	require.NoError(t, afero.WriteFile(fs, "anchor.der", fixture.anchorDER, 0644))
	require.NoError(t, afero.WriteFile(fs, "unrelated.crl", unrelatedCRLDER, 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--trust-anchors=anchor.der",
		"--crl=unrelated.crl",
		"--crl-policy=permissive",
	})

	assert.NoError(t, cmd.Execute())
}

func Test_CorimVerifyCmd_x5chain_pem_trust_anchor_ok(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	pemAnchor := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: testdata.RootCA})

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", testSignedCorimWithX5ChainCBOR(t), 0644))
	require.NoError(t, afero.WriteFile(fs, "anchor.pem", pemAnchor, 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--trust-anchors=anchor.pem",
	})

	assert.NoError(t, cmd.Execute())
}

func Test_CorimVerifyCmd_x5chain_multiple_trust_anchors_ok(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", testSignedCorimWithX5ChainCBOR(t), 0644))
	require.NoError(t, afero.WriteFile(fs, "anchor.der", testdata.RootCA, 0644))
	require.NoError(t, afero.WriteFile(fs, "unrelated.der", testUnrelatedTrustAnchor(t), 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--trust-anchors=unrelated.der",
		"--trust-anchors=anchor.der",
	})

	assert.NoError(t, cmd.Execute())
}

func Test_CorimVerifyCmd_x5chain_pem_crl_ok(t *testing.T) {
	cmd := NewCorimVerifyCmd()

	fixture := buildX5chainPKIFixture(t)
	pemCRL := pem.EncodeToMemory(&pem.Block{Type: "X509 CRL", Bytes: fixture.emptyCRL(t)})

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", fixture.signedCBOR, 0644))
	require.NoError(t, afero.WriteFile(fs, "anchor.der", fixture.anchorDER, 0644))
	require.NoError(t, afero.WriteFile(fs, "issuer.crl", pemCRL, 0644))

	cmd.SetArgs([]string{
		"--file=signed-x5chain.cbor",
		"--trust-anchors=anchor.der",
		"--crl=issuer.crl",
	})

	assert.NoError(t, cmd.Execute())
}

func Test_CorimVerifyCmd_x5chain_no_anchors_emits_warning(t *testing.T) {
	cmd := NewCorimVerifyCmd()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	fixture := buildX5chainPKIFixture(t)

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", fixture.signedCBOR, 0644))

	cmd.SetArgs([]string{"--file=signed-x5chain.cbor"})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "using x5chain")
	assert.Contains(t, stderr.String(), "OS trust store")
}

func Test_CorimVerifyCmd_sign_then_verify_x5chain(t *testing.T) {
	signCmd := NewCorimSignCmd()
	verifyCmd := NewCorimVerifyCmd()

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "ok.cbor", testCorimValid, 0644))
	require.NoError(t, afero.WriteFile(fs, "ok.json", testMetaValid, 0644))
	require.NoError(t, afero.WriteFile(fs, "ok.jwk", testEndEntityKeyJWK(t), 0644))
	require.NoError(t, afero.WriteFile(fs, "cert.der", testSigningCertificate, 0644))
	require.NoError(t, afero.WriteFile(fs, "intermediates.der", testIntermediateCerts, 0644))
	require.NoError(t, afero.WriteFile(fs, "rootCA.der", testdata.RootCA, 0644))

	signCmd.SetArgs([]string{
		"--file=ok.cbor",
		"--key=ok.jwk",
		"--meta=ok.json",
		"--cert=cert.der",
		"--intermediates=intermediates.der",
	})
	require.NoError(t, signCmd.Execute())

	verifyCmd.SetArgs([]string{
		"--file=signed-ok.cbor",
		"--trust-anchors=rootCA.der",
	})
	require.NoError(t, verifyCmd.Execute())
}

func Test_CorimVerifyCmd_key_skips_x5chain_warning(t *testing.T) {
	cmd := NewCorimVerifyCmd()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "signed-x5chain.cbor", testSignedCorimWithX5ChainCBOR(t), 0644))
	require.NoError(t, afero.WriteFile(fs, "ok.jwk", testEndEntityKeyJWK(t), 0644))

	cmd.SetArgs([]string{"--file=signed-x5chain.cbor", "--key=ok.jwk"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, stderr.String(), "x5chain/PKIX verification skipped")
}

func mustECPrivateKeyJWK(t *testing.T, key *ecdsa.PrivateKey) []byte {
	t.Helper()

	pad32 := func(b []byte) []byte {
		if len(b) > 32 {
			t.Fatalf("pad32: %d bytes exceeds 32", len(b))
		}
		out := make([]byte, 32)
		copy(out[32-len(b):], b)
		return out
	}

	jwk := map[string]string{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(pad32(key.X.Bytes())),
		"y":   base64.RawURLEncoding.EncodeToString(pad32(key.Y.Bytes())),
		"d":   base64.RawURLEncoding.EncodeToString(pad32(key.D.Bytes())),
	}

	out, err := json.Marshal(jwk)
	require.NoError(t, err)

	return out
}

func testUnrelatedTrustAnchor(t *testing.T) []byte {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Unrelated Trust Anchor"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	return caDER
}

func buildX5chainPKIFixture(t *testing.T) x5chainPKIFixture {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	ca, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	leafTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "Test Leaf"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, ca, &leafKey.PublicKey, caKey)
	require.NoError(t, err)

	signer, err := corim.NewSignerFromJWK(mustECPrivateKeyJWK(t, leafKey))
	require.NoError(t, err)

	var unsigned corim.UnsignedCorim
	require.NoError(t, unsigned.FromCBOR(testCorimValid))

	var meta corim.Meta
	require.NoError(t, meta.FromJSON(testMetaValid))

	s := corim.SignedCorim{
		UnsignedCorim: unsigned,
		Meta:          meta,
	}
	require.NoError(t, s.AddSigningCert(leafDER))
	require.NoError(t, s.AddIntermediateCerts(caDER))

	signedCBOR, err := s.Sign(signer)
	require.NoError(t, err)

	return x5chainPKIFixture{
		signedCBOR: signedCBOR,
		anchorDER:  caDER,
		ca:         ca,
		caKey:      caKey,
		leafSerial: big.NewInt(2),
	}
}

type x5chainPKIFixture struct {
	signedCBOR []byte
	anchorDER  []byte
	ca         *x509.Certificate
	caKey      *ecdsa.PrivateKey
	leafSerial *big.Int
}

func (f x5chainPKIFixture) emptyCRL(t *testing.T) []byte {
	t.Helper()

	crlDER, err := x509.CreateRevocationList(rand.Reader, &x509.RevocationList{
		Number:     big.NewInt(1),
		ThisUpdate: time.Now().Add(-time.Minute),
		NextUpdate: time.Now().Add(time.Hour),
	}, f.ca, f.caKey)
	require.NoError(t, err)

	return crlDER
}

func (f x5chainPKIFixture) expiredCRL(t *testing.T) []byte {
	t.Helper()

	crlDER, err := x509.CreateRevocationList(rand.Reader, &x509.RevocationList{
		Number:     big.NewInt(1),
		ThisUpdate: time.Now().Add(-2 * time.Hour),
		NextUpdate: time.Now().Add(-time.Hour),
	}, f.ca, f.caKey)
	require.NoError(t, err)

	return crlDER
}

func (f x5chainPKIFixture) crlWithRevokedLeaf(t *testing.T) []byte {
	t.Helper()

	crlDER, err := x509.CreateRevocationList(rand.Reader, &x509.RevocationList{
		Number:     big.NewInt(1),
		ThisUpdate: time.Now().Add(-time.Minute),
		NextUpdate: time.Now().Add(time.Hour),
		RevokedCertificateEntries: []x509.RevocationListEntry{
			{
				SerialNumber:   f.leafSerial,
				RevocationTime: time.Now().Add(-time.Minute),
			},
		},
	}, f.ca, f.caKey)
	require.NoError(t, err)

	return crlDER
}

func buildUnrelatedCRL(t *testing.T) (*x509.Certificate, []byte) {
	t.Helper()

	unrelatedKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	unrelatedTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(3),
		Subject:               pkix.Name{CommonName: "Unrelated CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	unrelatedCADER, err := x509.CreateCertificate(
		rand.Reader, unrelatedTemplate, unrelatedTemplate, &unrelatedKey.PublicKey, unrelatedKey)
	require.NoError(t, err)

	unrelatedCA, err := x509.ParseCertificate(unrelatedCADER)
	require.NoError(t, err)

	unrelatedCRLDER, err := x509.CreateRevocationList(rand.Reader, &x509.RevocationList{
		Number:     big.NewInt(1),
		ThisUpdate: time.Now().Add(-time.Minute),
		NextUpdate: time.Now().Add(time.Hour),
	}, unrelatedCA, unrelatedKey)
	require.NoError(t, err)

	return unrelatedCA, unrelatedCRLDER
}
