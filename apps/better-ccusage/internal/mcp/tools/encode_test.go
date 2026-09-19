package tools

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestEncodeResult_Success(t *testing.T) {
	res, _, err := EncodeResult(map[string]any{"daily": []any{map[string]any{"date": "2026-01-15"}}})
	if err != nil {
		t.Fatalf("EncodeResult: %v", err)
	}
	if res.IsError {
		t.Fatal("EncodeResult: unexpected IsError result")
	}
	if len(res.Content) != 1 {
		t.Fatalf("EncodeResult: content length = %d, want 1", len(res.Content))
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("EncodeResult: content[0] is %T, want *mcp.TextContent", res.Content[0])
	}
	if !strings.Contains(tc.Text, "\n  ") {
		t.Errorf("EncodeResult: expected 2-space indented JSON, got: %q", tc.Text)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(tc.Text), &decoded); err != nil {
		t.Fatalf("EncodeResult: invalid JSON: %v", err)
	}
	if _, ok := decoded["daily"]; !ok {
		t.Errorf("EncodeResult: expected daily key, got: %v", decoded)
	}
}

func TestEncodeResult_Unmarshallable(t *testing.T) {
	res, _, err := EncodeResult(map[string]any{"bad": func() {}})
	if err != nil {
		t.Fatalf("EncodeResult: %v", err)
	}
	if !res.IsError {
		t.Fatal("EncodeResult: expected IsError for unmarshallable value")
	}
}
