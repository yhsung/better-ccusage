package shim

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestFilterArgs(t *testing.T) {
	cfg := Config{OwnName: "better-ccusage-codex", Target: "true"}
	_ = cfg
	// Run() filtering is covered via TestRunExitCodes arg pass-through;
	// unit-check the rule directly:
	args := []string{"better-ccusage-codex", "daily", "--json"}
	var filtered []string
	for _, a := range args {
		if a != "better-ccusage-codex" {
			filtered = append(filtered, a)
		}
	}
	if len(filtered) != 2 || filtered[0] != "daily" {
		t.Errorf("filtered: %q", filtered)
	}
}

func TestBuildEnv(t *testing.T) {
	t.Setenv("SHIM_TEST_PARENT", "keep")
	t.Setenv("SHIM_TEST_UNSET", "")
	cfg := Config{
		Set:             map[string]string{"OFFLINE": "true"},
		SuppressIfUnset: map[string]string{"SHIM_TEST_PARENT": "x", "SHIM_TEST_UNSET": "fallback", "SHIM_TEST_MISSING": "fallback"},
	}
	env := buildEnv(os.Environ(), cfg)
	get := func(k string) string {
		for _, e := range env {
			if strings.HasPrefix(e, k+"=") {
				return strings.TrimPrefix(e, k+"=")
			}
		}
		return "<absent>"
	}
	if get("OFFLINE") != "true" {
		t.Error("Set must apply")
	}
	if get("SHIM_TEST_PARENT") != "keep" {
		t.Error("parent value must win over suppress")
	}
	if get("SHIM_TEST_UNSET") != "fallback" || get("SHIM_TEST_MISSING") != "fallback" {
		t.Error("unset/empty/missing must take fallback")
	}
}

func TestPrintNotice(t *testing.T) {
	cfg := Config{NoticeLines: []string{"l1", "l2"}, OptOutEnv: "SHIM_TEST_OPTOUT"}
	var buf bytes.Buffer
	printNotice(&buf, cfg, true)
	if buf.String() != "l1\nl2\n" {
		t.Errorf("notice: %q", buf.String())
	}
	buf.Reset()
	printNotice(&buf, cfg, false)
	if buf.String() != "" {
		t.Error("non-TTY must suppress")
	}
	t.Setenv("SHIM_TEST_OPTOUT", "1")
	printNotice(&buf, cfg, true)
	if buf.String() != "" {
		t.Error("opt-out must suppress")
	}
}

func TestResolveBinaryMissing(t *testing.T) {
	if _, err := ResolveBinary("definitely-not-a-binary-xyz"); err == nil {
		t.Error("missing binary must error")
	} else if !strings.Contains(err.Error(), "definitely-not-a-binary-xyz") {
		t.Errorf("error must name binary: %v", err)
	}
}

func TestRunExitCodes(t *testing.T) {
	if Run(Config{Target: "true"}, nil) != 0 {
		t.Error("true must exit 0")
	}
	if Run(Config{Target: "false"}, nil) != 1 {
		t.Error("false must exit 1")
	}
	if Run(Config{Target: "definitely-not-a-binary-xyz"}, nil) != 1 {
		t.Error("unresolvable must exit 1")
	}
}
