package monitor

import (
	"strings"
	"testing"
	"time"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/pkg/pricing"
)

func TestRenderBlocks(t *testing.T) {
	buckets := []cost.Bucket{
		{Key: "2026-01-15T05", Timestamp: time.Now(), InputTokens: 100, OutputTokens: 50, Cost: pricing.Money{Micros: 1500}},
	}
	out := renderBlocks(buckets)
	if !strings.Contains(out, "2026-01-15T05") {
		t.Errorf("missing block key in output: %q", out)
	}
	if !strings.Contains(out, "tokens=150") {
		t.Errorf("missing token total: %q", out)
	}
	if !strings.Contains(out, "$0.0015") {
		t.Errorf("missing cost: %q", out)
	}
}
