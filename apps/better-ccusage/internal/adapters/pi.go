package adapters

import "github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"

type piAdapter struct{}

func (piAdapter) Name() string { return "pi" }
func (piAdapter) Detect(e data.Entry) bool { return e.Project == "pi" || e.SessionID != "" && e.Project == "opencode-pi" }
func (piAdapter) Adapt(e data.Entry) (data.Entry, bool) { return e, true }
