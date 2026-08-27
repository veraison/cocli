// Copyright 2026 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"strings"

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

// coevKindFromOSArgs returns the payload kind by inspecting the first
// positional argument in os.Args (skipping any leading flags). This is
// a workaround because Cobra v1.2.1 only sets CalledAs() on the leaf command,
// so cmd.Parent().CalledAs() is always empty for intermediate commands.
func coevKindFromOSArgs() coevKind {
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if arg == "spdm-toc" {
			return coevKindSpdmToc
		}
		return coevKindConciseEvidence
	}
	return coevKindConciseEvidence
}

var coevCmd = &cobra.Command{
	Use:     "coev",
	Aliases: []string{"spdm-toc"},
	Short:   "CoEV / SPDM-TOC manipulation",

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		coevCurrentKind = coevKindFromOSArgs()
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
