// Copyright 2021-2025 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/veraison/corim/corim"
	"github.com/veraison/corim/cots"
)

var (
	corimDisplayCorimFile *string
	corimDisplayShowTags  *bool
)

var corimDisplayCmd = NewCorimDisplayCmd()

func NewCorimDisplayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "display",
		Short: "display the content of a CoRIM as JSON",
		Long: `display the content of a CoRIM as JSON

	Display the contents of the signed CoRIM signed-corim.cbor 
	
	  cocli corim display --file signed-corim.cbor

	Display the contents of the signed CoRIM yet-another-signed-corim.cbor and
	also unpack any embedded CoMID, CoSWID and CoTS
	
	  cocli corim display --file yet-another-signed-corim.cbor --show-tags
	`,

		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkCorimDisplayArgs(); err != nil {
				return err
			}

			return display(*corimDisplayCorimFile, *corimDisplayShowTags)
		},
	}

	corimDisplayCorimFile = cmd.Flags().StringP("file", "f", "", "a CoRIM file (in CBOR format)")
	corimDisplayShowTags = cmd.Flags().BoolP("show-tags", "v", false, "display embedded tags")

	return cmd
}

func checkCorimDisplayArgs() error {
	if corimDisplayCorimFile == nil || *corimDisplayCorimFile == "" {
		return errors.New("no CoRIM supplied")
	}

	return nil
}

func displaySignedCorim(s corim.SignedCorim, corimFile string, showTags bool) error {
	metaJSON, err := json.MarshalIndent(&s.Meta, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding CoRIM Meta from %s: %w", corimFile, err)
	}

	fmt.Println("Meta:")
	fmt.Println(string(metaJSON))

	corimJSON, err := json.MarshalIndent(&s.UnsignedCorim, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding unsigned CoRIM from %s: %w", corimFile, err)
	}

	fmt.Println("CoRIM:")
	fmt.Println(string(corimJSON))

	if showTags {
		fmt.Println("Tags:")
		displayTags(s.UnsignedCorim.Tags, s.UnsignedCorim.Profile)
	}

	return nil
}

func displayUnsignedCorim(u corim.UnsignedCorim, corimFile string, showTags bool) error {
	corimJSON, err := json.MarshalIndent(&u, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding unsigned CoRIM from %s: %w", corimFile, err)
	}

	fmt.Println("Corim:")
	fmt.Println(string(corimJSON))

	if showTags {
		fmt.Println("Tags:")
		displayTags(u.Tags, u.Profile)
	}

	return nil
}

func display(corimFile string, showTags bool) error {
	var (
		corimCBOR []byte
		err       error
	)

	// read the CoRIM file
	if corimCBOR, err = afero.ReadFile(fs, corimFile); err != nil {
		return fmt.Errorf("error loading CoRIM from %s: %w", corimFile, err)
	}

	// try to decode as a signed CoRIM
	var s corim.SignedCorim
	if err = s.FromCOSE(corimCBOR); err == nil {
		// successfully decoded as signed CoRIM
		return displaySignedCorim(s, corimFile, showTags)
	}

	// if decoding as signed CoRIM failed, attempt to decode as unsigned CoRIM
	var u corim.UnsignedCorim
	if err = u.FromCBOR(corimCBOR); err != nil {
		return fmt.Errorf("error decoding CoRIM (signed or unsigned) from %s: %w", corimFile, err)
	}

	// successfully decoded as unsigned CoRIM
	return displayUnsignedCorim(u, corimFile, showTags)
}

// displayTags processes and displays embedded tags within a CoRIM.
func displayTags(tags []corim.Tag, p *corim.Profile) {
	for i, t := range tags {
		hdr := fmt.Sprintf(">> [ %d ]", i)

		switch t.Number {
		case corim.ComidTag:
			if err := printComid(t.Content, p, hdr); err != nil {
				fmt.Printf(">> skipping malformed CoMID tag at index %d: %v\n", i, err)
			}
		case corim.CoswidTag:
			if err := printCoswid(t.Content, hdr); err != nil {
				fmt.Printf(">> skipping malformed CoSWID tag at index %d: %v\n", i, err)
			}
		case cots.CotsTag:
			if err := printCots(t.Content, hdr); err != nil {
				fmt.Printf(">> skipping malformed CoTS tag at index %d: %v\n", i, err)
			}
		default:
			fmt.Printf(">> unmatched CBOR tag: %d\n", t.Number)
		}
	}
}

func init() {
	corimCmd.AddCommand(corimDisplayCmd)
}
