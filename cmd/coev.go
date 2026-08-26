// Copyright 2026 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

type coevKind int

const (
	coevKindConciseEvidence coevKind = iota
	coevKindSpdmToc
)

// coevCurrentKind is set at runtime by coevCmd's PersistentPreRun based on the
// alias used to invoke the command. Tests may set it directly.
var coevCurrentKind = coevKindConciseEvidence

var coevCmd = &cobra.Command{
	Use:     "coev",
	Aliases: []string{"spdm-toc"},
	Short:   "CoEV / SPDM-TOC manipulation",

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Parent() != nil && cmd.Parent().CalledAs() == "spdm-toc" {
			coevCurrentKind = coevKindSpdmToc
		} else {
			coevCurrentKind = coevKindConciseEvidence
		}
	},

	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help() // nolint: errcheck
			os.Exit(0)
		}
	},
}

func init() {
	rootCmd.AddCommand(coevCmd)
}
