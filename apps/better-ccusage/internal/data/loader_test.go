package data

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_EmptyDirs(t *testing.T) {
	l := NewLoader()
	_, err := l.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for empty dirs")
	}
}

func TestLoad_ReadsJSONL(t *testing.T) {
	dir := t.TempDir()
	project := filepath.Join(dir, "projects", "myproj")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	jsonl := `{"timestamp":"2026-01-15T10:00:00Z","sessionId":"s1","model":"claude-sonnet-4-5-20250929","inputTokens":100,"outputTokens":50}
{"timestamp":"2026-01-15T11:00:00Z","sessionId":"s1","model":"claude-sonnet-4-5-20250929","inputTokens":200,"outputTokens":100}
this is not valid json
{"timestamp":"2026-01-15T12:00:00Z","sessionId":"s1","model":"claude-sonnet-4-5-20250929","inputTokens":50,"outputTokens":25}
`
	if err := os.WriteFile(filepath.Join(project, "abc.jsonl"), []byte(jsonl), 0o644); err != nil {
		t.Fatal(err)
	}
	l := NewLoader(dir)
	entries, err := l.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries (1 malformed skipped), got %d", len(entries))
	}
	// Verify sorted by timestamp
	if !entries[0].Timestamp.Before(entries[1].Timestamp) {
		t.Error("entries not sorted by timestamp")
	}
	if entries[0].InputTokens != 100 || entries[1].InputTokens != 200 {
		t.Errorf("unexpected token counts: %+v", entries)
	}
}

func TestLoad_MissingDir(t *testing.T) {
	l := NewLoader("/nonexistent/path/that/does/not/exist")
	entries, err := l.Load(context.Background())
	if err != nil {
		t.Fatalf("missing dir should be tolerated, got: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestFilterByTime(t *testing.T) {
	entries := []Entry{
		{Timestamp: mustTime("2026-01-15T10:00:00Z")},
		{Timestamp: mustTime("2026-01-15T11:00:00Z")},
		{Timestamp: mustTime("2026-01-15T12:00:00Z")},
	}
	since := mustTime("2026-01-15T10:30:00Z")
	until := mustTime("2026-01-15T11:30:00Z")
	out := FilterByTime(entries, &since, &until)
	if len(out) != 1 {
		t.Errorf("expected 1 entry, got %d", len(out))
	}
}

func mustTime(s string) (t time.Time) {
	t, _ = timeParse(s)
	return
}

func timeParse(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}