// Copyright 2021-2024 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"crypto/x509"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/veraison/corim/corim"
	cose "github.com/veraison/go-cose"
)

var (
	corimVerifyCorimFile *string
	corimVerifyKeyFile   *string
)

var corimVerifyCmd = NewCorimVerifyCmd()

func NewCorimVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "verify a signed CoRIM using the supplied key or x5chain",
		Long: `verify a signed CoRIM using the supplied key or x5chain

	Verify with a JWK public key:

	  cocli corim verify --file=signed-corim.cbor --key=key.jwk

	Verify using the x5chain header (signing cert at [0], intermediates at [1..];
	trust anchor via --trust-anchors, not in the header). When signing with cocli,
	use --cert; intermediate certificates can be embedded with sign --intermediates
	or supplied later with verify --intermediates:

	  cocli corim verify --file=signed-x5chain.cbor --trust-anchors=cmd/testcases/test-certs/rootCA.der
	  cocli corim verify --file=signed-x5chain.cbor --trust-anchors=cmd/testcases/test-certs/rootCA.der --crl=issuer.crl
	  cocli corim verify --file=signed-x5chain.cbor --trust-anchors=cmd/testcases/test-certs/rootCA.der \
	    --crl=issuer.crl --crl-policy=permissive
	  cocli corim verify --file=signed-x5chain.cbor --trust-anchors=root.pem \
	    --intermediates=issuer.pem --crl=issuer.crl --revocation-mode=leaf-only

	When --trust-anchors is omitted, the OS trust store is used (with a warning;
	not recommended for production CoRIM trust decisions). When supplied, only
	those anchors are trusted (override). --crl-policy requires --crl. When
	--crl is supplied, --crl-policy selects strict (every checked certificate must
	have a valid matching issuer CRL) or permissive (skip missing issuer CRLs).
	--revocation-mode selects full-chain (all non-anchor certificates), leaf-only,
	or disabled (skip revocation checks). Supplied CRL files must still be readable
	and parseable, even when checks are disabled. If omitted, full-chain is used
	with --crl and disabled otherwise. Explicit full-chain or leaf-only requires --crl.
	--intermediates supplies additional certificates to complete the chain;
	these certificates are not trust anchors.
	--key cannot be combined with --trust-anchors, --crl, --revocation-mode,
	or --intermediates. If --key is used on a
	CoRIM that also has an x5chain header, PKIX verification is skipped (warning).
	`,

		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := parseVerifyFlags(cmd)
			if err != nil {
				return err
			}

			if err := verify(opts, cmd.ErrOrStderr()); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), ">> %q verified\n", opts.corimFile)

			return nil
		},
	}

	corimVerifyCorimFile = cmd.Flags().StringP("file", "f", "", "a signed CoRIM file (in CBOR format)")
	corimVerifyKeyFile = cmd.Flags().StringP("key", "k", "",
		"verification key in JWK format; optional when the CoRIM has an x5chain header")
	cmd.Flags().StringArray("trust-anchors", nil,
		"path to trust-anchor certificate (DER or PEM; PEM may bundle multiple); repeatable")
	cmd.Flags().StringArray("crl", nil,
		"path to CRL (DER or PEM; PEM may contain multiple); repeatable")
	cmd.Flags().String("crl-policy", "strict",
		"CRL behaviour when --crl is supplied: strict or permissive")
	cmd.Flags().String("revocation-mode", "",
		"full-chain, leaf-only, or disabled; default: full-chain with --crl, disabled otherwise")
	cmd.Flags().StringArray("intermediates", nil,
		"path to additional intermediate certificate (DER or PEM bundle); repeatable; does not grant trust")
	cmd.Flags().SortFlags = false

	return cmd
}

type verifyOptions struct {
	corimFile         string
	keyFile           string
	trustAnchorFiles  []string
	crlFiles          []string
	crlPolicy         cose.CRLPolicy
	revocationMode    cose.RevocationMode
	intermediateFiles []string
}

