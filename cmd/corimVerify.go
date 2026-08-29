// Copyright 2021-2024 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/veraison/corim/corim"
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
	trust anchor via --trust-anchors, not in the header). CoRIM must be signed with
	--cert / --intermediates:

	  cocli corim verify --file=signed-x5chain.cbor --trust-anchors=cmd/testcases/test-certs/rootCA.der
	  cocli corim verify --file=signed-x5chain.cbor --trust-anchors=cmd/testcases/test-certs/rootCA.der --crl=issuer.crl
	  cocli corim verify --file=signed-x5chain.cbor --trust-anchors=cmd/testcases/test-certs/rootCA.der \
	    --crl=issuer.crl --crl-policy=permissive

	When --trust-anchors is omitted, the OS trust store is used (with a warning;
	not recommended for production CoRIM trust decisions). When supplied, only
	those anchors are trusted (override). --crl-policy requires --crl. When
	--crl is supplied, --crl-policy selects strict (every in-chain issuer must have
	a valid matching CRL) or permissive (skip issuers with no matching CRL).
	--key cannot be combined with --trust-anchors or --crl. If --key is used on a
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
	cmd.Flags().SortFlags = false

	return cmd
}

type verifyOptions struct {
	corimFile        string
	keyFile          string
	trustAnchorFiles []string
	crlFiles         []string
	crlPolicy        corim.CrlPolicy
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

	return nil
}

func parseCrlPolicy(value string) (corim.CrlPolicy, error) {
	switch value {
	case "", "strict":
		return corim.CrlPolicyStrict, nil
	case "permissive":
		return corim.CrlPolicyPermissive, nil
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

	// strip ASN header bytes if present (d9 01 f4 d9 01 f6)
	signedCorimCBOR = stripASNHeaderBytes(signedCorimCBOR)

	if err = s.FromCOSE(signedCorimCBOR); err != nil {
		return fmt.Errorf("error decoding signed CoRIM from %s: %w", opts.corimFile, err)
	}

	if opts.keyFile != "" {
		return verifyWithKey(opts.corimFile, opts.keyFile, &s, stderr)
	}

	if s.SigningCert == nil {
		return errors.New("no verification method: supply --key or use a CoRIM with x5chain header")
	}

	return verifyWithX5Chain(
		opts.corimFile, opts.trustAnchorFiles, opts.crlFiles, opts.crlPolicy, &s, stderr)
}

func verifyWithX5Chain(
	signedCorimFile string,
	trustAnchorFiles, crlFiles []string,
	crlPolicy corim.CrlPolicy,
	s *corim.SignedCorim,
	stderr io.Writer,
) error {
	anchors, err := corim.LoadTrustAnchors(func(path string) ([]byte, error) {
		return afero.ReadFile(fs, path)
	}, trustAnchorFiles, crlFiles)
	if err != nil {
		return fmt.Errorf("error loading trust material for %s: %w", signedCorimFile, err)
	}

	if len(trustAnchorFiles) == 0 {
		_, _ = fmt.Fprintln(stderr,
			"warning: --trust-anchors not supplied; using OS trust store (not recommended for production)")
	}

	anchors.CrlPolicy = crlPolicy

	if err = s.VerifyWithX5Chain(anchors); err != nil {
		return fmt.Errorf("error verifying %s using x5chain: %w", signedCorimFile, err)
	}

	return nil
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
