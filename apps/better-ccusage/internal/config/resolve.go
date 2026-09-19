package config

import (
	"strings"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
)

// ResolveDirs merges directories from CLAUDE_CONFIG_DIR env, config extras,
// and the data.DefaultDirs fallback. Order: env first, then extras, then defaults.
func ResolveDirs(envValue string, cfg *Config) []string {
	var dirs []string
	if envValue != "" {
		for _, d := range strings.Split(envValue, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				dirs = append(dirs, d)
			}
		}
	}
	if cfg != nil {
		dirs = append(dirs, cfg.ExtraConfigDirs...)
	}
	dirs = append(dirs, data.DefaultDirs()...)
	return dirs
}