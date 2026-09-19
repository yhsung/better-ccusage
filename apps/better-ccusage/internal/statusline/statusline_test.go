package statusline

import (
	"strings"
	"testing"

	"github.com/cobra91/better-ccusage/pkg/pricing"
	"github.com/cobra91/better-ccusage/pkg/terminal"
)

func TestRender_Basic(t *testing.T) {
	out := Render(Input{
		TotalCost:   pricing.Money{Micros: 50_000},
		TotalTokens: 1500,
		Models:      []string{"claude-sonnet-4-5-20250929"},
	})
	if !strings.Contains(out, "claude-sonnet-4-5") {
		t.Errorf("missing model: %q", out)
	}
	if !strings.Contains(out, "1500 tokens") {
		t.Errorf("missing token count: %q", out)
	}
	if !strings.Contains(out, "$0.0500") {
		t.Errorf("missing cost: %q", out)
	}
}

func TestRender_NoModel(t *testing.T) {
	out := Render(Input{TotalCost: pricing.Money{}, TotalTokens: 100})
	if !strings.Contains(out, "unknown") {
		t.Errorf("expected 'unknown' for empty models, got %q", out)
	}
	// Normalize ANSI before substring check
	normalized := terminal.StripANSI(out)
	if !strings.Contains(normalized, "unknown") {
		t.Errorf("normalized output missing 'unknown': %q", normalized)
	}
}
