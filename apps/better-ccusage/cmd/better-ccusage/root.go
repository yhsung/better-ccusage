package main

import (
	"bytes"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/commands"
	"github.com/cobra91/better-ccusage/pkg/pricing"
	"github.com/cobra91/better-ccusage/pkg/terminal"
	"github.com/spf13/cobra"
)

func newRootCmd(log *terminal.Logger) *cobra.Command {
	prices, err := pricing.LoadPrices(bytes.NewReader(pricing.EmbeddedPrices))
	if err != nil {
		log.Warn("failed to load embedded pricing: %v", err)
	}

	root := &cobra.Command{
		Use:     "better-ccusage",
		Short:   "Analyze Claude Code usage from local JSONL files",
		Version: version,
	}
	daily, _ := commands.NewDailyCmd(prices)
	monthly, _ := commands.NewMonthlyCmd(prices)
	session, _ := commands.NewSessionCmd(prices)
	blocks, _ := commands.NewBlocksCmd(prices)
	blocksLive, _ := commands.NewBlocksLiveCmd(prices)
	statusline, _ := commands.NewStatuslineCmd(prices)
	weekly, _ := commands.NewWeeklyCmd(prices)
	root.AddCommand(
		daily,
		monthly,
		session,
		blocks,
		blocksLive,
		statusline,
		weekly,
	)
	return root
}
