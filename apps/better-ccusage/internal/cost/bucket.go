package cost

import (
	"time"

	"github.com/cobra91/better-ccusage/pkg/pricing"
)

// Bucket is the unit of aggregation output produced by Aggregate.
type Bucket struct {
	// Key is the group identifier (date string, session ID, or block key).
	Key string
	// Timestamp is the representative timestamp for the bucket — the
	// timestamp of the first entry that landed in it.
	Timestamp time.Time
	// InputTokens is the sum of input tokens across all entries in the bucket.
	InputTokens int64
	// OutputTokens is the sum of output tokens across all entries in the bucket.
	OutputTokens int64
	// CacheCreationTokens is the sum of cache-creation tokens across all
	// entries in the bucket.
	CacheCreationTokens int64
	// CacheReadTokens is the sum of cache-read tokens across all entries in
	// the bucket.
	CacheReadTokens int64
	// Cost is the monetary cost for the bucket. Populated by ApplyPrices.
	Cost pricing.Money
	// Model is the primary model used in the bucket (most-used).
	Model string
	// Models is the sorted list of distinct non-empty models seen in the
	// bucket. Populated by Aggregate.
	Models []string
	// Count is the number of entries aggregated into the bucket.
	Count int
}