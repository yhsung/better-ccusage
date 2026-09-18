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

// LookupFuzzy scores every key in the table against model using a simple
// partial-match score (longest common prefix length / max(len(model), len(key))).
// Returns the best match whose score >= threshold, or (Price{}, 0.0) if none.
// Threshold of 0.6 matches the spec; pass 0.0 to accept any non-zero match.
func (pt *PriceTable) LookupFuzzy(model string, threshold float64) (Price, float64) {
	entries := pt.rawPrices()
	if entries == nil || model == "" {
		return Price{}, 0.0
	}
	bestPrice := Price{}
	bestScore := 0.0
	for key, p := range entries {
		if p == nil {
			continue
		}
		s := fuzzyScore(model, key)
		if s > bestScore {
			bestScore = s
			bestPrice = *p
		}
	}
	if bestScore < threshold {
		return Price{}, 0.0
	}
	return bestPrice, bestScore
}

// Lookup performs the full 3-tier resolution: exact → provider-prefix → fuzzy.
// Returns (Price, confidence). Confidence is 1.0 for exact, 0.85 for
// provider-prefix, and the fuzzy score for tier 3 (>= 0.6 to be considered a
// match). Returns (Price{}, 0.0) when no tier produces a match.
func (pt *PriceTable) Lookup(model string) (Price, float64) {
	if p, ok := pt.LookupExact(model); ok {
		return p, 1.0
	}
	if p, ok := pt.LookupProviderPrefix(model); ok {
		return p, 0.85
	}
	return pt.LookupFuzzy(model, 0.6)
}

// fuzzyScore computes a simple similarity score between a and b in [0, 1].
// The score is the longest common prefix length divided by the longer string.
func fuzzyScore(a, b string) float64 {
	la := len(a)
	lb := len(b)
	maxLen := la
	if lb > maxLen {
		maxLen = lb
	}
	if maxLen == 0 {
		return 0.0
	}
	common := 0
	n := la
	if lb < n {
		n = lb
	}
	for i := 0; i < n; i++ {
		if a[i] == b[i] {
			common++
		} else {
			break
		}
	}
	return float64(common) / float64(maxLen)
}