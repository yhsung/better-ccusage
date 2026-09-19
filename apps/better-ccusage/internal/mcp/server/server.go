// Package server builds the MCP server and registers the
// usage-reporting tools.
package server

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/mcp/tools"
	"github.com/cobra91/better-ccusage/pkg/pricing"
)

// Name is the MCP server name.
const Name = "better-ccusage-mcp"

// Opts configures the MCP server.
type Opts struct {
	ConfigDir   string
	DefaultMode cost.CostMode
	Prices      *pricing.PriceTable
	Version     string
}

// NewServer builds an MCP server with daily/session/monthly/blocks tools.
func NewServer(opts Opts) *mcp.Server {
	d := tools.Deps{ConfigDir: opts.ConfigDir, DefaultMode: opts.DefaultMode, Prices: opts.Prices}
	srv := mcp.NewServer(&mcp.Implementation{Name: Name, Version: opts.Version}, nil)
	mcp.AddTool[tools.ReportArgs, any](srv, &mcp.Tool{Name: "daily", Description: "Show usage report grouped by date"},
		func(ctx context.Context, req *mcp.CallToolRequest, args tools.ReportArgs) (*mcp.CallToolResult, any, error) {
			return tools.Daily(ctx, d, args)
		})
	mcp.AddTool[tools.ReportArgs, any](srv, &mcp.Tool{Name: "session", Description: "Show usage report grouped by conversation session"},
		func(ctx context.Context, req *mcp.CallToolRequest, args tools.ReportArgs) (*mcp.CallToolResult, any, error) {
			return tools.Session(ctx, d, args)
		})
	mcp.AddTool[tools.ReportArgs, any](srv, &mcp.Tool{Name: "monthly", Description: "Show usage report grouped by month"},
		func(ctx context.Context, req *mcp.CallToolRequest, args tools.ReportArgs) (*mcp.CallToolResult, any, error) {
			return tools.Monthly(ctx, d, args)
		})
	mcp.AddTool[tools.ReportArgs, any](srv, &mcp.Tool{Name: "blocks", Description: "Show usage report grouped by session billing blocks"},
		func(ctx context.Context, req *mcp.CallToolRequest, args tools.ReportArgs) (*mcp.CallToolResult, any, error) {
			return tools.Blocks(ctx, d, args)
		})
	return srv
}
