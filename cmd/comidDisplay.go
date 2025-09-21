// Copyright 2021-2024 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var (
	comidDisplayFiles []string
	comidDisplayDirs  []string
)

var comidDisplayCmd = NewComidDisplayCmd()

func NewComidDisplayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "display",
		Short: "display one or more CBOR-encoded CoMID(s) in human readable (JSON) format",
		Long: `display one or more CBOR-encoded CoMID(s) in human readable (JSON) format.
	You can supply individual CoMID files or directories containing CoMID files.

	Display CoMID in file c.cbor.

	  cocli comid display --file=c.cbor

	Display CoMIDs in files c1.cbor, c2.cbor and any cbor file in the comids/
	directory.
	
	  cocli comid display --file=c1.cbor --file=c2.cbor --dir=comids
	`,

		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkComidDisplayArgs(); err != nil {
				return err
			}

			VerboseInfo("Collecting CoMID files from specified paths")
			filesList := filesList(comidDisplayFiles, comidDisplayDirs, ".cbor")
			if len(filesList) == 0 {
				VerboseInfo("No .cbor files found in specified locations")
				return errors.New("no files found")
			}

			VerboseInfo("Found %d CoMID files to process", len(filesList))

			errs := 0
			for i, file := range filesList {
				VerboseProgress(i+1, len(filesList), "files processed")
				if err := displayComidFile(file); err != nil {
					fmt.Printf(">> failed displaying %q: %v\n", file, err)
					VerboseDebug("Failed to display file %s: %v", file, err)
					errs++
					continue
				}
			}

			if errs != 0 {
				VerboseInfo("Completed with %d failures out of %d files", errs, len(filesList))
				return fmt.Errorf("%d/%d display(s) failed", errs, len(filesList))
			}
			VerboseInfo("Successfully displayed all %d CoMID files", len(filesList))
			return nil
		},
	}

	cmd.Flags().StringArrayVarP(
		&comidDisplayFiles, "file", "f", []string{}, "a CoMID file (in CBOR format)",
	)

	cmd.Flags().StringArrayVarP(
		&comidDisplayDirs, "dir", "d", []string{}, "a directory containing CoMID files (in CBOR format)",
	)

	return cmd
}

func displayComidFile(file string) error {
	var (
		data []byte
		err  error
	)

	VerboseDebug("Reading CoMID file: %s", file)
	if data, err = afero.ReadFile(fs, file); err != nil {
		return fmt.Errorf("error loading CoMID from %s: %w", file, err)
	}

	// Get file stats for verbose output
	if stat, err := fs.Stat(file); err == nil {
		VerboseFileStats(file, stat.Size())
	}

	VerboseTrace("Starting CBOR decoding for file: %s", file)
	VerboseTrace("Raw CBOR data length: %d bytes", len(data))

	// use file name as heading
	return VerboseOperation(fmt.Sprintf("displaying CoMID from %s", file), func() error {
		return printComid(data, ">> ["+file+"]")
	})
}

func checkComidDisplayArgs() error {
	if len(comidDisplayFiles) == 0 && len(comidDisplayDirs) == 0 {
		return errors.New("no files supplied")
	}
	return nil
}

func init() {
	comidCmd.AddCommand(comidDisplayCmd)
}
