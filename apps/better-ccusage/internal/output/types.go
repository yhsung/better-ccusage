package output

import "github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"

// DailyResult is the JSON output shape for the `daily` command.
type DailyResult struct {
	Daily   []DailyRow `json:"daily"`
	Summary Summary    `json:"summary"`
}

// DailyRow is one day's usage.
type DailyRow struct {
	Date                string   `json:"date"`
	InputTokens         int64    `json:"inputTokens"`
	OutputTokens        int64    `json:"outputTokens"`
	CacheCreationTokens int64    `json:"cacheCreationTokens"`
	CacheReadTokens     int64    `json:"cacheReadTokens"`
	TotalTokens         int64    `json:"totalTokens"`
	CostUSD             float64  `json:"costUSD"`
	Models              []string `json:"models"`
}

// Summary is the totals row.
type Summary struct {
	TotalTokens int64   `json:"totalTokens"`
	CostUSD     float64 `json:"costUSD"`
}

// BucketsToDailyRows converts cost buckets to DailyRows.
func BucketsToDailyRows(buckets []cost.Bucket) []DailyRow {
	out := make([]DailyRow, 0, len(buckets))
	for _, b := range buckets {
		models := append([]string(nil), b.Models...)
		if models == nil {
			models = []string{}
		}
		out = append(out, DailyRow{
			Date:                b.Key,
			InputTokens:         b.InputTokens,
			OutputTokens:        b.OutputTokens,
			CacheCreationTokens: b.CacheCreationTokens,
			CacheReadTokens:     b.CacheReadTokens,
			TotalTokens:         b.InputTokens + b.OutputTokens + b.CacheCreationTokens + b.CacheReadTokens,
			CostUSD:             float64(b.Cost.Micros) / 1_000_000,
			Models:              models,
		})
	}
	return out
}

// SumRows totals a slice of DailyRows.
func SumRows(rows []DailyRow) Summary {
	var s Summary
	for _, r := range rows {
		s.TotalTokens += r.TotalTokens
		s.CostUSD += r.CostUSD
	}
	return s
}