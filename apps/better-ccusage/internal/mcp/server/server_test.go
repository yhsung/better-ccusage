package server

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
)

func TestNewServer_ListsFourTools_Integration(t *testing.T) {
	ctx := context.Background()
	srv := NewServer(Opts{Version: "test"})
	ct, st := mcp.NewInMemoryTransports()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	srvSess, err := srv.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server Connect: %v", err)
	}
	defer srvSess.Close()
	cliSess, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client Connect: %v", err)
	}
	defer cliSess.Close()
	res, err := cliSess.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	want := []string{"blocks", "daily", "monthly", "session"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("tools = %v, want %v", names, want)
	}
}

func TestServer_CallDaily_Integration(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
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
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	ctx := context.Background()
	srv := NewServer(Opts{ConfigDir: dir, DefaultMode: cost.CostAuto, Prices: nil, Version: "test"})
	ct, st := mcp.NewInMemoryTransports()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	srvSess, err := srv.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server Connect: %v", err)
	}
	defer srvSess.Close()
	cliSess, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client Connect: %v", err)
	}
	defer cliSess.Close()
	res, err := cliSess.CallTool(ctx, &mcp.CallToolParams{Name: "daily", Arguments: map[string]any{"mode": "auto"}})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("CallTool: unexpected IsError result: %+v", res.Content)
	}
	if len(res.Content) == 0 {
		t.Fatal("CallTool: empty content")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("CallTool: content[0] is %T, want *mcp.TextContent", res.Content[0])
	}
	if !strings.Contains(tc.Text, `"daily"`) {
		t.Errorf("CallTool: expected JSON with daily key, got: %q", tc.Text)
	}
}
