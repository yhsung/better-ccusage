package output

import (
	"bytes"
	"encoding/json"
	"testing"
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