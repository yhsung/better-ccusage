package adapters

import (
	"strings"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
)

type codexAdapter struct{}

func (codexAdapter) Name() string { return "codex" }
func (codexAdapter) Detect(e data.Entry) bool {
	return strings.HasPrefix(e.Model, "codex/") || e.SessionID != "" && strings.Contains(e.Model, "codex")
}
func (codexAdapter) Adapt(e data.Entry) (data.Entry, bool) {
	e.Model = strings.TrimPrefix(e.Model, "codex/")
	return e, true
}
