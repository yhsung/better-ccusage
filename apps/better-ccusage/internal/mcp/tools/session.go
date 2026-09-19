package tools

import (
	"context"
	"io"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/commands"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/mcp/transport"
)

// Session implements the MCP `session` tool: usage grouped by conversation session.
func Session(ctx context.Context, d Deps, args ReportArgs) (*mcp.CallToolResult, any, error) {
	common, err := ParseCommon(d, args)
	if err != nil {
		return transport.ToolError(err)
	}
	result, err := commands.Session(ctx, commands.SessionOpts{CommonOpts: common}, io.Discard, d.Prices)
	if err != nil {
		return transport.ToolError(err)
	}
	return EncodeResult(result)
}
