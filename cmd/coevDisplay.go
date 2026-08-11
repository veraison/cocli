// Copyright 2026 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/veraison/corim/coev"
)

var (
	coevDisplayFiles []string
	coevDisplayDirs  []string
)

var coevDisplayCmd = NewCoevDisplayCmd()

func NewCoevDisplayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "display",
		Short: "display one or more CBOR-encoded CoEV(s) in human readable (JSON) format",
		Long: `display one or more CBOR-encoded CoEV(s) in human readable (JSON) format.

	The format is auto-detected from the CBOR tag:
	  CBOR tag 570: tagged-spdm-toc
	  CBOR tag 571: tagged-concise-evidence
	  No tag: untagged concise-evidence

	Display CoEV in file e.cbor.

	  cocli coev display --file=e.cbor

	Display CoEVs in files e1.cbor, e2.cbor and any cbor file in the coevs/ directory.

	  cocli coev display --file=e1.cbor --file=e2.cbor --dir=coevs
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkCoevDisplayArgs(); err != nil {
				return err
			}

			files := filesList(coevDisplayFiles, coevDisplayDirs, ".cbor")
			if len(files) == 0 {
				return errors.New("no files found")
			}

			errs := 0
			for _, file := range files {
				if err := displayCoevFile(file); err != nil {
					fmt.Printf(">> failed displaying %q: %v\n", file, err)
					errs++
					continue
				}
			}

			if errs != 0 {
				return fmt.Errorf("%d/%d display(s) failed", errs, len(files))
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVarP(
		&coevDisplayFiles, "file", "f", []string{}, "a CoEV file (in CBOR format)",
	)

	cmd.Flags().StringArrayVarP(
		&coevDisplayDirs, "dir", "d", []string{}, "a directory containing CoEV files (in CBOR format)",
	)

	return cmd
}

func checkCoevDisplayArgs() error {
	if len(coevDisplayFiles) == 0 && len(coevDisplayDirs) == 0 {
		return errors.New("no files supplied")
	}
	return nil
}

func displayCoevFile(file string) error {
	data, err := afero.ReadFile(fs, file)
	if err != nil {
		return fmt.Errorf("error loading CoEV from %s: %w", file, err)
	}

	j, err := coevCBORToJSON(data)
	if err != nil {
		return err
	}

	return printCoevJSON(j, ">> ["+file+"]")
}

// coevCBORToJSON decodes a CBOR CoEV payload to JSON, auto-detecting the format
// from the leading CBOR tag (570 = tagged-spdm-toc, 571 = tagged-concise-evidence,
// untagged = bare concise-evidence).
func coevCBORToJSON(data []byte) ([]byte, error) {
	if bytes.HasPrefix(data, coev.SpdmTocTag) {
		var toc coev.TaggedSpdmToc
		if err := toc.FromCBOR(data); err != nil {
			return nil, fmt.Errorf("decoding SPDM-TOC: %w", err)
		}
		return toc.ToJSON()
	}

	if bytes.HasPrefix(data, coev.ConciseEvidenceTag) {
		data = data[3:]
	}

	var ce coev.ConciseEvidence
	if err := ce.RegisterExtensions(coev.SpdmExtensionMap()); err != nil {
		return nil, fmt.Errorf("registering extensions: %w", err)
	}
	if err := ce.FromCBOR(data); err != nil {
		return nil, fmt.Errorf("decoding ConciseEvidence: %w", err)
	}
	return ce.ToJSON()
}

func printCoevJSON(j []byte, heading string) error {
	var buf bytes.Buffer
	if err := json.Indent(&buf, j, "", "  "); err != nil {
		return fmt.Errorf("JSON formatting failed: %w", err)
	}
	fmt.Println(heading)
	fmt.Println(buf.String())
	return nil
}

func init() {
	coevCmd.AddCommand(coevDisplayCmd)
}
