package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func writeFakeCodexScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-codex")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func codexText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("empty content")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content[0] is %T, want *mcp.TextContent", res.Content[0])
	}
	return tc.Text
}

func TestCodexDaily_Success(t *testing.T) {
	bin := writeFakeCodexScript(t, `echo '{"daily":[],"summary":{"totalTokens":0,"costUSD":0}}'`)
	res, _, err := CodexDaily(context.Background(), bin, CodexArgs{})
	if err != nil {
		t.Fatalf("CodexDaily: %v", err)
	}
	if res.IsError {
		t.Fatalf("CodexDaily: unexpected IsError: %+v", res.Content)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(codexText(t, res)), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := decoded["daily"]; !ok {
		t.Fatalf("expected daily key, got: %v", decoded)
	}
}

func TestCodexDaily_Failure(t *testing.T) {
	bin := writeFakeCodexScript(t, `echo boom >&2; exit 3`)
	res, _, err := CodexDaily(context.Background(), bin, CodexArgs{})
	if err != nil {
		t.Fatalf("CodexDaily: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError result")
	}
}

func TestCodexDaily_MissingBin(t *testing.T) {
	res, _, err := CodexDaily(context.Background(), "", CodexArgs{})
	if err != nil {
		t.Fatalf("CodexDaily: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError result")
	}
	if !strings.Contains(codexText(t, res), "better-ccusage-codex") {
		t.Fatalf("expected shim name in error, got: %+v", res.Content)
	}
}
