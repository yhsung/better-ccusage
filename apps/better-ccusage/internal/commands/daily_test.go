package commands

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
	"github.com/cobra91/better-ccusage/pkg/pricing"
)

func setupDailyFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	project := filepath.Join(dir, "projects", "myproj")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	jsonl := `{"timestamp":"2026-01-15T10:00:00Z","sessionId":"s1","model":"claude-sonnet-4-5-20250929","inputTokens":1000,"outputTokens":500}
{"timestamp":"2026-01-16T10:00:00Z","sessionId":"s1","model":"claude-sonnet-4-5-20250929","inputTokens":2000,"outputTokens":1000}
`
	if err := os.WriteFile(filepath.Join(project, "abc.jsonl"), []byte(jsonl), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func testPriceTable(t *testing.T) *pricing.PriceTable {
	t.Helper()
	pt, err := pricing.LoadPrices(strings.NewReader(`{"claude-sonnet-4-5-20250929":{"input_cost_per_token":0.000003,"output_cost_per_token":0.000015}}`))
	if err != nil {
		t.Fatalf("LoadPrices: %v", err)
	}
	return pt
}

// isolateHome points HOME at an empty temp dir so ResolveDirs'
// DefaultDirs fallback (and config.DefaultPath) stay hermetic and never
// pick up the developer's real ~/.claude data.
func isolateHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func TestDaily_Integration(t *testing.T) {
	isolateHome(t)
	dir := setupDailyFixture(t)
	pt := testPriceTable(t)
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	var buf strings.Builder
	opts := DailyOpts{
		CommonOpts: CommonOpts{Mode: cost.CostCalculate, JSON: true},
	}
	result, err := Daily(context.Background(), opts, &buf, pt)
	if err != nil {
		t.Fatalf("Daily: %v", err)
	}
	if len(result.Daily) != 2 {
		t.Errorf("expected 2 daily rows, got %d", len(result.Daily))
	}
	if result.Summary.TotalTokens == 0 {
		t.Error("expected non-zero total tokens in summary")
	}
}

func TestDaily_NoData(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir() // no projects/ fixture inside
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	opts := DailyOpts{
		CommonOpts: CommonOpts{Mode: cost.CostCalculate},
	}
	var buf strings.Builder
	_, err := Daily(context.Background(), opts, &buf, testPriceTable(t))
	if err == nil {
		t.Fatal("Daily: got nil, want error for empty range")
	}
	if !errors.Is(err, errs.ErrNoData) {
		t.Errorf("Daily: error %v does not wrap ErrNoData", err)
	}
}

func TestNewDailyCmd_JSON(t *testing.T) {
	isolateHome(t)
	dir := setupDailyFixture(t)
	cmd, _ := NewDailyCmd(testPriceTable(t))
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
	if !strings.Contains(out, "2026-01-15") || !strings.Contains(out, "2026-01-16") {
		t.Errorf("expected both fixture dates in output, got: %q", out)
	}
}

func TestNewDailyCmd_Table(t *testing.T) {
	isolateHome(t)
	dir := setupDailyFixture(t)
	cmd, _ := NewDailyCmd(testPriceTable(t))
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--config-dir", dir, "--mode", "calculate"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "2026-01-15") || !strings.Contains(out, "2026-01-16") {
		t.Errorf("expected both fixture dates in table, got: %q", out)
	}
}
