package adapters

import "github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"

// Provider normalizes a data.Entry for a specific provider (claude, codex, ...).
type Provider interface {
	Name() string
	Detect(e data.Entry) bool
	Adapt(e data.Entry) (data.Entry, bool)
}

// Manager applies all registered providers to a slice of entries.
type Manager struct {
	providers []Provider
}

// NewManager returns a Manager pre-loaded with all built-in providers
// (claude base + 6 provider-specific adapters).
func NewManager() *Manager {
	return &Manager{providers: builtins()}
}

// Normalize runs each entry through the first provider whose Detect returns true.
// If no provider matches, the entry passes through unchanged.
func (m *Manager) Normalize(entries []data.Entry) []data.Entry {
	out := make([]data.Entry, 0, len(entries))
	for _, e := range entries {
		matched := false
		for _, p := range m.providers {
			if p.Detect(e) {
				if normalized, ok := p.Adapt(e); ok {
					out = append(out, normalized)
				}
				matched = true
				break
			}
		}
		if !matched {
			out = append(out, e)
		}
	}
	return out
}

func builtins() []Provider {
	return []Provider{
		codexAdapter{},
		opencodeAdapter{},
		devinAdapter{},
		piAdapter{},
		zcodeAdapter{},
		droidAdapter{},
		claudeAdapter{}, // last (fallback)
	}
}