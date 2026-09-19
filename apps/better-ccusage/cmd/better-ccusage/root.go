package main

import (
	"github.com/cobra91/better-ccusage/pkg/terminal"
	"github.com/spf13/cobra"
)

func newRootCmd(log *terminal.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "better-ccusage",
		Short:   "Analyze Claude Code usage from local JSONL files",
		Version: version,
	}
}
