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

func setupBlocksToolFixture(t *testing.T) string {
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

func TestBlocks_Integration(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := setupBlocksToolFixture(t)
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	d := Deps{ConfigDir: dir, DefaultMode: cost.CostAuto, Prices: nil}
	res, _, err := Blocks(context.Background(), d, ReportArgs{})
	if err != nil {
		t.Fatalf("Blocks: %v", err)
	}
	if res.IsError {
		t.Fatalf("Blocks: unexpected IsError result: %+v", res.Content)
	}
	if len(res.Content) == 0 {
		t.Fatal("Blocks: empty content")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("Blocks: content[0] is %T, want *mcp.TextContent", res.Content[0])
	}
	if !strings.Contains(tc.Text, `"daily"`) {
		t.Errorf("Blocks: expected JSON with daily key, got: %q", tc.Text)
	}
}

func TestBlocks_NoData(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir() // no projects/ fixture inside
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	d := Deps{ConfigDir: dir, DefaultMode: cost.CostAuto, Prices: nil}
	res, _, err := Blocks(context.Background(), d, ReportArgs{})
	if err != nil {
		t.Fatalf("Blocks: %v", err)
	}
	if !res.IsError {
		t.Fatal("Blocks: expected IsError for empty dir")
	}
	if len(res.Content) == 0 {
		t.Fatal("Blocks: empty content")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("Blocks: content[0] is %T, want *mcp.TextContent", res.Content[0])
	}
	if !strings.Contains(tc.Text, "NO_DATA") {
		t.Errorf("Blocks: expected NO_DATA, got: %q", tc.Text)
	}
}
