// Package data contains the core data model and JSONL loader for better-ccusage.
package data

import (
	"encoding/json"
	"fmt"
	"time"
)

// Entry represents one line of a Claude Code JSONL log file.
// Field names match the TS schema in apps/better-ccusage/src/_types.ts.
type Entry struct {
	Timestamp           time.Time `json:"-"`
	TimestampRaw        string    `json:"timestamp"`
	SessionID           string    `json:"sessionId"`
	Project             string    `json:"project,omitempty"`
	Model               string    `json:"model"`
	InputTokens         int64     `json:"inputTokens"`
	OutputTokens        int64     `json:"outputTokens"`
	CacheCreationTokens int64     `json:"cacheCreationTokens"`
	CacheReadTokens     int64     `json:"cacheReadTokens"`
	CostUSD             *float64  `json:"costUSD,omitempty"`
	IsAPIError          bool      `json:"isApiError,omitempty"`
}

// UnmarshalJSON parses the timestamp string into time.Time.
func (e *Entry) UnmarshalJSON(data []byte) error {
	type alias Entry
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*e = Entry(a)
	if a.TimestampRaw != "" {
		ts, err := time.Parse(time.RFC3339Nano, a.TimestampRaw)
		if err != nil {
			return fmt.Errorf("parsing timestamp %q: %w", a.TimestampRaw, err)
		}
		e.Timestamp = ts
	}
	return nil
}