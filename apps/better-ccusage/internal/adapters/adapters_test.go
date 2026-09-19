package adapters

import (
	"testing"
	"time"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
)

func mkE(model string) data.Entry {
	return data.Entry{
		Timestamp:    time.Now(),
		TimestampRaw: time.Now().Format(time.RFC3339),
		Model:        model,
	}
}

func TestManager_DefaultsToClaude(t *testing.T) {
	m := NewManager()
	entries := []data.Entry{mkE("claude-sonnet-4-5-20250929"), mkE("kimi-for-coding")}
	out := m.Normalize(entries)
	if len(out) != 2 {
		t.Fatalf("expected 2 entries (both via claude fallback), got %d", len(out))
	}
}

func TestManager_EmptyEntries(t *testing.T) {
	m := NewManager()
	out := m.Normalize(nil)
	if len(out) != 0 {
		t.Errorf("expected 0 entries, got %d", len(out))
	}
}

func TestCodexAdapter(t *testing.T) {
	m := NewManager()
	out := m.Normalize([]data.Entry{mkE("codex/kimi-for-coding")})
	if out[0].Model != "kimi-for-coding" {
		t.Errorf("expected stripped model, got %q", out[0].Model)
	}
}

func TestOpencodeAdapter(t *testing.T) {
	m := NewManager()
	out := m.Normalize([]data.Entry{mkE("opencode/claude-sonnet-4-5-20250929")})
	if out[0].Model != "claude-sonnet-4-5-20250929" {
		t.Errorf("expected stripped model, got %q", out[0].Model)
	}
}