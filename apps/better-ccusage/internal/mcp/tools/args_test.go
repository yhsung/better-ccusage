package tools

import (
	"testing"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
)

func TestParseCommon(t *testing.T) {
	d := Deps{ConfigDir: "/tmp/x", DefaultMode: cost.CostAuto}
	got, err := ParseCommon(d, ReportArgs{Mode: "calculate", Since: "2026-01-01T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != cost.CostCalculate {
		t.Errorf("mode: got %v, want calculate", got.Mode)
	}
	if got.ConfigDir != "/tmp/x" {
		t.Errorf("configdir: got %q", got.ConfigDir)
	}
	if got.Since == nil || got.Until != nil {
		t.Error("since must parse, until must stay nil")
	}

	def, err := ParseCommon(d, ReportArgs{})
	if err != nil {
		t.Fatal(err)
	}
	if def.Mode != cost.CostAuto {
		t.Errorf("default mode: got %v", def.Mode)
	}

	if _, err := ParseCommon(d, ReportArgs{Mode: "bogus"}); err == nil {
		t.Error("bad mode must error")
	}
	if _, err := ParseCommon(d, ReportArgs{Since: "not-a-date"}); err == nil {
		t.Error("bad since must error")
	}
}
