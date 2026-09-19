// Package cost aggregates usage entries into buckets and applies per-bucket
// pricing. It is a pure package — no I/O, no globals, no goroutines.
package cost

// GroupKey controls how Aggregate groups entries.
type GroupKey int

const (
	// GroupByDay buckets entries by calendar date in the entry's local timezone.
	GroupByDay GroupKey = iota
	// GroupByMonth buckets entries by year-month.
	GroupByMonth
	// GroupBySession buckets entries by Claude Code session ID.
	GroupBySession
	// GroupByBlock buckets entries into 5-hour billing blocks.
	GroupByBlock
	// GroupByWeek buckets entries by ISO week (e.g. "2026-W03").
	GroupByWeek GroupKey = 4
)

// CostMode controls how ApplyPrices derives per-bucket cost.
type CostMode int

const (
	// CostAuto calculates cost from token counts when pricing data exists for
	// the bucket's model; otherwise the cost stays zero.
	CostAuto CostMode = iota
	// CostCalculate always calculates cost from token counts; ignores any
	// pre-calculated upstream values.
	CostCalculate
	// CostDisplay relies on pre-calculated costUSD being supplied upstream.
	// Buckets without pre-calculated cost remain zero.
	CostDisplay
)

// String returns a human-readable name for the cost mode.
func (m CostMode) String() string {
	switch m {
	case CostAuto:
		return "auto"
	case CostCalculate:
		return "calculate"
	case CostDisplay:
		return "display"
	default:
		return "unknown"
	}
}