package commands

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
)

func setupStatuslineFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	project := filepath.Join(dir, "projects", "myproj")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	jsonl := `{"timestamp":"2026-01-15T10:00:00Z","sessionId":"s1","model":"claude-sonnet-4-5-20250929","inputTokens":1000,"outputTokens":500}
`
	if err := os.WriteFile(filepath.Join(project, "abc.jsonl"), []byte(jsonl), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestStatusline_Integration(t *testing.T) {
	isolateHome(t)
	dir := setupStatuslineFixture(t)
	pt := testPriceTable(t)
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	var buf strings.Builder
	opts := StatuslineOpts{
		CommonOpts: CommonOpts{Mode: cost.CostCalculate},
	}
	if err := Statusline(context.Background(), opts, &buf, pt); err != nil {
		t.Fatalf("Statusline: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if out == "" {
		t.Fatal("Statusline: expected non-empty rendered line")
	}
	if !strings.Contains(out, "tokens") {
		t.Errorf("expected token count in output, got: %q", out)
	}
	if lines := strings.Split(out, "\n"); len(lines) != 1 {
		t.Errorf("expected single-line output, got %d lines: %q", len(lines), out)
	}
}

func TestStatusline_NoData(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir() // no projects/ fixture inside
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	var buf strings.Builder
	opts := StatuslineOpts{
		CommonOpts: CommonOpts{Mode: cost.CostCalculate},
	}
	err := Statusline(context.Background(), opts, &buf, testPriceTable(t))
	if err == nil {
		t.Fatal("Statusline: got nil, want error for empty data")
	}
	if !errors.Is(err, errs.ErrNoData) {
		t.Errorf("Statusline: error %v does not wrap ErrNoData", err)
	}
}

func TestNewStatuslineCmd(t *testing.T) {
	isolateHome(t)
	dir := setupStatuslineFixture(t)
	cmd, _ := NewStatuslineCmd(testPriceTable(t))
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--config-dir", dir, "--mode", "calculate"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if out == "" {
		t.Fatal("expected non-empty statusline output")
	}
	if !strings.Contains(out, "tokens") {
		t.Errorf("expected token count in output, got: %q", out)
	}
}
