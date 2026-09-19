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

// BlocksOpts are the blocks command's options.
type BlocksOpts struct {
	CommonOpts
	Active     bool
	Recent     bool
	TokenLimit int64
}

// Blocks is the public Run entry point for the blocks command.
// Returns the typed result so the MCP server can call it directly.
//
// Static snapshot: Active/Recent/TokenLimit are parsed and stored but
// filtering/projection is a later task; the pipeline stays verbatim.
func Blocks(ctx context.Context, opts BlocksOpts, w io.Writer, prices *pricing.PriceTable) (output.DailyResult, error) {
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
	buckets := cost.Aggregate(entries, cost.GroupByBlock)
	buckets = cost.ApplyPrices(buckets, prices, opts.Mode)
	rows := output.BucketsToDailyRows(buckets)
	return output.DailyResult{Daily: rows, Summary: output.SumRows(rows)}, nil
}

// NewBlocksCmd returns the cobra command for `better-ccusage blocks` along
// with its bound opts.
//
// BindCommonFlags is called at construction time (not inside RunE) so
// flags are registered before parsing and PreRunE (since/until parsing)
// runs before RunE.
func NewBlocksCmd(prices *pricing.PriceTable) (*cobra.Command, *BlocksOpts) {
	opts := &BlocksOpts{}
	cmd := &cobra.Command{
		Use:   "blocks",
		Short: "Show 5-hour billing blocks",
	}
	common := BindCommonFlags(cmd)
	cmd.Flags().BoolVar(&opts.Active, "active", false, "show only the active block (with projection)")
	cmd.Flags().BoolVar(&opts.Recent, "recent", false, "show blocks from the last 3 days")
	cmd.Flags().Int64Var(&opts.TokenLimit, "token-limit", 0, "warn when a block exceeds this token count")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts.CommonOpts = *common
		result, err := Blocks(cmd.Context(), *opts, cmd.OutOrStdout(), prices)
		if err != nil {
			return err
		}
		if opts.JSON {
			return output.EncodeJSON(cmd.OutOrStdout(), result)
		}
		cols := []terminal.Column{
			{Header: "Block", Width: 18},
			{Header: "Tokens", Width: 12},
			{Header: "Cost", Width: 12, Style: terminal.StyleCost},
		}
		rows := make([][]string, 0, len(result.Daily))
		for _, r := range result.Daily {
			rows = append(rows, []string{r.Date, fmt.Sprint(r.TotalTokens), fmt.Sprintf("$%.4f", r.CostUSD)})
		}
		tbl := terminal.NewTable(cmd.OutOrStdout(), cols...)
		tbl.SetRows(rows)
		return tbl.Render()
	}
	return cmd, opts
}
