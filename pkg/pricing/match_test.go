package pricing

import (
	"strings"
	"testing"
)

func newTestTable(t *testing.T) *PriceTable {
	t.Helper()
	r := strings.NewReader(`{
		"claude-sonnet-4-5-20250929": {
			"input_cost_per_token": 0.000003,
			"output_cost_per_token": 0.000015
		},
		"moonshot/kimi-for-coding": {
			"input_cost_per_token": 0.000001,
			"output_cost_per_token": 0.000003
		}
	}`)
	pt, err := LoadPrices(r)
	if err != nil {
		t.Fatalf("LoadPrices: %v", err)
	}
	return pt
}

func TestLookupExact_Found(t *testing.T) {
	pt := newTestTable(t)
	got, ok := pt.LookupExact("claude-sonnet-4-5-20250929")
	if !ok {
		t.Fatal("expected match")
	}
	if got.InputCostPerToken != 0.000003 {
		t.Errorf("input cost: got %f, want 0.000003", got.InputCostPerToken)
	}
}

func TestLookupExact_NotFound(t *testing.T) {
	pt := newTestTable(t)
	_, ok := pt.LookupExact("nonexistent-model")
	if ok {
		t.Error("expected no match for nonexistent-model")
	}
}

func TestLookupExact_NilTable(t *testing.T) {
	var pt *PriceTable
	_, ok := pt.LookupExact("claude-sonnet-4-5-20250929")
	if ok {
		t.Error("expected no match for nil table")
	}
}

func TestLookupExact_CaseSensitive(t *testing.T) {
	pt := newTestTable(t)
	_, ok := pt.LookupExact("Claude-Sonnet-4-5-20250929")
	if ok {
		t.Error("expected no match — lookup must be case-sensitive (matches TS)")
	}
}
