package pricing

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

func TestLoadPrices_Success(t *testing.T) {
	r := strings.NewReader(`{
		"claude-sonnet-4-5-20250929": {
			"input_cost_per_token": 0.000003,
			"output_cost_per_token": 0.000015
		}
	}`)
	pt, err := LoadPrices(r)
	if err != nil {
		t.Fatalf("LoadPrices: %v", err)
	}
	if got := pt.entries["claude-sonnet-4-5-20250929"]; got == nil {
		t.Fatal("expected entry for claude-sonnet-4-5-20250929")
	} else if math.Abs(got.InputCostPerToken-0.000003) > 1e-9 {
		t.Errorf("input cost: got %f, want 0.000003", got.InputCostPerToken)
	}
}

func TestLoadPrices_InvalidJSON(t *testing.T) {
	r := strings.NewReader(`{not valid json`)
	_, err := LoadPrices(r)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadPrices_EmptyReader(t *testing.T) {
	r := strings.NewReader("")
	pt, err := LoadPrices(r)
	if err != nil {
		t.Fatalf("LoadPrices: %v", err)
	}
	if len(pt.entries) != 0 {
		t.Errorf("expected empty table, got %d entries", len(pt.entries))
	}
}

func TestEmbeddedPrices_NonEmpty(t *testing.T) {
	if len(EmbeddedPrices) == 0 {
		t.Fatal("EmbeddedPrices is empty — go:embed failed")
	}
	if !strings.Contains(string(EmbeddedPrices), "claude-sonnet-4-5-20250929") {
		t.Error("EmbeddedPrices missing expected model entry")
	}
}

func TestLoadPrices_SkipsNonModelEntries(t *testing.T) {
	// Mirrors upstream LiteLLM JSON: "sample_spec" is a docs placeholder
	// whose cost fields are strings, not numbers.
	r := strings.NewReader(`{
		"sample_spec": {"input_cost_per_token": "max input tokens, if the provider specifies it"},
		"claude-sonnet-4-5-20250929": {"input_cost_per_token": 0.000003, "output_cost_per_token": 0.000015}
	}`)
	pt, err := LoadPrices(r)
	if err != nil {
		t.Fatalf("LoadPrices: %v", err)
	}
	if len(pt.entries) != 1 {
		t.Errorf("expected 1 entry (sample_spec skipped), got %d", len(pt.entries))
	}
	if _, ok := pt.LookupExact("claude-sonnet-4-5-20250929"); !ok {
		t.Error("expected claude-sonnet-4-5-20250929 to load")
	}
}

func TestLoadPrices_UnknownBadEntryErrors(t *testing.T) {
	r := strings.NewReader(`{
		"bogus-model": {"input_cost_per_token": "not-a-number"},
		"claude-sonnet-4-5-20250929": {"input_cost_per_token": 0.000003, "output_cost_per_token": 0.000015}
	}`)
	_, err := LoadPrices(r)
	if err == nil {
		t.Fatal("expected error for unknown undecodable entry, got nil")
	}
	if !strings.Contains(err.Error(), "bogus-model") {
		t.Errorf("error should name the bad entry, got: %v", err)
	}
}

func TestLoadPrices_EmbeddedIsValid(t *testing.T) {
	pt, err := LoadPrices(bytes.NewReader(EmbeddedPrices))
	if err != nil {
		t.Fatalf("embedded prices invalid: %v", err)
	}
	if len(pt.rawPrices()) == 0 {
		t.Error("embedded prices loaded but empty")
	}
	if got := len(pt.rawPrices()); got < 500 {
		t.Errorf("embedded model count = %d, want >= 500 (guards silent per-entry skips on upstream schema drift)", got)
	}
	if _, ok := pt.LookupExact("claude-sonnet-4-5-20250929"); !ok {
		t.Error("expected claude-sonnet-4-5-20250929 in embedded prices")
	}
	if _, conf := pt.Lookup("claude-sonnet-4-5-20250929"); conf == 0 {
		t.Error("expected Lookup to resolve claude-sonnet-4-5-20250929")
	}
}
