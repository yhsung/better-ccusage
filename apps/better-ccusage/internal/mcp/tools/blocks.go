package tools

import (
	"context"
	"io"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/commands"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/mcp/transport"
)

// Blocks implements the MCP `blocks` tool: usage grouped by 5-hour billing blocks.
func Blocks(ctx context.Context, d Deps, args ReportArgs) (*mcp.CallToolResult, any, error) {
	common, err := ParseCommon(d, args)
	if err != nil {
		return transport.ToolError(err)
	}
	result, err := commands.Blocks(ctx, commands.BlocksOpts{CommonOpts: common}, io.Discard, d.Prices)
	if err != nil {
		return transport.ToolError(err)
	}
	return EncodeResult(result)
}
