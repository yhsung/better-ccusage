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
		"kimi-for-coding": {
			"input_cost_per_token": 0.000001,
			"output_cost_per_token": 0.000003
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

func TestLookupProviderPrefix_Found(t *testing.T) {
	pt := newTestTable(t)
	// "moonshot/kimi-for-coding" → after stripping provider, key becomes "kimi-for-coding"
	// which exists in the table.
	got, ok := pt.LookupProviderPrefix("moonshot/kimi-for-coding")
	if !ok {
		t.Fatal("expected provider-prefix match")
	}
	if got.InputCostPerToken != 0.000001 {
		t.Errorf("input cost: got %f, want 0.000001", got.InputCostPerToken)
	}
}

func TestLookupProviderPrefix_NoSlash(t *testing.T) {
	pt := newTestTable(t)
	_, ok := pt.LookupProviderPrefix("claude-sonnet-4-5-20250929")
	if ok {
		t.Error("expected no provider-prefix match when no slash present")
	}
}

func TestLookupProviderPrefix_SuffixNotInTable(t *testing.T) {
	pt := newTestTable(t)
	_, ok := pt.LookupProviderPrefix("acme/unknown-model")
	if ok {
		t.Error("expected no provider-prefix match when suffix not in table")
	}
}

func TestLookup_Tier1Preferred(t *testing.T) {
	// "claude-sonnet-4-5-20250929" exists exactly. Even though a provider-prefix
	// form could exist too, tier 1 wins and returns confidence 1.0.
	pt := newTestTable(t)
	_, conf := pt.Lookup("claude-sonnet-4-5-20250929")
	if conf != 1.0 {
		t.Errorf("confidence: got %f, want 1.0", conf)
	}
}

func TestLookup_Tier2Confidence(t *testing.T) {
	// "moonshot/kimi-for-coding" has tier-2 match; confidence must be in (0.0, 1.0].
	pt := newTestTable(t)
	_, conf := pt.Lookup("moonshot/kimi-for-coding")
	if conf <= 0.0 || conf > 1.0 {
		t.Errorf("confidence: got %f, want (0.0, 1.0]", conf)
	}
}

func TestLookupFuzzy_TypoMatch(t *testing.T) {
	// "kimi-for-codin" (missing 'g') is a fuzzy match for "kimi-for-coding"
	pt := newTestTable(t)
	p, score := pt.LookupFuzzy("kimi-for-codin", 0.6)
	if score < 0.6 {
		t.Fatalf("expected match above threshold, got score %f", score)
	}
	if p.InputCostPerToken != 0.000001 {
		t.Errorf("matched wrong price: got %f, want 0.000001", p.InputCostPerToken)
	}
}

func TestLookupFuzzy_BelowThreshold(t *testing.T) {
	pt := newTestTable(t)
	_, score := pt.LookupFuzzy("zzzzzzzzzzz", 0.6)
	if score >= 0.6 {
		t.Errorf("expected no match above threshold, got score %f", score)
	}
}

func TestLookup_Tier3Confidence(t *testing.T) {
	// Fuzzy match — confidence must equal the score returned by LookupFuzzy.
	pt := newTestTable(t)
	_, conf := pt.Lookup("kimi-for-codin")
	if conf < 0.6 || conf > 1.0 {
		t.Errorf("confidence: got %f, want [0.6, 1.0]", conf)
	}
}

func TestLookup_NoMatchReturnsZeroConfidence(t *testing.T) {
	pt := newTestTable(t)
	_, conf := pt.Lookup("totally-unrelated-model-name-zzz")
	if conf != 0.0 {
		t.Errorf("expected 0.0 confidence for no match, got %f", conf)
	}
}
