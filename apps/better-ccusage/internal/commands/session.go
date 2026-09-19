package commands

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/adapters"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/config"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/output"
	"github.com/cobra91/better-ccusage/pkg/pricing"
	"github.com/cobra91/better-ccusage/pkg/terminal"
)

// SessionOpts are the session command's options.
type SessionOpts struct {
	CommonOpts
}

// Session is the public Run entry point for the session command.
// Returns the typed result so the MCP server can call it directly.
func Session(ctx context.Context, opts SessionOpts, w io.Writer, prices *pricing.PriceTable) (output.DailyResult, error) {
	cfg, _ := config.Load(config.DefaultPath())
	configDir := opts.ConfigDir
	if configDir == "" {
		configDir = os.Getenv("CLAUDE_CONFIG_DIR")
	}
	dirs := config.ResolveDirs(configDir, cfg)
	loader := data.NewLoader(dirs...)
	entries, err := loader.Load(ctx)
	if err != nil {
		return output.DailyResult{}, fmt.Errorf("loading data: %w", err)
	}
	if opts.Since != nil || opts.Until != nil {
		entries = data.FilterByTime(entries, opts.Since, opts.Until)
	}
	if len(entries) == 0 {
		return output.DailyResult{}, fmt.Errorf("no data in range: %w", errs.ErrNoData)
	}
	entries = adapters.NewManager().Normalize(entries)
	buckets := cost.Aggregate(entries, cost.GroupBySession)
	buckets = cost.ApplyPrices(buckets, prices, opts.Mode)
	rows := output.BucketsToDailyRows(buckets)
	return output.DailyResult{Daily: rows, Summary: output.SumRows(rows)}, nil
}

// NewSessionCmd returns the cobra command for `better-ccusage session` along
// with its bound opts.
//
// BindCommonFlags is called at construction time (not inside RunE) so
// flags are registered before parsing and PreRunE (since/until parsing)
// runs before RunE.
func NewSessionCmd(prices *pricing.PriceTable) (*cobra.Command, *SessionOpts) {
	opts := &SessionOpts{}
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Show session-based usage report",
	}
	common := BindCommonFlags(cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts.CommonOpts = *common
		result, err := Session(cmd.Context(), *opts, cmd.OutOrStdout(), prices)
		if err != nil {
			return err
		}
		if opts.JSON {
			return output.EncodeJSON(cmd.OutOrStdout(), result)
		}
		// Render as table
		cols := []terminal.Column{
			{Header: "Session", Width: 30},
			{Header: "Input", Width: 10},
			{Header: "Output", Width: 10},
			{Header: "Cost", Width: 12, Style: terminal.StyleCost},
		}
		rows := make([][]string, 0, len(result.Daily))
		for _, r := range result.Daily {
			rows = append(rows, []string{r.Date, fmt.Sprint(r.InputTokens), fmt.Sprint(r.OutputTokens), fmt.Sprintf("$%.4f", r.CostUSD)})
		}
		tbl := terminal.NewTable(cmd.OutOrStdout(), cols...)
		tbl.SetRows(rows)
		return tbl.Render()
	}
	return cmd, opts
}