func parseVerifyFlags(cmd *cobra.Command) (verifyOptions, error) {
	var opts verifyOptions

	opts.corimFile = *corimVerifyCorimFile
	if opts.corimFile == "" {
		return opts, errors.New("no CoRIM supplied")
	}

	opts.keyFile = *corimVerifyKeyFile

	trustAnchorFiles, err := cmd.Flags().GetStringArray("trust-anchors")
	if err != nil {
		return opts, err
	}
	opts.trustAnchorFiles = trustAnchorFiles

	crlFiles, err := cmd.Flags().GetStringArray("crl")
	if err != nil {
		return opts, err
	}
	opts.crlFiles = crlFiles

	opts.intermediateFiles, err = cmd.Flags().GetStringArray("intermediates")
	if err != nil {
		return opts, err
	}

	crlPolicyStr, err := cmd.Flags().GetString("crl-policy")
	if err != nil {
		return opts, err
	}

	// Check --crl presence before parsing policy so invalid names do not mask this error.
	if cmd.Flags().Changed("crl-policy") && len(crlFiles) == 0 {
		return opts, errors.New("cannot use --crl-policy without --crl")
	}

	opts.crlPolicy, err = parseCrlPolicy(crlPolicyStr)
	if err != nil {
		return opts, err
	}

	if opts.keyFile != "" {
		if err := validateKeyPathFlags(opts); err != nil {
			return opts, err
		}
		if cmd.Flags().Changed("revocation-mode") {
			return opts, errors.New("cannot use --revocation-mode with --key")
		}
	}

	// Preserve the CLI's opt-in CRL checking when the mode is omitted.
	opts.revocationMode = cose.RevocationDisabled
	if len(crlFiles) > 0 {
		opts.revocationMode = cose.RevocationFullChain
	}
	if cmd.Flags().Changed("revocation-mode") {
		mode, flagErr := cmd.Flags().GetString("revocation-mode")
		if flagErr != nil {
			return opts, flagErr
		}
		opts.revocationMode, err = parseRevocationMode(mode)
		if err != nil {
			return opts, err
		}
		if opts.revocationMode != cose.RevocationDisabled && len(crlFiles) == 0 {
			return opts, errors.New("--revocation-mode full-chain or leaf-only requires --crl")
		}
	}

	return opts, nil
}

func validateKeyPathFlags(opts verifyOptions) error {
	if len(opts.trustAnchorFiles) > 0 {
		return errors.New("cannot use --trust-anchors with --key")
	}

	if len(opts.crlFiles) > 0 {
		return errors.New("cannot use --crl with --key")
	}
	if len(opts.intermediateFiles) > 0 {
		return errors.New("cannot use --intermediates with --key")
	}

	return nil
}

func parseRevocationMode(value string) (cose.RevocationMode, error) {
	switch value {
	case "full-chain":
		return cose.RevocationFullChain, nil
	case "leaf-only":
		return cose.RevocationLeafOnly, nil
	case "disabled":
		return cose.RevocationDisabled, nil
	default:
		return 0, fmt.Errorf("invalid revocation mode %q (want full-chain, leaf-only, or disabled)", value)
	}
}

func parseCrlPolicy(value string) (cose.CRLPolicy, error) {
	switch value {
	case "", "strict":
		return cose.CRLPolicyStrict, nil
	case "permissive":
		return cose.CRLPolicyPermissive, nil
	default:
		return 0, fmt.Errorf("invalid CRL policy %q (want strict or permissive)", value)
	}
}

