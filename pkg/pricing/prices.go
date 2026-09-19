package pricing

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
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
//
// Upstream LiteLLM JSON contains a "sample_spec" placeholder entry whose
// cost fields are documentation strings, not numbers. That single known
// placeholder is skipped silently. Any other entry that does not decode as
// a Price returns an error listing the offending keys, so upstream schema
// drift can't pass silently. A top-level JSON syntax error still returns an
// error.
func LoadPrices(r io.Reader) (*PriceTable, error) {
	if r == nil {
		return nil, fmt.Errorf("nil reader")
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading prices: %w", err)
	}
	raw := rawPricesFile{}
	if len(data) == 0 {
		return &PriceTable{entries: raw}, nil
	}
	var entries map[string]json.RawMessage
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parsing prices: %w", err)
	}
	var skipped []string
	for name, msg := range entries {
		var p Price
		if err := json.Unmarshal(msg, &p); err != nil {
			// Only the known upstream "sample_spec" docs placeholder is
			// skipped silently; any other undecodable entry is recorded
			// and reported so schema drift can't pass silently.
			if name == "sample_spec" {
				continue
			}
			skipped = append(skipped, name)
			continue
		}
		cp := p
		raw[name] = &cp
	}
	if len(skipped) > 0 {
		sort.Strings(skipped)
		return nil, fmt.Errorf("parsing prices: %d invalid model entries: %s", len(skipped), strings.Join(skipped, ", "))
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
