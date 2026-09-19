package adapters

import (
	"strings"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
)

type opencodeAdapter struct{}

func (opencodeAdapter) Name() string { return "opencode" }
func (opencodeAdapter) Detect(e data.Entry) bool {
	return strings.HasPrefix(e.Model, "opencode/")
}
func (opencodeAdapter) Adapt(e data.Entry) (data.Entry, bool) {
	e.Model = strings.TrimPrefix(e.Model, "opencode/")
	return e, true
}
