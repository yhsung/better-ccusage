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
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/statusline"
	"github.com/cobra91/better-ccusage/pkg/pricing"
)

// StatuslineOpts are the statusline command's options.
type StatuslineOpts struct {
	CommonOpts
}

// Statusline is the public Run entry point for the statusline command.
// It aggregates all usage and renders a single compact line for Claude Code.
func Statusline(ctx context.Context, opts StatuslineOpts, w io.Writer, prices *pricing.PriceTable) error {
	cfg, _ := config.Load(config.DefaultPath())
	configDir := opts.ConfigDir
	if configDir == "" {
		configDir = os.Getenv("CLAUDE_CONFIG_DIR")
	}
	dirs := config.ResolveDirs(configDir, cfg)
	loader := data.NewLoader(dirs...)
	entries, err := loader.Load(ctx)
	if err != nil {
		return fmt.Errorf("loading data: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("no data: %w", errs.ErrNoData)
	}
	entries = adapters.NewManager().Normalize(entries)
	buckets := cost.Aggregate(entries, cost.GroupByDay)
	buckets = cost.ApplyPrices(buckets, prices, opts.Mode)
	var totalCost pricing.Money
	var totalTokens int64
	models := map[string]bool{}
	for _, b := range buckets {
		totalCost = totalCost.Add(b.Cost)
		totalTokens += b.InputTokens + b.OutputTokens + b.CacheCreationTokens + b.CacheReadTokens
		models[b.Model] = true
	}
	modelList := make([]string, 0, len(models))
	for m := range models {
		modelList = append(modelList, m)
	}
	out := statusline.Render(statusline.Input{TotalCost: totalCost, TotalTokens: totalTokens, Models: modelList})
	_, err = io.WriteString(w, out+"\n")
	return err
}

// NewStatuslineCmd returns the cobra command for `better-ccusage statusline`
// along with its bound opts.
//
// BindCommonFlags is called at construction time (not inside RunE) so
// flags are registered before parsing and PreRunE (since/until parsing)
// runs before RunE.
func NewStatuslineCmd(prices *pricing.PriceTable) (*cobra.Command, *StatuslineOpts) {
	opts := &StatuslineOpts{}
	cmd := &cobra.Command{
		Use:   "statusline",
		Short: "Render compact statusline for Claude Code",
	}
	common := BindCommonFlags(cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts.CommonOpts = *common
		return Statusline(cmd.Context(), *opts, cmd.OutOrStdout(), prices)
	}
	return cmd, opts
}
