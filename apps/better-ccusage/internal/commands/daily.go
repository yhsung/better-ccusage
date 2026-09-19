package commands

import (
	"bytes"
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
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/jq"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/output"
	"github.com/cobra91/better-ccusage/pkg/pricing"
	"github.com/cobra91/better-ccusage/pkg/terminal"
)

// DailyOpts are the daily command's options.
type DailyOpts struct {
	CommonOpts
}

// Daily is the public Run entry point for the daily command.
// Returns the typed result so the MCP server can call it directly.
func Daily(ctx context.Context, opts DailyOpts, w io.Writer, prices *pricing.PriceTable) (output.DailyResult, error) {
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
	buckets := cost.Aggregate(entries, cost.GroupByDay)
	buckets = cost.ApplyPrices(buckets, prices, opts.Mode)
	rows := output.BucketsToDailyRows(buckets)
	return output.DailyResult{Daily: rows, Summary: output.SumRows(rows)}, nil
}

// NewDailyCmd returns the cobra command for `better-ccusage daily` along
// with its bound opts.
//
// BindCommonFlags is called at construction time (not inside RunE) so
// flags are registered before parsing and PreRunE (since/until parsing)
// runs before RunE.
func NewDailyCmd(prices *pricing.PriceTable) (*cobra.Command, *DailyOpts) {
	opts := &DailyOpts{}
	cmd := &cobra.Command{
		Use:   "daily",
		Short: "Show daily usage report",
	}
	common := BindCommonFlags(cmd)
	var jqExpr string
	cmd.Flags().StringVar(&jqExpr, "jq", "", "post-process JSON output with a jq expression")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts.CommonOpts = *common
		result, err := Daily(cmd.Context(), *opts, cmd.OutOrStdout(), prices)
		if err != nil {
			return err
		}
		if opts.JSON || jqExpr != "" {
			var buf bytes.Buffer
			if err := output.EncodeJSON(&buf, result); err != nil {
				return err
			}
			out := buf.Bytes()
			if jqExpr != "" {
				filtered, err := jq.Process(jqExpr, out)
				if err != nil {
					return err
				}
				out = filtered
			}
			_, err = cmd.OutOrStdout().Write(ensureTrailingNewline(out))
			return err
		}
		// Render as table
		cols := []terminal.Column{
			{Header: "Date", Width: 12},
			{Header: "Input", Width: 10},
			{Header: "Output", Width: 10},
			{Header: "Cache R", Width: 10},
			{Header: "Cost", Width: 12, Style: terminal.StyleCost},
		}
		rows := make([][]string, 0, len(result.Daily))
		for _, r := range result.Daily {
			rows = append(rows, []string{r.Date, fmt.Sprint(r.InputTokens), fmt.Sprint(r.OutputTokens), fmt.Sprint(r.CacheReadTokens), fmt.Sprintf("$%.4f", r.CostUSD)})
		}
		tbl := terminal.NewTable(cmd.OutOrStdout(), cols...)
		tbl.SetRows(rows)
		return tbl.Render()
	}
	return cmd, opts
}

// ensureTrailingNewline appends a newline if out doesn't end with one.
// EncodeJSON output already ends with "\n"; jq.Process output does not.
func ensureTrailingNewline(out []byte) []byte {
	if len(out) == 0 || out[len(out)-1] == '\n' {
		return out
	}
	return append(out, '\n')
}
