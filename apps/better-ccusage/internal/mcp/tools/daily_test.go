package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
)

func setupDailyToolFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	project := filepath.Join(dir, "projects", "p")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	jsonl := `{"timestamp":"2026-01-15T10:00:00Z","sessionId":"s1","model":"claude-sonnet-4-5-20250929","inputTokens":1000,"outputTokens":500}
`
	if err := os.WriteFile(filepath.Join(project, "session1.jsonl"), []byte(jsonl), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestDaily_Integration(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := setupDailyToolFixture(t)
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	d := Deps{ConfigDir: dir, DefaultMode: cost.CostAuto, Prices: nil}
	res, _, err := Daily(context.Background(), d, ReportArgs{})
	if err != nil {
		t.Fatalf("Daily: %v", err)
	}
	if res.IsError {
		t.Fatalf("Daily: unexpected IsError result: %+v", res.Content)
	}
	if len(res.Content) == 0 {
		t.Fatal("Daily: empty content")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("Daily: content[0] is %T, want *mcp.TextContent", res.Content[0])
	}
	if !strings.Contains(tc.Text, `"daily"`) {
		t.Errorf("Daily: expected JSON with daily key, got: %q", tc.Text)
	}
}

func TestDaily_InvalidMode(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	d := Deps{ConfigDir: t.TempDir(), DefaultMode: cost.CostAuto, Prices: nil}
	res, _, err := Daily(context.Background(), d, ReportArgs{Mode: "bogus"})
	if err != nil {
		t.Fatalf("Daily: %v", err)
	}
	if !res.IsError {
		t.Fatal("Daily: expected IsError for bogus mode")
	}
	if len(res.Content) == 0 {
		t.Fatal("Daily: empty content")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("Daily: content[0] is %T, want *mcp.TextContent", res.Content[0])
	}
	if !strings.Contains(tc.Text, "INVALID_ARGS") {
		t.Errorf("Daily: expected INVALID_ARGS, got: %q", tc.Text)
	}
}
