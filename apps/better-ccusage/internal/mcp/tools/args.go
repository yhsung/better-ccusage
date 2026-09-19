// Package tools implements the MCP tool handlers for the MCP server.
// One file per tool; shared arg parsing and encoding lives here.
package tools

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/commands"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/mcp/transport"
	"github.com/cobra91/better-ccusage/pkg/pricing"
)

// Deps are the server-owned inputs every tool handler needs.
type Deps struct {
	ConfigDir   string
	DefaultMode cost.CostMode
	Prices      *pricing.PriceTable
}

// ReportArgs are the MCP arguments shared by daily/session/monthly/blocks.
// Dates are RFC3339 (the only format the Go command layer parses).
type ReportArgs struct {
	Since string `json:"since,omitempty" jsonschema:"Filter entries since this RFC3339 timestamp"`
	Until string `json:"until,omitempty" jsonschema:"Filter entries until this RFC3339 timestamp"`
	Mode  string `json:"mode,omitempty" jsonschema:"Cost calculation mode: auto, calculate, display"`
}

// ParseCommon converts MCP args into commands.CommonOpts.
// Empty mode falls back to the server default mode.
func ParseCommon(d Deps, args ReportArgs) (commands.CommonOpts, error) {
	var out commands.CommonOpts
	out.ConfigDir = d.ConfigDir
	switch args.Mode {
	case "":
		out.Mode = d.DefaultMode
	case "auto":
		out.Mode = cost.CostAuto
	case "calculate":
		out.Mode = cost.CostCalculate
	case "display":
		out.Mode = cost.CostDisplay
	default:
		return commands.CommonOpts{}, fmt.Errorf("%w: %q", errs.ErrInvalidMode, args.Mode)
	}
	if args.Since != "" {
		t, err := time.Parse(time.RFC3339, args.Since)
		if err != nil {
			return commands.CommonOpts{}, fmt.Errorf("%w: since %q", errs.ErrInvalidArgs, args.Since)
		}
		out.Since = &t
	}
	if args.Until != "" {
		t, err := time.Parse(time.RFC3339, args.Until)
		if err != nil {
			return commands.CommonOpts{}, fmt.Errorf("%w: until %q", errs.ErrInvalidArgs, args.Until)
		}
		out.Until = &t
	}
	return out, nil
}

// EncodeResult marshals a command result as 2-space JSON text content.
func EncodeResult(v any) (*mcp.CallToolResult, any, error) {
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return transport.ToolError(err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(body)}},
	}, nil, nil
}
