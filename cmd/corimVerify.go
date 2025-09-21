// Copyright 2021-2024 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"crypto"
	"errors"
	"fmt"

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
		Short: "verify a signed CoRIM using the supplied key",
		Long: `verify a signed CoRIM using the supplied key

	Verify the signed CoRIM signed-corim.cbor using the key in JWK format from
	file key.jwk
	
	  cocli corim verify --file=signed-corim.cbor --key=key.jwk
	`,

		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkCorimVerifyArgs(); err != nil {
				return err
			}

			// checkCorimVerifyArgs makes sure corimVerifyCorimFile is not nil
			err := verify(*corimVerifyCorimFile, *corimVerifyKeyFile)
			if err != nil {
				return err
			}
			fmt.Printf(">> %q verified\n", *corimVerifyCorimFile)

			return nil
		},
	}

	corimVerifyCorimFile = cmd.Flags().StringP("file", "f", "", "a signed CoRIM file (in CBOR format)")
	corimVerifyKeyFile = cmd.Flags().StringP("key", "k", "", "verification key in JWK format")

	return cmd
}

func checkCorimVerifyArgs() error {
	if corimVerifyCorimFile == nil || *corimVerifyCorimFile == "" {
		return errors.New("no CoRIM supplied")
	}

	if corimVerifyKeyFile == nil || *corimVerifyKeyFile == "" {
		return errors.New("no key supplied")
	}

	return nil
}

func verify(signedCorimFile, keyFile string) error {
	var (
		signedCorimCBOR []byte
		keyJWK          []byte
		err             error
		pkey            crypto.PublicKey
		s               corim.SignedCorim
	)

	VerboseInfo("Starting CoRIM verification process")
	VerboseDebug("Signed CoRIM file: %s", signedCorimFile)
	VerboseDebug("Key file: %s", keyFile)

	VerboseDebug("Reading signed CoRIM file")
	if signedCorimCBOR, err = afero.ReadFile(fs, signedCorimFile); err != nil {
		return fmt.Errorf("error loading signed CoRIM from %s: %w", signedCorimFile, err)
	}

	// Get file stats for verbose output
	if stat, err := fs.Stat(signedCorimFile); err == nil {
		VerboseFileStats(signedCorimFile, stat.Size())
	}

	VerboseTrace("Original signed CoRIM data length: %d bytes", len(signedCorimCBOR))

	// strip ASN header bytes if present (d9 01 f4 d9 01 f6)
	originalLen := len(signedCorimCBOR)
	signedCorimCBOR = stripASNHeaderBytes(signedCorimCBOR)
	if len(signedCorimCBOR) != originalLen {
		VerboseInfo("Stripped ASN header bytes (%d bytes removed)", originalLen-len(signedCorimCBOR))
	} else {
		VerboseDebug("No ASN header bytes detected")
	}

	VerboseDebug("Decoding COSE Sign1 structure")
	VerboseTrace("Processing COSE data length: %d bytes", len(signedCorimCBOR))
	if err = s.FromCOSE(signedCorimCBOR); err != nil {
		VerboseDebug("COSE decoding failed: %v", err)
		return fmt.Errorf("error decoding signed CoRIM from %s: %w", signedCorimFile, err)
	}
	VerboseInfo("Successfully decoded COSE Sign1 structure")

	VerboseDebug("Reading verification key file")
	if keyJWK, err = afero.ReadFile(fs, keyFile); err != nil {
		return fmt.Errorf("error loading verifying key from %s: %w", keyFile, err)
	}

	// Get key file stats
	if stat, err := fs.Stat(keyFile); err == nil {
		VerboseFileStats(keyFile, stat.Size())
	}

	VerboseTrace("JWK data length: %d bytes", len(keyJWK))
	VerboseDebug("Parsing JWK to extract public key")
	if pkey, err = corim.NewPublicKeyFromJWK(keyJWK); err != nil {
		VerboseDebug("JWK parsing failed: %v", err)
		return fmt.Errorf("error loading verifying key from %s: %w", keyFile, err)
	}
	VerboseInfo("Successfully loaded public key from JWK")
	VerboseTrace("Public key type: %T", pkey)

	VerboseInfo("Performing cryptographic signature verification")
	if err = s.Verify(pkey); err != nil {
		VerboseDebug("Signature verification failed: %v", err)
		return fmt.Errorf("error verifying %s with key %s: %w", signedCorimFile, keyFile, err)
	}

	VerboseInfo("Signature verification successful")
	VerboseDebug("CoRIM contains %d embedded tags", len(s.UnsignedCorim.Tags))

	return nil
}

func init() {
	corimCmd.AddCommand(corimVerifyCmd)
}
