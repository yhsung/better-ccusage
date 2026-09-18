package pricing

import (
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
