package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
)

// Config is the user-facing configuration loaded from ~/.config/better-ccusage/config.json.
type Config struct {
	PricingURL      string   `json:"pricingUrl,omitempty"`
	Offline         bool     `json:"offline,omitempty"`
	DefaultMode     string   `json:"defaultMode,omitempty"`
	ExtraConfigDirs []string `json:"extraConfigDirs,omitempty"`
}

// Load reads the config file at path. If the file does not exist, returns
// an empty Config (not an error) — matches TS behavior.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%w: %v", errs.ErrInvalidJSON, err)
	}
	return &c, nil
}

// DefaultPath returns the default config file path.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return home + "/.config/better-ccusage/config.json"
}