func verify(opts verifyOptions, stderr io.Writer) error {
	var (
		signedCorimCBOR []byte
		err             error
		s               corim.SignedCorim
	)

	if signedCorimCBOR, err = afero.ReadFile(fs, opts.corimFile); err != nil {
		return fmt.Errorf("error loading signed CoRIM from %s: %w", opts.corimFile, err)
	}

	if err = s.FromCOSE(signedCorimCBOR); err != nil {
		return fmt.Errorf("error decoding signed CoRIM from %s: %w", opts.corimFile, err)
	}

	if opts.keyFile != "" {
		return verifyWithKey(opts.corimFile, opts.keyFile, &s, stderr)
	}

	if s.SigningCert == nil {
		return errors.New("no verification method: supply --key or use a CoRIM with x5chain header")
	}

	return verifyWithX5Chain(opts, &s, stderr)
}

func verifyWithX5Chain(
	opts verifyOptions,
	s *corim.SignedCorim,
	stderr io.Writer,
) error {
	anchors, err := cose.LoadTrustAnchors(func(path string) ([]byte, error) {
		return afero.ReadFile(fs, path)
	}, opts.trustAnchorFiles, opts.crlFiles)
	if err != nil {
		return fmt.Errorf("error loading trust material for %s: %w", opts.corimFile, err)
	}

	if len(opts.trustAnchorFiles) == 0 {
		anchors.UseSystemRoots = true
		_, _ = fmt.Fprintln(stderr,
			"warning: --trust-anchors not supplied; using OS trust store (not recommended for production)")
	}

	anchors.RevocationMode = opts.revocationMode
	if opts.revocationMode == cose.RevocationDisabled && len(opts.crlFiles) > 0 {
		_, _ = fmt.Fprintln(stderr, "warning: --revocation-mode=disabled; supplied CRLs are not used for revocation checks")
	}
	verifyOpts := &cose.X5ChainVerifyOptions{CRLPolicy: opts.crlPolicy}
	if len(opts.intermediateFiles) > 0 {
		verifyOpts.AdditionalIntermediates, err = loadVerificationIntermediates(opts.intermediateFiles)
		if err != nil {
			return fmt.Errorf("error loading intermediate certificates for %s: %w", opts.corimFile, err)
		}
	}

	if err = s.VerifyWithX5Chain(anchors, verifyOpts); err != nil {
		return fmt.Errorf("error verifying %s using x5chain: %w", opts.corimFile, err)
	}

	return nil
}

func loadVerificationIntermediates(paths []string) (*x509.CertPool, error) {
	// Reuse go-cose's DER/PEM certificate loader, adding the result only to the
	// intermediate pool so these certificates cannot become trust anchors.
	pool := x509.NewCertPool()
	for _, path := range paths {
		data, err := afero.ReadFile(fs, path)
		if err != nil {
			return nil, fmt.Errorf("loading intermediate certificates from %s: %w", path, err)
		}
		certs, err := cose.LoadTrustAnchors(func(string) ([]byte, error) {
			return data, nil
		}, []string{path}, nil)
		if err != nil {
			// Replace the loader's trust-anchor file context, retaining its parse error.
			cause := errors.Unwrap(err)
			if cause == nil {
				cause = err
			}
			return nil, fmt.Errorf("parsing intermediate certificates from %s: %w", path, cause)
		}
		for _, cert := range certs.Anchors {
			pool.AddCert(cert)
		}
	}
	return pool, nil
}

func verifyWithKey(signedCorimFile, keyFile string, s *corim.SignedCorim, stderr io.Writer) error {
	keyJWK, err := afero.ReadFile(fs, keyFile)
	if err != nil {
		return fmt.Errorf("error loading verifying key from %s: %w", keyFile, err)
	}

	pkey, err := corim.NewPublicKeyFromJWK(keyJWK)
	if err != nil {
		return fmt.Errorf("error loading verifying key from %s: %w", keyFile, err)
	}

	if s.SigningCert != nil {
		_, _ = fmt.Fprintln(stderr, "warning: using --key; x5chain/PKIX verification skipped")
	}

	if err = s.Verify(pkey); err != nil {
		return fmt.Errorf("error verifying %s with key %s: %w", signedCorimFile, keyFile, err)
	}

	return nil
}

func init() {
	corimCmd.AddCommand(corimVerifyCmd)
}
