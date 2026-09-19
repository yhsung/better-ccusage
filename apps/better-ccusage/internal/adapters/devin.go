package adapters

import "github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"

type devinAdapter struct{}

func (devinAdapter) Name() string { return "devin" }
func (devinAdapter) Detect(e data.Entry) bool { return e.Project == "devin" }
func (devinAdapter) Adapt(e data.Entry) (data.Entry, bool) { return e, true }
