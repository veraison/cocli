// Copyright 2021-2024 Contributors to the Veraison project.
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
	VerboseDebug("Extracting Meta information from signed CoRIM")
	metaJSON, err := json.MarshalIndent(&s.Meta, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding CoRIM Meta from %s: %w", corimFile, err)
	}

	VerboseTrace("Meta JSON size: %d bytes", len(metaJSON))
	fmt.Println("Meta:")
	fmt.Println(string(metaJSON))

	VerboseDebug("Extracting unsigned CoRIM content")
	corimJSON, err := json.MarshalIndent(&s.UnsignedCorim, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding unsigned CoRIM from %s: %w", corimFile, err)
	}

	VerboseTrace("CoRIM JSON size: %d bytes", len(corimJSON))
	fmt.Println("CoRIM:")
	fmt.Println(string(corimJSON))

	if showTags {
		VerboseInfo("Displaying embedded tags (%d total)", len(s.UnsignedCorim.Tags))
		fmt.Println("Tags:")
		displayTags(s.UnsignedCorim.Tags)
	} else {
		VerboseDebug("Skipping tag display (show-tags not enabled)")
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
		displayTags(u.Tags)
	}

	return nil
}

func display(corimFile string, showTags bool) error {
	var (
		corimCBOR []byte
		err       error
	)

	VerboseInfo("Processing CoRIM file: %s", corimFile)
	VerboseDebug("Show tags mode: %t", showTags)

	// read the CoRIM file
	VerboseDebug("Reading CoRIM file from disk")
	if corimCBOR, err = afero.ReadFile(fs, corimFile); err != nil {
		return fmt.Errorf("error loading CoRIM from %s: %w", corimFile, err)
	}

	// Get file stats for verbose output
	if stat, err := fs.Stat(corimFile); err == nil {
		VerboseFileStats(corimFile, stat.Size())
	}

	VerboseTrace("Original CBOR data length: %d bytes", len(corimCBOR))

	// strip ASN header bytes if present (d9 01 f4 d9 01 f6)
	originalLen := len(corimCBOR)
	corimCBOR = stripASNHeaderBytes(corimCBOR)
	if len(corimCBOR) != originalLen {
		VerboseInfo("Stripped ASN header bytes (%d bytes removed)", originalLen-len(corimCBOR))
	} else {
		VerboseDebug("No ASN header bytes detected")
	}

	VerboseTrace("Processing CBOR data length: %d bytes", len(corimCBOR))

	// try to decode as a signed CoRIM
	VerboseDebug("Attempting to decode as signed CoRIM (COSE format)")
	var s corim.SignedCorim
	if err = s.FromCOSE(corimCBOR); err == nil {
		VerboseInfo("Successfully decoded as signed CoRIM")
		VerboseDebug("CoRIM has %d tags", len(s.UnsignedCorim.Tags))
		// successfully decoded as signed CoRIM
		return displaySignedCorim(s, corimFile, showTags)
	}

	VerboseDebug("Failed to decode as signed CoRIM: %v", err)
	VerboseDebug("Attempting to decode as unsigned CoRIM (CBOR format)")

	// if decoding as signed CoRIM failed, attempt to decode as unsigned CoRIM
	var u corim.UnsignedCorim
	if err = u.FromCBOR(corimCBOR); err != nil {
		VerboseDebug("Failed to decode as unsigned CoRIM: %v", err)
		return fmt.Errorf("error decoding CoRIM (signed or unsigned) from %s: %w", corimFile, err)
	}

	VerboseInfo("Successfully decoded as unsigned CoRIM")
	VerboseDebug("CoRIM has %d tags", len(u.Tags))

	// successfully decoded as unsigned CoRIM
	return displayUnsignedCorim(u, corimFile, showTags)
}

// displayTags processes and displays embedded tags within a CoRIM.
func displayTags(tags []corim.Tag) {
	for i, t := range tags {
		hdr := fmt.Sprintf(">> [ %d ]", i)
		VerboseProgress(i+1, len(tags), "tags processed")

		switch t.Number {
		case corim.ComidTag:
			VerboseDebug("Processing CoMID tag at index %d (content size: %d bytes)", i, len(t.Content))
			if err := printComid(t.Content, hdr); err != nil {
				fmt.Printf(">> skipping malformed CoMID tag at index %d: %v\n", i, err)
				VerboseDebug("CoMID tag parsing failed: %v", err)
			} else {
				VerboseTrace("Successfully displayed CoMID tag at index %d", i)
			}
		case corim.CoswidTag:
			VerboseDebug("Processing CoSWID tag at index %d (content size: %d bytes)", i, len(t.Content))
			if err := printCoswid(t.Content, hdr); err != nil {
				fmt.Printf(">> skipping malformed CoSWID tag at index %d: %v\n", i, err)
				VerboseDebug("CoSWID tag parsing failed: %v", err)
			} else {
				VerboseTrace("Successfully displayed CoSWID tag at index %d", i)
			}
		case cots.CotsTag:
			VerboseDebug("Processing CoTS tag at index %d (content size: %d bytes)", i, len(t.Content))
			if err := printCots(t.Content, hdr); err != nil {
				fmt.Printf(">> skipping malformed CoTS tag at index %d: %v\n", i, err)
				VerboseDebug("CoTS tag parsing failed: %v", err)
			} else {
				VerboseTrace("Successfully displayed CoTS tag at index %d", i)
			}
		default:
			VerboseDebug("Encountered unmatched CBOR tag: %d at index %d", t.Number, i)
			fmt.Printf(">> unmatched CBOR tag: %d\n", t.Number)
		}
	}
}

func init() {
	corimCmd.AddCommand(corimDisplayCmd)
}
