package adapters

import "github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"

// claudeAdapter is the base provider for vanilla Claude Code logs.
type claudeAdapter struct{}

func (claudeAdapter) Name() string { return "claude" }

// Detect returns true for any entry — claude is the default fallback.
func (claudeAdapter) Detect(e data.Entry) bool { return e.Model != "" }

// Adapt returns the entry unchanged. Provider-specific normalizations
// (e.g. unprefixing model names) happen in dedicated adapters.
func (claudeAdapter) Adapt(e data.Entry) (data.Entry, bool) { return e, true }