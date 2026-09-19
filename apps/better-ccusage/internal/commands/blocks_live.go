package commands

import (
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/config"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/monitor"
	"github.com/cobra91/better-ccusage/pkg/pricing"
)

// NewBlocksLiveCmd returns the cobra command for `better-ccusage blocks.live`
// along with its bound opts.
//
// BindCommonFlags is called at construction time (not inside RunE) so flags
// are registered before parsing and PreRunE (since/until parsing) runs
// before RunE. Live TUI: JSON output is n/a.
func NewBlocksLiveCmd(prices *pricing.PriceTable) (*cobra.Command, *BlocksOpts) {
	opts := &BlocksOpts{}
	cmd := &cobra.Command{
		Use:   "blocks.live",
		Short: "Live-updating 5-hour billing blocks",
	}
	common := BindCommonFlags(cmd)
	cmd.Flags().Int64Var(&opts.TokenLimit, "token-limit", 0, "warn when a block exceeds this token count")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts.CommonOpts = *common
		cfg, _ := config.Load(config.DefaultPath())
		configDir := opts.ConfigDir
		if configDir == "" {
			configDir = os.Getenv("CLAUDE_CONFIG_DIR")
		}
		dirs := config.ResolveDirs(configDir, cfg)
		return monitor.Run(cmd.Context(), monitor.Opts{
			Loader:          data.NewLoader(dirs...),
			Prices:          prices,
			Mode:            opts.Mode,
			RefreshInterval: 30 * time.Second,
		})
	}
	return cmd, opts
}
