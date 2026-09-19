package commands

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
)

func TestBlocks_Integration(t *testing.T) {
	isolateHome(t)
	dir := setupDailyFixture(t)
	pt := testPriceTable(t)
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	var buf strings.Builder
	opts := BlocksOpts{
		CommonOpts: CommonOpts{Mode: cost.CostCalculate, JSON: true},
	}
	result, err := Blocks(context.Background(), opts, &buf, pt)
	if err != nil {
		t.Fatalf("Blocks: %v", err)
	}
	if len(result.Daily) == 0 {
		t.Error("expected at least 1 block row")
	}
}

func TestBlocks_NoData(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir() // no projects/ fixture inside
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	opts := BlocksOpts{
		CommonOpts: CommonOpts{Mode: cost.CostCalculate},
	}
	var buf strings.Builder
	_, err := Blocks(context.Background(), opts, &buf, testPriceTable(t))
	if err == nil {
		t.Fatal("Blocks: got nil, want error for empty range")
	}
	if !errors.Is(err, errs.ErrNoData) {
		t.Errorf("Blocks: error %v does not wrap ErrNoData", err)
	}
}

func TestNewBlocksCmd_JSON(t *testing.T) {
	isolateHome(t)
	dir := setupDailyFixture(t)
	cmd, _ := NewBlocksCmd(testPriceTable(t))
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--json", "--config-dir", dir, "--mode", "calculate"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"daily"`) {
		t.Errorf("expected JSON output with daily key, got: %q", out)
	}
}

func TestNewBlocksCmd_Table(t *testing.T) {
	isolateHome(t)
	dir := setupDailyFixture(t)
	cmd, _ := NewBlocksCmd(testPriceTable(t))
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--config-dir", dir, "--mode", "calculate"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Block") {
		t.Errorf("expected table header Block, got: %q", out)
	}
}

func TestNewBlocksCmd_Flags(t *testing.T) {
	isolateHome(t)
	cmd, opts := NewBlocksCmd(testPriceTable(t))
	for _, name := range []string{"active", "recent", "token-limit"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected --%s flag to be registered", name)
		}
	}
	var buf strings.Builder
	cmd.SetOut(&buf)
	dir := setupDailyFixture(t)
	cmd.SetArgs([]string{"--config-dir", dir, "--mode", "calculate", "--active", "--recent", "--token-limit", "1000"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !opts.Active {
		t.Error("expected Active to be true after --active")
	}
	if !opts.Recent {
		t.Error("expected Recent to be true after --recent")
	}
	if opts.TokenLimit != 1000 {
		t.Errorf("expected TokenLimit 1000, got %d", opts.TokenLimit)
	}
}
