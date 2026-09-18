package pricing

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
