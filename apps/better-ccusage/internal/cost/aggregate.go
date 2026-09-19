package cost

import (
	"fmt"
	"sort"
	"time"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
	"github.com/cobra91/better-ccusage/pkg/pricing"
)

// Aggregate groups entries by key into buckets. Pure function, no I/O.
//
// Each bucket records the most-frequent non-empty entry Model as its primary
// Model (used by ApplyPrices for price lookup), the sorted distinct model
// list in Models, and the sum of per-entry pre-calculated CostUSD in Cost
// (so CostDisplay/CostAuto can honor upstream pre-calc).
func Aggregate(entries []data.Entry, key GroupKey) []Bucket {
	groups := make(map[string]*Bucket)
	modelCounts := make(map[string]map[string]int)
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
		if e.CostUSD != nil {
			b.Cost.Micros += int64(*e.CostUSD * 1_000_000)
		}
		if e.Model != "" {
			mc, ok := modelCounts[gk]
			if !ok {
				mc = make(map[string]int)
				modelCounts[gk] = mc
			}
			mc[e.Model]++
		}
		b.Count++
	}
	out := make([]Bucket, 0, len(groups))
	for gk, b := range groups {
		if mc := modelCounts[gk]; len(mc) > 0 {
			best, bestN := "", -1
			for m, n := range mc {
				if n > bestN || (n == bestN && m < best) {
					best, bestN = m, n
				}
			}
			b.Model = best
			b.Models = make([]string, 0, len(mc))
			for m := range mc {
				b.Models = append(b.Models, m)
			}
			sort.Strings(b.Models)
		}
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
	case GroupByWeek:
		yr, wk := e.Timestamp.ISOWeek()
		return fmt.Sprintf("%04d-W%02d", yr, wk)
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
		// Display mode shows the pre-calculated costUSD summed by
		// Aggregate; buckets without pre-calc remain zero.
		return b
	default: // CostAuto
		return autoCost(b, price)
	}
}

// autoCost prefers the pre-calculated costUSD summed by Aggregate when
// present; otherwise it calculates from tokens when pricing data exists.
// Buckets with neither pre-calc nor pricing remain zero.
func autoCost(b Bucket, price pricing.Price) Bucket {
	if b.Cost.Micros != 0 {
		return b
	}
	if price.InputCostPerToken > 0 || price.OutputCostPerToken > 0 {
		return calcCost(b, price)
	}
	return b
}

// calcCost multiplies each token count by its per-token price and sums the
// result. Per-token prices are converted to micros first; token counts are
// multiplied via Money.MulFloat so all arithmetic stays in int64.
// Cache-read tokens are priced at CacheReadInputTokenCost; when that rate is
// zero/unset (older entries), they fall back to the base input rate.
func calcCost(b Bucket, p pricing.Price) Bucket {
	cacheReadRate := p.CacheReadInputTokenCost
	if cacheReadRate == 0 {
		cacheReadRate = p.InputCostPerToken
	}
	in := pricing.Money{Micros: int64(p.InputCostPerToken * 1_000_000)}.MulFloat(float64(b.InputTokens))
	// NOTE: cache-read rates are often sub-micro per token (e.g. $0.30/MTok
	// = 0.3 micros), which would truncate to zero if converted to micros
	// per-unit first — so multiply rate × tokens × 1e6 in float before
	// converting.
	cacheR := pricing.Money{Micros: int64(cacheReadRate * float64(b.CacheReadTokens) * 1_000_000)}
	cacheW := pricing.Money{Micros: int64(p.CacheCreationInputTokenCost * 1_000_000)}.MulFloat(float64(b.CacheCreationTokens))
	out := pricing.Money{Micros: int64(p.OutputCostPerToken * 1_000_000)}.MulFloat(float64(b.OutputTokens))
	b.Cost = in.Add(cacheR).Add(cacheW).Add(out)
	return b
}