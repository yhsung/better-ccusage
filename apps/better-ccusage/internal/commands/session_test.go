package commands

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
)

func TestSession_Integration(t *testing.T) {
	isolateHome(t)
	dir := setupDailyFixture(t)
	pt := testPriceTable(t)
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	var buf strings.Builder
	opts := SessionOpts{
		CommonOpts: CommonOpts{Mode: cost.CostCalculate, JSON: true},
	}
	result, err := Session(context.Background(), opts, &buf, pt)
	if err != nil {
		t.Fatalf("Session: %v", err)
	}
	if len(result.Daily) != 1 {
		t.Errorf("expected 1 session row (both entries share session s1), got %d", len(result.Daily))
	}
	if result.Summary.TotalTokens == 0 {
		t.Error("expected non-zero total tokens in summary")
	}
}

func TestSession_NoData(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir() // no projects/ fixture inside
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	opts := SessionOpts{
		CommonOpts: CommonOpts{Mode: cost.CostCalculate},
	}
	var buf strings.Builder
	_, err := Session(context.Background(), opts, &buf, testPriceTable(t))
	if err == nil {
		t.Fatal("Session: got nil, want error for empty range")
	}
	if !errors.Is(err, errs.ErrNoData) {
		t.Errorf("Session: error %v does not wrap ErrNoData", err)
	}
}

func TestNewSessionCmd_JSON(t *testing.T) {
	isolateHome(t)
	dir := setupDailyFixture(t)
	cmd, _ := NewSessionCmd(testPriceTable(t))
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
	if !strings.Contains(out, "s1") {
		t.Errorf("expected fixture session s1 in output, got: %q", out)
	}
}

func TestNewSessionCmd_Table(t *testing.T) {
	isolateHome(t)
	dir := setupDailyFixture(t)
	cmd, _ := NewSessionCmd(testPriceTable(t))
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--config-dir", dir, "--mode", "calculate"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "s1") {
		t.Errorf("expected fixture session s1 in table, got: %q", out)
	}
}
