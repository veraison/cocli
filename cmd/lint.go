// Copyright 2026 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/veraison/cocli/internal/lint"
)

var (
	lintRules string
	lintFiles []string
	lintDirs  []string
)

var lintCmd = NewLintCmd()

func NewLintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint",
		Short: "check JSON template(s) against a set of lint rules",
		Long: `check JSON template(s) against a set of lint rules

	Check template t.json against the rules in r.yaml.

	  cocli lint --rules=r.yaml --file=t.json

	Check t1.json, t2.json and any json file in the templates/ directory.

	  cocli lint --rules=r.yaml --file=t1.json --file=t2.json --dir=templates

	The rules file is YAML (or JSON).  Each rule selects nodes with a JSONPath
	(RFC 9535) expression and asserts on them.  Rules with "warn" or "info"
	severity are reported but do not cause a non-zero exit status.
	`,

		RunE: func(cmd *cobra.Command, args []string) error {
			if lintRules == "" {
				return errors.New("no rules file supplied")
			}

			if len(lintFiles) == 0 && len(lintDirs) == 0 {
				return errors.New("no files or directories supplied")
			}

			rules, err := loadLintRules(lintRules)
			if err != nil {
				return err
			}

			filesList := filesList(lintFiles, lintDirs, ".json")
			if len(filesList) == 0 {
				return errors.New("no files found")
			}

			failed := 0
			for _, file := range filesList {
				findings, err := lintFile(rules, file)
				if err != nil {
					fmt.Printf("[error] %q: %v\n", file, err)
					failed++
					continue
				}

				for _, f := range findings {
					fmt.Printf("%q: %s\n", file, f)
				}

				if lint.HasErrors(findings) {
					failed++
				}
			}

			if failed != 0 {
				return fmt.Errorf("%d/%d file(s) failed lint", failed, len(filesList))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&lintRules, "rules", "r", "", "a lint rules file (YAML or JSON)")
	cmd.Flags().StringArrayVarP(&lintFiles, "file", "f", []string{}, "a JSON template file")
	cmd.Flags().StringArrayVarP(&lintDirs, "dir", "d", []string{}, "a directory containing JSON template files")

	return cmd
}

func loadLintRules(path string) (*lint.RuleSet, error) {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, fmt.Errorf("reading rules: %w", err)
	}

	return lint.Load(data)
}

func lintFile(rules *lint.RuleSet, path string) ([]lint.Finding, error) {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, err
	}

	doc, err := lint.Parse(data)
	if err != nil {
		return nil, err
	}

	return rules.Check(doc), nil
}

func init() {
	rootCmd.AddCommand(lintCmd)
}
