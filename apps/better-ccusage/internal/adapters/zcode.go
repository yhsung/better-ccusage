package adapters

import "github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"

type zcodeAdapter struct{}

func (zcodeAdapter) Name() string { return "zcode" }
func (zcodeAdapter) Detect(e data.Entry) bool { return e.Project == "zcode" }
func (zcodeAdapter) Adapt(e data.Entry) (data.Entry, bool) { return e, true }
