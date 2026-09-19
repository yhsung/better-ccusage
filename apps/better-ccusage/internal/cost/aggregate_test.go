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

func TestAggregate_SetsModelAndPreCalcCost(t *testing.T) {
	day := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	pre1, pre2 := 0.004, 0.006
	entries := []data.Entry{
		{Timestamp: day, TimestampRaw: day.Format(time.RFC3339), Model: "model-a", InputTokens: 100, CostUSD: &pre1},
		{Timestamp: day, TimestampRaw: day.Format(time.RFC3339), Model: "model-b", InputTokens: 100, CostUSD: &pre2},
		{Timestamp: day, TimestampRaw: day.Format(time.RFC3339), Model: "model-a", InputTokens: 100},
	}
	buckets := Aggregate(entries, GroupByDay)
	if len(buckets) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(buckets))
	}
	b := buckets[0]
	if b.Model != "model-a" {
		t.Errorf("primary model: got %q, want %q (most frequent)", b.Model, "model-a")
	}
	if len(b.Models) != 2 {
		t.Errorf("models: got %v, want 2 distinct", b.Models)
	}
	if b.Cost.Micros != 10_000 {
		t.Errorf("pre-calc cost: got %d micros, want 10000", b.Cost.Micros)
	}
}

func TestApplyPrices_DisplayUsesPreCalc(t *testing.T) {
	pt, _ := pricing.LoadPrices(strings.NewReader(`{"model-a":{"input_cost_per_token":0.000003,"output_cost_per_token":0.000015}}`))
	day := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	pre := 0.004
	entries := []data.Entry{
		{Timestamp: day, TimestampRaw: day.Format(time.RFC3339), Model: "model-a", InputTokens: 100, CostUSD: &pre},
	}
	buckets := Aggregate(entries, GroupByDay)
	out := ApplyPrices(buckets, pt, CostDisplay)
	if out[0].Cost.Micros != 4_000 {
		t.Errorf("display cost: got %d, want 4000 (pre-calc)", out[0].Cost.Micros)
	}
}

func TestApplyPrices_AutoPrefersPreCalc(t *testing.T) {
	pt, _ := pricing.LoadPrices(strings.NewReader(`{"model-a":{"input_cost_per_token":0.000003,"output_cost_per_token":0.000015}}`))
	day := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	pre := 0.004
	withPre := []data.Entry{
		{Timestamp: day, TimestampRaw: day.Format(time.RFC3339), Model: "model-a", InputTokens: 1000, CostUSD: &pre},
	}
	out := ApplyPrices(Aggregate(withPre, GroupByDay), pt, CostAuto)
	if out[0].Cost.Micros != 4_000 {
		t.Errorf("auto with pre-calc: got %d, want 4000", out[0].Cost.Micros)
	}
	withoutPre := []data.Entry{
		{Timestamp: day, TimestampRaw: day.Format(time.RFC3339), Model: "model-a", InputTokens: 1000, OutputTokens: 500},
	}
	out = ApplyPrices(Aggregate(withoutPre, GroupByDay), pt, CostAuto)
	if out[0].Cost.Micros != 10_500 {
		t.Errorf("auto without pre-calc: got %d, want 10500 (calculated)", out[0].Cost.Micros)
	}
}

func TestCalcCost_CacheReadRate(t *testing.T) {
	pt, _ := pricing.LoadPrices(strings.NewReader(`{"model-a":{"input_cost_per_token":0.000003,"output_cost_per_token":0.000015,"cache_read_input_token_cost":0.0000003}}`))
	day := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	buckets := Aggregate([]data.Entry{
		{Timestamp: day, TimestampRaw: day.Format(time.RFC3339), Model: "model-a", InputTokens: 1000, CacheReadTokens: 1000},
	}, GroupByDay)
	out := ApplyPrices(buckets, pt, CostCalculate)
	// 1000*0.000003=3000 + 1000*0.0000003=300 → 3300
	if out[0].Cost.Micros != 3_300 {
		t.Errorf("cache-read cost: got %d, want 3300", out[0].Cost.Micros)
	}
	// Fallback: no cache-read rate → cache reads priced at input rate.
	pt2, _ := pricing.LoadPrices(strings.NewReader(`{"model-a":{"input_cost_per_token":0.000003,"output_cost_per_token":0.000015}}`))
	out2 := ApplyPrices(buckets, pt2, CostCalculate)
	// 2000*0.000003=6000
	if out2[0].Cost.Micros != 6_000 {
		t.Errorf("cache-read fallback cost: got %d, want 6000", out2[0].Cost.Micros)
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