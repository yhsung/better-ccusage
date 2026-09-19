package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
)

func TestEncodeJSON_Indented(t *testing.T) {
	var buf bytes.Buffer
	if err := EncodeJSON(&buf, map[string]any{"a": 1, "b": "two"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Should contain newlines (indented)
	if len(out) < 10 {
		t.Errorf("output too short: %q", out)
	}
	// Should parse back
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Errorf("output not valid JSON: %v", err)
	}
	if got["a"].(float64) != 1 {
		t.Errorf("roundtrip failed: %v", got)
	}
}

func TestBucketsToDailyRows(t *testing.T) {
	rows := BucketsToDailyRows(nil)
	if len(rows) != 0 {
		t.Errorf("expected 0 rows for nil, got %d", len(rows))
	}
}

func TestBucketsToDailyRows_PopulatesModels(t *testing.T) {
	rows := BucketsToDailyRows([]cost.Bucket{{
		Key:    "2026-01-15",
		Models: []string{"model-a", "model-b"},
		Model:  "model-a",
	}})
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if len(rows[0].Models) != 2 || rows[0].Models[0] != "model-a" {
		t.Errorf("models: got %v, want [model-a model-b]", rows[0].Models)
	}
	// Buckets without models must encode as [] not null.
	rows = BucketsToDailyRows([]cost.Bucket{{Key: "2026-01-16"}})
	if rows[0].Models == nil {
		t.Error("models should be non-nil empty slice, got nil")
	}
}