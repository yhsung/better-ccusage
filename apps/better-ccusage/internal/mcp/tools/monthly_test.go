package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
)

func setupMonthlyToolFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	project := filepath.Join(dir, "projects", "p")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	jsonl := `{"timestamp":"2026-01-15T10:00:00Z","sessionId":"s1","model":"claude-sonnet-4-5-20250929","inputTokens":1000,"outputTokens":500}
{"timestamp":"2026-01-20T11:00:00Z","sessionId":"s1","model":"claude-sonnet-4-5-20250929","inputTokens":2000,"outputTokens":1000}
`
	if err := os.WriteFile(filepath.Join(project, "session1.jsonl"), []byte(jsonl), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestMonthly_Integration(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := setupMonthlyToolFixture(t)
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	d := Deps{ConfigDir: dir, DefaultMode: cost.CostAuto, Prices: nil}
	res, _, err := Monthly(context.Background(), d, ReportArgs{})
	if err != nil {
		t.Fatalf("Monthly: %v", err)
	}
	if res.IsError {
		t.Fatalf("Monthly: unexpected IsError result: %+v", res.Content)
	}
	if len(res.Content) == 0 {
		t.Fatal("Monthly: empty content")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("Monthly: content[0] is %T, want *mcp.TextContent", res.Content[0])
	}
	if !strings.Contains(tc.Text, `"daily"`) {
		t.Fatalf("Monthly: expected JSON with daily key, got: %q", tc.Text)
	}
	var decoded struct {
		Daily []json.RawMessage `json:"daily"`
	}
	if err := json.Unmarshal([]byte(tc.Text), &decoded); err != nil {
		t.Fatalf("Monthly: invalid JSON: %v", err)
	}
	if len(decoded.Daily) != 1 {
		t.Errorf("Monthly: expected 1 collapsed monthly row, got %d", len(decoded.Daily))
	}
}

func TestMonthly_InvalidDate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	d := Deps{ConfigDir: t.TempDir(), DefaultMode: cost.CostAuto, Prices: nil}
	res, _, err := Monthly(context.Background(), d, ReportArgs{Since: "not-a-date"})
	if err != nil {
		t.Fatalf("Monthly: %v", err)
	}
	if !res.IsError {
		t.Fatal("Monthly: expected IsError for invalid since date")
	}
	if len(res.Content) == 0 {
		t.Fatal("Monthly: empty content")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("Monthly: content[0] is %T, want *mcp.TextContent", res.Content[0])
	}
	if !strings.Contains(tc.Text, "INVALID_ARGS") {
		t.Errorf("Monthly: expected INVALID_ARGS, got: %q", tc.Text)
	}
}
