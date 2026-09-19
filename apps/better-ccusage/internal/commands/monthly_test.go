package commands

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
)

func TestMonthly_Integration(t *testing.T) {
	isolateHome(t)
	dir := setupDailyFixture(t)
	pt := testPriceTable(t)
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	var buf strings.Builder
	opts := MonthlyOpts{
		CommonOpts: CommonOpts{Mode: cost.CostCalculate, JSON: true},
	}
	result, err := Monthly(context.Background(), opts, &buf, pt)
	if err != nil {
		t.Fatalf("Monthly: %v", err)
	}
	if len(result.Daily) != 1 {
		t.Errorf("expected 1 monthly row (both dates in same month), got %d", len(result.Daily))
	}
	if result.Summary.TotalTokens == 0 {
		t.Error("expected non-zero total tokens in summary")
	}
}

func TestMonthly_NoData(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir() // no projects/ fixture inside
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	opts := MonthlyOpts{
		CommonOpts: CommonOpts{Mode: cost.CostCalculate},
	}
	var buf strings.Builder
	_, err := Monthly(context.Background(), opts, &buf, testPriceTable(t))
	if err == nil {
		t.Fatal("Monthly: got nil, want error for empty range")
	}
	if !errors.Is(err, errs.ErrNoData) {
		t.Errorf("Monthly: error %v does not wrap ErrNoData", err)
	}
}

func TestNewMonthlyCmd_JSON(t *testing.T) {
	isolateHome(t)
	dir := setupDailyFixture(t)
	cmd, _ := NewMonthlyCmd(testPriceTable(t))
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
	if !strings.Contains(out, "2026-01") {
		t.Errorf("expected fixture month in output, got: %q", out)
	}
}

func TestNewMonthlyCmd_Table(t *testing.T) {
	isolateHome(t)
	dir := setupDailyFixture(t)
	cmd, _ := NewMonthlyCmd(testPriceTable(t))
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--config-dir", dir, "--mode", "calculate"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "2026-01") {
		t.Errorf("expected fixture month in table, got: %q", out)
	}
}
