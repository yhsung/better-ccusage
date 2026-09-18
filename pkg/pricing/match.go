package pricing

import "strings"

// LookupExact returns the price for model if an exact (case-sensitive) match
// exists in the table. Returns (Price{}, false) otherwise.
func (pt *PriceTable) LookupExact(model string) (Price, bool) {
	entries := pt.rawPrices()
	if entries == nil {
		return Price{}, false
	}
	p, ok := entries[model]
	if !ok || p == nil {
		return Price{}, false
	}
	return *p, true
}

// LookupProviderPrefix splits model on "/" and looks up the suffix in the
// table. Returns (Price{}, false) if no "/" is present or the suffix is not
// found. Matches TS behavior: the suffix after the last "/" is the lookup key.
func (pt *PriceTable) LookupProviderPrefix(model string) (Price, bool) {
	entries := pt.rawPrices()
	if entries == nil {
		return Price{}, false
	}
	idx := strings.LastIndex(model, "/")
	if idx < 0 || idx == len(model)-1 {
		return Price{}, false
	}
	suffix := model[idx+1:]
	p, ok := entries[suffix]
	if !ok || p == nil {
		return Price{}, false
	}
	return *p, true
}

// Lookup performs the tier-1 + tier-2 resolution. Tier-3 (fuzzy) is added in
// the next task. Returns (Price, confidence). Confidence is 1.0 for exact
// matches and 0.85 for provider-prefix matches (matches TS scoring band).
func (pt *PriceTable) Lookup(model string) (Price, float64) {
	if p, ok := pt.LookupExact(model); ok {
		return p, 1.0
	}
	if p, ok := pt.LookupProviderPrefix(model); ok {
		return p, 0.85
	}
	return Price{}, 0.0
}
