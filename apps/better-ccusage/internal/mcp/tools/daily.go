package tools

import (
	"context"
	"io"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/commands"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/mcp/transport"
)

// Daily implements the MCP `daily` tool: usage report grouped by date.
func Daily(ctx context.Context, d Deps, args ReportArgs) (*mcp.CallToolResult, any, error) {
	common, err := ParseCommon(d, args)
	if err != nil {
		return transport.ToolError(err)
	}
	result, err := commands.Daily(ctx, commands.DailyOpts{CommonOpts: common}, io.Discard, d.Prices)
	if err != nil {
		return transport.ToolError(err)
	}
	return EncodeResult(result)
}
