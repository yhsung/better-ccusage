package commands

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
)

func TestBindCommonFlags_Defaults(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	opts := BindCommonFlags(cmd)
	if err := cmd.ParseFlags([]string{}); err != nil {
		t.Fatal(err)
	}
	if opts.Mode != cost.CostAuto {
		t.Errorf("default mode: got %q, want %q", opts.Mode, cost.CostAuto)
	}
	if opts.Offline {
		t.Error("default offline: got true, want false")
	}
	if opts.JSON {
		t.Error("default json: got true, want false")
	}
	if opts.Since != nil {
		t.Errorf("default since: got %v, want nil", opts.Since)
	}
	if opts.Until != nil {
		t.Errorf("default until: got %v, want nil", opts.Until)
	}
	if opts.ConfigDir != "" {
		t.Errorf("default config-dir: got %q, want empty", opts.ConfigDir)
	}
}

func TestBindCommonFlags_ModeValues(t *testing.T) {
	for _, tc := range []struct {
		flag string
		want cost.CostMode
	}{
		{"auto", cost.CostAuto},
		{"calculate", cost.CostCalculate},
		{"display", cost.CostDisplay},
	} {
		cmd := &cobra.Command{Use: "test"}
		opts := BindCommonFlags(cmd)
		if err := cmd.ParseFlags([]string{"--mode", tc.flag}); err != nil {
			t.Fatalf("--mode %s: ParseFlags: %v", tc.flag, err)
		}
		if opts.Mode != tc.want {
			t.Errorf("--mode %s: got %q, want %q", tc.flag, opts.Mode, tc.want)
		}
	}
}

func TestBindCommonFlags_InvalidMode(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	BindCommonFlags(cmd)
	err := cmd.ParseFlags([]string{"--mode", "bogus"})
	if err == nil {
		t.Fatal("ParseFlags(--mode bogus): got nil, want error")
	}
	if !errors.Is(err, errs.ErrIncompatibleMode) {
		t.Errorf("ParseFlags(--mode bogus): error %v does not wrap ErrIncompatibleMode", err)
	}
}

func TestBindCommonFlags_SinceUntil(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	opts := BindCommonFlags(cmd)
	if err := cmd.ParseFlags([]string{"--since", "2026-01-01T00:00:00Z", "--until", "2026-02-01T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if err := cmd.PreRunE(cmd, nil); err != nil {
		t.Fatalf("PreRunE: %v", err)
	}
	if opts.Since == nil || opts.Since.Format("2006-01-02") != "2026-01-01" {
		t.Errorf("since: got %v, want 2026-01-01", opts.Since)
	}
	if opts.Until == nil || opts.Until.Format("2006-01-02") != "2026-02-01" {
		t.Errorf("until: got %v, want 2026-02-01", opts.Until)
	}
}

func TestBindCommonFlags_InvalidSince(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	BindCommonFlags(cmd)
	if err := cmd.ParseFlags([]string{"--since", "not-a-time"}); err != nil {
		t.Fatal(err)
	}
	if err := cmd.PreRunE(cmd, nil); err == nil {
		t.Error("PreRunE(--since not-a-time): got nil, want error")
	}
}
