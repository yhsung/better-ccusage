package cost

import (
	"fmt"
	"sort"
	"time"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
	"github.com/cobra91/better-ccusage/pkg/pricing"
)

// Aggregate groups entries by key into buckets. Pure function, no I/O.
func Aggregate(entries []data.Entry, key GroupKey) []Bucket {
	groups := make(map[string]*Bucket)
	for _, e := range entries {
		gk := groupKeyFor(e, key)
		b, ok := groups[gk]
		if !ok {
			b = &Bucket{Key: gk, Timestamp: e.Timestamp}
			groups[gk] = b
		}
		b.InputTokens += e.InputTokens
		b.OutputTokens += e.OutputTokens
		b.CacheCreationTokens += e.CacheCreationTokens
		b.CacheReadTokens += e.CacheReadTokens
		b.Count++
	}
	out := make([]Bucket, 0, len(groups))
	for _, b := range groups {
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	return out
}

// groupKeyFor returns the grouping key for an entry under the given key
// strategy.
func groupKeyFor(e data.Entry, key GroupKey) string {
	switch key {
	case GroupByDay:
		return e.Timestamp.Format("2006-01-02")
	case GroupByMonth:
		return e.Timestamp.Format("2006-01")
	case GroupBySession:
		return e.SessionID
	case GroupByBlock:
		return blockKey(e.Timestamp)
	default:
		return ""
	}
}

// blockKey returns a 5-hour billing-block key like "2026-01-15T05".
// Matches TS session-blocks.ts: hour = floor(UTC_hour / 5) * 5.
func blockKey(t time.Time) string {
	h := t.UTC().Hour() / 5 * 5
	return t.UTC().Format("2006-01-02") + fmt.Sprintf("T%02d", h)
}

// ApplyPrices derives Cost on each bucket according to mode and prices.
func ApplyPrices(buckets []Bucket, prices *pricing.PriceTable, mode CostMode) []Bucket {
	out := make([]Bucket, len(buckets))
	for i, b := range buckets {
		price, _ := prices.Lookup(b.Model)
		out[i] = applyPrice(b, price, mode)
	}
	return out
}

// applyPrice dispatches to the correct per-mode cost function.
func applyPrice(b Bucket, price pricing.Price, mode CostMode) Bucket {
	switch mode {
	case CostCalculate:
		return calcCost(b, price)
	case CostDisplay:
		// Display mode shows pre-calculated costUSD; without per-bucket
		// pre-calc, it stays zero. Plans that need true pre-calc display
		// should pre-compute costUSD per entry upstream.
		return b
	default: // CostAuto
		return autoCost(b, price)
	}
}

// autoCost calculates from tokens when pricing data is present; otherwise
// leaves the bucket unchanged (cost stays zero).
func autoCost(b Bucket, price pricing.Price) Bucket {
	if price.InputCostPerToken > 0 || price.OutputCostPerToken > 0 {
		return calcCost(b, price)
	}
	return b
}

// calcCost multiplies each token count by its per-token price and sums the
// result. Per-token prices are converted to micros first; token counts are
// multiplied via Money.MulFloat so all arithmetic stays in int64.
func calcCost(b Bucket, p pricing.Price) Bucket {
	in := pricing.Money{Micros: int64(p.InputCostPerToken * 1_000_000)}.MulFloat(float64(b.InputTokens + b.CacheReadTokens))
	cacheW := pricing.Money{Micros: int64(p.CacheCreationInputTokenCost * 1_000_000)}.MulFloat(float64(b.CacheCreationTokens))
	out := pricing.Money{Micros: int64(p.OutputCostPerToken * 1_000_000)}.MulFloat(float64(b.OutputTokens))
	b.Cost = in.Add(cacheW).Add(out)
	return b
}