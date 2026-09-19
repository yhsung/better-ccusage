package transport

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
)

func TestMapError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code string
	}{
		{"nodata", errs.ErrNoData, "NO_DATA"},
		{"unknownmodel", errs.ErrUnknownModel, "UNKNOWN_MODEL"},
		{"invalidmode", errs.ErrInvalidMode, "INVALID_ARGS"},
		{"invalidargs", errs.ErrInvalidArgs, "INVALID_ARGS"},
		{"wrapped", errors.Join(errs.ErrNoData, errors.New("x")), "NO_DATA"},
		{"default", errors.New("boom"), "INTERNAL"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, msg := MapError(tc.err)
			if code != tc.code {
				t.Errorf("code: got %q, want %q", code, tc.code)
			}
			if msg == "" {
				t.Error("message must not be empty")
			}
		})
	}
}

func TestErrorResult_Shape(t *testing.T) {
	res, _, err := ErrorResult("NO_DATA", "no usage data found", "hint")
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("IsError must be true")
	}
	if len(res.Content) != 1 {
		t.Fatalf("content length: got %d, want 1", len(res.Content))
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content type: got %T, want *mcp.TextContent", res.Content[0])
	}
	var p ErrorPayload
	if err := json.Unmarshal([]byte(tc.Text), &p); err != nil {
		t.Fatal(err)
	}
	if p.Code != "NO_DATA" || p.Message == "" || p.Hint == "" {
		t.Errorf("payload: %+v", p)
	}
}
