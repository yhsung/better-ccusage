package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_MissingFile(t *testing.T) {
	c, err := Load("/nonexistent/config.json")
	if err != nil {
		t.Fatalf("missing file should not error, got %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"pricingUrl":"https://example.com/prices.json","offline":true,"defaultMode":"calculate","extraConfigDirs":["/foo"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !c.Offline {
		t.Error("expected Offline=true")
	}
	if c.DefaultMode != "calculate" {
		t.Errorf("DefaultMode: got %q, want calculate", c.DefaultMode)
	}
	if len(c.ExtraConfigDirs) != 1 || c.ExtraConfigDirs[0] != "/foo" {
		t.Errorf("ExtraConfigDirs: got %v", c.ExtraConfigDirs)
	}
}

func TestResolveDirs_EnvFirst(t *testing.T) {
	got := ResolveDirs("/env/dir", &Config{ExtraConfigDirs: []string{"/cfg/dir"}})
	if got[0] != "/env/dir" {
		t.Errorf("env dir should be first, got %v", got)
	}
}

func TestResolveDirs_OnlyDefaults(t *testing.T) {
	got := ResolveDirs("", &Config{})
	if len(got) == 0 {
		t.Error("expected at least default dirs")
	}
}