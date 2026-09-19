package commands

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
)

func TestAggregate_GroupByWeek(t *testing.T) {
	// 2026-01-11 is a Sunday (ISO W02), 2026-01-12 is the following Monday (W03).
	sun := time.Date(2026, 1, 11, 10, 0, 0, 0, time.UTC)
	mon := time.Date(2026, 1, 12, 10, 0, 0, 0, time.UTC)
	tue := time.Date(2026, 1, 13, 10, 0, 0, 0, time.UTC)
	entries := []data.Entry{
		{Timestamp: sun, Model: "m", InputTokens: 100},
		{Timestamp: mon, Model: "m", InputTokens: 200},
		{Timestamp: tue, Model: "m", InputTokens: 300},
	}
	buckets := cost.Aggregate(entries, cost.GroupByWeek)
	if len(buckets) != 2 {
		t.Fatalf("expected 2 week buckets, got %d", len(buckets))
	}
	if buckets[0].Key != "2026-W02" {
		t.Errorf("week bucket 0 key: got %q, want %q", buckets[0].Key, "2026-W02")
	}
	if buckets[1].Key != "2026-W03" {
		t.Errorf("week bucket 1 key: got %q, want %q", buckets[1].Key, "2026-W03")
	}
	if buckets[1].InputTokens != 500 {
		t.Errorf("W03 input: got %d, want 500", buckets[1].InputTokens)
	}
}

func TestWeekly_Integration(t *testing.T) {
	isolateHome(t)
	dir := setupDailyFixture(t)
	pt := testPriceTable(t)
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	var buf strings.Builder
	opts := WeeklyOpts{
		CommonOpts: CommonOpts{Mode: cost.CostCalculate, JSON: true},
	}
	result, err := Weekly(context.Background(), opts, &buf, pt)
	if err != nil {
		t.Fatalf("Weekly: %v", err)
	}
	// Both fixture dates (Thu 2026-01-15, Fri 2026-01-16) fall in ISO W03.
	if len(result.Daily) != 1 {
		t.Fatalf("expected 1 weekly row (both dates in same ISO week), got %d", len(result.Daily))
	}
	if result.Daily[0].Date != "2026-W03" {
		t.Errorf("weekly row date: got %q, want %q", result.Daily[0].Date, "2026-W03")
	}
	if result.Summary.TotalTokens == 0 {
		t.Error("expected non-zero total tokens in summary")
	}
}

func TestWeekly_NoData(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir() // no projects/ fixture inside
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	opts := WeeklyOpts{
		CommonOpts: CommonOpts{Mode: cost.CostCalculate},
	}
	var buf strings.Builder
	_, err := Weekly(context.Background(), opts, &buf, testPriceTable(t))
	if err == nil {
		t.Fatal("Weekly: got nil, want error for empty range")
	}
	if !errors.Is(err, errs.ErrNoData) {
		t.Errorf("Weekly: error %v does not wrap ErrNoData", err)
	}
}

func TestNewWeeklyCmd_JSON(t *testing.T) {
	isolateHome(t)
	dir := setupDailyFixture(t)
	cmd, _ := NewWeeklyCmd(testPriceTable(t))
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
	if !strings.Contains(out, "2026-W03") {
		t.Errorf("expected fixture ISO week in output, got: %q", out)
	}
}

func TestNewWeeklyCmd_Table(t *testing.T) {
	isolateHome(t)
	dir := setupDailyFixture(t)
	cmd, _ := NewWeeklyCmd(testPriceTable(t))
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--config-dir", dir, "--mode", "calculate"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "2026-W03") {
		t.Errorf("expected fixture ISO week in table, got: %q", out)
	}
}
