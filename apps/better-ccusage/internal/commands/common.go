// Package commands holds shared CLI flag wiring for all better-ccusage commands.
package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
)

// CommonOpts are the flags shared across all better-ccusage commands.
type CommonOpts struct {
	Mode      cost.CostMode
	Offline   bool
	JSON      bool
	Since     *time.Time
	Until     *time.Time
	ConfigDir string
}

// costModeValue adapts cost.CostMode to a string-valued pflag.Value so --mode
// stays a string flag on the CLI while opts.Mode stays typed.
type costModeValue struct {
	mode *cost.CostMode
}

func (v costModeValue) String() string {
	if v.mode == nil {
		return cost.CostAuto.String()
	}
	return v.mode.String()
}

func (v costModeValue) Type() string { return "string" }

func (v costModeValue) Set(s string) error {
	switch s {
	case "auto":
		*v.mode = cost.CostAuto
	case "calculate":
		*v.mode = cost.CostCalculate
	case "display":
		*v.mode = cost.CostDisplay
	default:
		return fmt.Errorf("invalid --mode %q (want auto, calculate, or display): %w", s, errs.ErrInvalidMode)
	}
	return nil
}

// BindCommonFlags attaches persistent flags to cmd and returns the opts
// pointer that cobra will populate.
func BindCommonFlags(cmd *cobra.Command) *CommonOpts {
	opts := &CommonOpts{Mode: cost.CostAuto}
	cmd.PersistentFlags().Var(costModeValue{mode: &opts.Mode}, "mode", "cost calculation mode: auto, calculate, display")
	cmd.PersistentFlags().BoolVar(&opts.Offline, "offline", false, "skip live pricing fetch")
	cmd.PersistentFlags().BoolVar(&opts.JSON, "json", false, "output as JSON")
	cmd.PersistentFlags().StringVar(&opts.ConfigDir, "config-dir", "", "additional Claude config directory (comma-separated for multiple)")
	cmd.PersistentFlags().String("since", "", "filter entries since this RFC3339 timestamp")
	cmd.PersistentFlags().String("until", "", "filter entries until this RFC3339 timestamp")
	// Pre-run parsing (chains any existing PreRunE so callers that set
	// their own hook before/after binding keep both behaviors).
	prevPreRunE := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if prevPreRunE != nil {
			if err := prevPreRunE(cmd, args); err != nil {
				return err
			}
		}
		if s, _ := cmd.Flags().GetString("since"); s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return err
			}
			opts.Since = &t
		}
		if s, _ := cmd.Flags().GetString("until"); s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return err
			}
			opts.Until = &t
		}
		return nil
	}
	return opts
}
