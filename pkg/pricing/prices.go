package pricing

import (
	"encoding/json"
	"fmt"
	"io"
)

// Price holds the per-token pricing for one model, in dollars (not micros).
type Price struct {
	InputCostPerToken            float64 `json:"input_cost_per_token"`
	OutputCostPerToken           float64 `json:"output_cost_per_token"`
	CacheCreationInputTokenCost  float64 `json:"cache_creation_input_token_cost,omitempty"`
	CacheReadInputTokenCost      float64 `json:"cache_read_input_token_cost,omitempty"`
	MaxTokens                    int     `json:"max_tokens,omitempty"`
}

// rawPricesFile matches the upstream JSON shape: model name → *Price.
type rawPricesFile map[string]*Price

// PriceTable is an immutable lookup table of model prices.
type PriceTable struct {
	entries rawPricesFile
}

// LoadPrices parses a JSON pricing file from r.
func LoadPrices(r io.Reader) (*PriceTable, error) {
	if r == nil {
		return nil, fmt.Errorf("nil reader")
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading prices: %w", err)
	}
	if len(data) == 0 {
		return &PriceTable{entries: rawPricesFile{}}, nil
	}
	var raw rawPricesFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing prices: %w", err)
	}
	if raw == nil {
		raw = rawPricesFile{}
	}
	return &PriceTable{entries: raw}, nil
}

// rawPrices returns the underlying map for internal use (e.g. fuzzy scorer).
// Returns nil if pt is nil.
func (pt *PriceTable) rawPrices() rawPricesFile {
	if pt == nil {
		return nil
	}
	return pt.entries
}
