// Copyright 2021-2026 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/veraison/corim/corim"
)

var (
	comidValidateFiles   []string
	comidValidateDirs    []string
	comidValidateProfile string
)

var comidValidateCmd = NewComidValidateCmd()

func NewComidValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "validate one or more CBOR-encoded CoMID(s)",
		Long: `validate one or more CBOR-encoded CoMID(s)

	Validate CoMID in file c.cbor.

	  cocli comid validate --file=c.cbor

	Validate CoMIDs in files c1.cbor, c2.cbor and any cbor file in the comids/
	directory.
	
	  cocli comid validate --file=c1.cbor --file=c2.cbor --dir=comids
	`,

		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkComidValidateArgs(); err != nil {
				return err
			}

			filesList := filesList(comidValidateFiles, comidValidateDirs, ".cbor")
			if len(filesList) == 0 {
				return errors.New("no files found")
			}

			errs := 0
			for _, file := range filesList {
				err := validateComid(file)
				if err != nil {
					fmt.Printf("[invalid] %q: %v\n", file, err)
					errs++
					continue
				}
				fmt.Printf("[valid] %q\n", file)
			}

			if errs != 0 {
				return fmt.Errorf("%d/%d validation(s) failed", errs, len(filesList))
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVarP(
		&comidValidateFiles, "file", "f", []string{}, "a CoMID file (in CBOR format)",
	)

	cmd.Flags().StringVarP(
		&comidValidateProfile, "profile", "p", "", "an optional, scheme-specific profile applicable to all CoMID files",
	)

	cmd.Flags().StringArrayVarP(
		&comidValidateDirs, "dir", "d", []string{}, "a directory containing CoMID files (in CBOR format)",
	)

	return cmd
}

func validateComid(file string) error {
	var (
		data []byte
		err  error
		p    *corim.Profile
	)

	if data, err = afero.ReadFile(fs, file); err != nil {
		return fmt.Errorf("error loading CoMID from %s: %w", file, err)
	}

	if comidValidateProfile != "" {
		p, err = corim.NewProfileFromString(comidValidateProfile)
		if err != nil {
			return fmt.Errorf("error creating profile %q for CoMID: %w", comidValidateProfile, err)
		}
	}

	c, err := corim.UnmarshalComidFromCBOR(data, p)
	if err != nil {
		return fmt.Errorf("error decoding CoMID from %s: %w", file, err)
	}

	if err = c.Valid(); err != nil {
		return fmt.Errorf("error validating CoMID %s: %w", file, err)
	}

	return nil
}

func checkComidValidateArgs() error {
	if len(comidValidateFiles) == 0 && len(comidValidateDirs) == 0 {
		return errors.New("no files supplied")
	}
	return nil
}

func init() {
	comidCmd.AddCommand(comidValidateCmd)
}
