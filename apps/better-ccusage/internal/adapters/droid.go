package adapters

import "github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"

type droidAdapter struct{}

func (droidAdapter) Name() string { return "droid" }
func (droidAdapter) Detect(e data.Entry) bool { return e.Project == "droid" }
func (droidAdapter) Adapt(e data.Entry) (data.Entry, bool) { return e, true }
