package cost

import (
	"strings"
	"testing"
	"time"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
	"github.com/cobra91/better-ccusage/pkg/pricing"
)

func mkEntry(t time.Time, model string, in, out int64) data.Entry {
	return data.Entry{
		Timestamp:    t,
		TimestampRaw: t.Format(time.RFC3339),
		Model:        model,
		InputTokens:  in,
		OutputTokens: out,
	}
}

func TestAggregate_GroupByDay(t *testing.T) {
	day1 := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 16, 10, 0, 0, 0, time.UTC)
	entries := []data.Entry{
		mkEntry(day1, "claude-sonnet-4-5-20250929", 100, 50),
		mkEntry(day1, "claude-sonnet-4-5-20250929", 200, 100),
		mkEntry(day2, "claude-sonnet-4-5-20250929", 50, 25),
	}
	buckets := Aggregate(entries, GroupByDay)
	if len(buckets) != 2 {
		t.Fatalf("expected 2 day buckets, got %d", len(buckets))
	}
	if buckets[0].InputTokens != 300 {
		t.Errorf("day1 input: got %d, want 300", buckets[0].InputTokens)
	}
	if buckets[1].InputTokens != 50 {
		t.Errorf("day2 input: got %d, want 50", buckets[1].InputTokens)
	}
}

func TestApplyPrices_Calculate(t *testing.T) {
	day := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	buckets := []Bucket{{
		Key: "2026-01-15", Timestamp: day,
		InputTokens: 1000, OutputTokens: 500,
		Model: "claude-sonnet-4-5-20250929",
	}}
	price := pricing.Price{
		InputCostPerToken:  0.000003,
		OutputCostPerToken: 0.000015,
	}
	out := ApplyPrices(buckets, &pricing.PriceTable{}, CostCalculate)
	_ = price // prices used via lookup; use real table in TestApplyPrices_WithTable
	if out[0].Cost.Micros == 0 {
		// With empty PriceTable, lookup returns zero, cost is zero. We test real cost via WithTable.
		t.Logf("note: empty table yields zero cost (expected for CostCalculate with no prices)")
	}
}

func TestApplyPrices_WithTable(t *testing.T) {
	pt, _ := pricing.LoadPrices(strings.NewReader(`{"claude-sonnet-4-5-20250929":{"input_cost_per_token":0.000003,"output_cost_per_token":0.000015}}`))
	day := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	buckets := []Bucket{{
		Key: "2026-01-15", Timestamp: day,
		InputTokens: 1000, OutputTokens: 500,
		Model:       "claude-sonnet-4-5-20250929",
	}}
	out := ApplyPrices(buckets, pt, CostCalculate)
	// 1000 * 0.000003 = 0.003 → 3000 micros
	// 500 * 0.000015 = 0.0075 → 7500 micros
	// total: 10500 micros
	if out[0].Cost.Micros != 10500 {
		t.Errorf("cost: got %d micros, want 10500", out[0].Cost.Micros)
	}
}