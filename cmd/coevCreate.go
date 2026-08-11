// Copyright 2026 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/veraison/corim/coev"
)

var (
	coevCreateFiles     []string
	coevCreateDirs      []string
	coevCreateOutputDir string
)

var coevCreateCmd = NewCoevCreateCmd()

func NewCoevCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create one or more CBOR-encoded CoEV(s) from the supplied JSON template(s)",
		Long: `create one or more CBOR-encoded CoEV(s) from the supplied JSON template(s)

	A template containing a "tagged-evidence" key is encoded as a tagged-spdm-toc
	(CBOR tag 570). Any other template is encoded as a tagged-concise-evidence
	(CBOR tag 571).

	Create a CoEV from template t1.json and save it to the current directory.

	  cocli coev create --template=t1.json

	Create CoEVs from templates t1.json and t2.json, plus any template found in
	the templates/ directory. Save them to the coevs/ directory.

	  cocli coev create --template=t1.json \
	                    --template=t2.json \
	                    --template-dir=templates \
	                    --output-dir=coevs

	Note: output file names are derived from template file names, so all template
	file names (even from different directories) MUST be different.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkCoevCreateArgs(); err != nil {
				return err
			}

			files := filesList(coevCreateFiles, coevCreateDirs, ".json")
			if len(files) == 0 {
				return errors.New("no files found")
			}

			errs := 0
			for _, tmplFile := range files {
				cborFile, err := coevTemplateToCBOR(tmplFile, coevCreateOutputDir)
				if err != nil {
					fmt.Printf(">> creation failed for %q: %v\n", tmplFile, err)
					errs++
					continue
				}
				fmt.Printf(">> created %q from %q\n", cborFile, tmplFile)
			}

			if errs != 0 {
				return fmt.Errorf("%d/%d creation(s) failed", errs, len(files))
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVarP(
		&coevCreateFiles, "template", "t", []string{}, "a CoEV template file (in JSON format)",
	)

	cmd.Flags().StringArrayVarP(
		&coevCreateDirs, "template-dir", "T", []string{}, "a directory containing CoEV template files",
	)

	cmd.Flags().StringVarP(
		&coevCreateOutputDir, "output-dir", "o", ".", "directory where the created files are stored",
	)

	return cmd
}

func checkCoevCreateArgs() error {
	if len(coevCreateFiles) == 0 && len(coevCreateDirs) == 0 {
		return errors.New("no templates supplied")
	}
	return nil
}

// coevTemplateToCBOR encodes a JSON CoEV template to CBOR.
// Templates with a "tagged-evidence" key are encoded as tagged-spdm-toc
// (tag 570); all others are encoded as tagged-concise-evidence (tag 571).
func coevTemplateToCBOR(tmplFile, outputDir string) (string, error) {
	tmplData, err := afero.ReadFile(fs, tmplFile)
	if err != nil {
		return "", fmt.Errorf("error loading template from %s: %w", tmplFile, err)
	}

	var cborData []byte

	var probe struct {
		TaggedEvidence *json.RawMessage `json:"tagged-evidence"`
	}
	if err := json.Unmarshal(tmplData, &probe); err != nil {
		return "", fmt.Errorf("error parsing template %s: %w", tmplFile, err)
	}

	if probe.TaggedEvidence != nil {
		var toc coev.TaggedSpdmToc
		if err := toc.FromJSON(tmplData); err != nil {
			return "", fmt.Errorf("error decoding SPDM-TOC from %s: %w", tmplFile, err)
		}
		cborData, err = toc.ToCBOR()
		if err != nil {
			return "", fmt.Errorf("error encoding SPDM-TOC from %s: %w", tmplFile, err)
		}
	} else {
		var ce coev.ConciseEvidence
		if err := ce.RegisterExtensions(coev.SpdmExtensionMap()); err != nil {
			return "", fmt.Errorf("error registering extensions for %s: %w", tmplFile, err)
		}
		if err := ce.FromJSON(tmplData); err != nil {
			return "", fmt.Errorf("error decoding CoEV from %s: %w", tmplFile, err)
		}
		tce, err := coev.NewTaggedConciseEvidence(&ce)
		if err != nil {
			return "", fmt.Errorf("error tagging CoEV from %s: %w", tmplFile, err)
		}
		cborData, err = tce.ToCBOR()
		if err != nil {
			return "", fmt.Errorf("error encoding CoEV from %s: %w", tmplFile, err)
		}
	}

	cborFile := makeFileName(outputDir, tmplFile, ".cbor")
	if err := afero.WriteFile(fs, cborFile, cborData, 0644); err != nil {
		return "", fmt.Errorf("error saving CBOR file %s: %w", cborFile, err)
	}

	return cborFile, nil
}

func init() {
	coevCmd.AddCommand(coevCreateCmd)
}
