// Package server builds the MCP server and registers the
// usage-reporting tools.
package server

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/mcp/tools"
	"github.com/cobra91/better-ccusage/pkg/pricing"
	"github.com/cobra91/better-ccusage/pkg/shim"
)

// Name is the MCP server name.
const Name = "better-ccusage-mcp"

// Opts configures the MCP server.
type Opts struct {
	ConfigDir   string
	DefaultMode cost.CostMode
	Prices      *pricing.PriceTable
	Version     string
	// CodexBin overrides the codex shim path; empty resolves better-ccusage-codex lazily per call.
	CodexBin string
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
	mcp.AddTool[tools.CodexArgs, any](srv, &mcp.Tool{Name: "codex-daily", Description: "Show Codex usage grouped by day"},
		func(ctx context.Context, req *mcp.CallToolRequest, args tools.CodexArgs) (*mcp.CallToolResult, any, error) {
			return tools.CodexDaily(ctx, resolveCodexBin(opts), args)
		})
	mcp.AddTool[tools.CodexArgs, any](srv, &mcp.Tool{Name: "codex-monthly", Description: "Show Codex usage grouped by month"},
		func(ctx context.Context, req *mcp.CallToolRequest, args tools.CodexArgs) (*mcp.CallToolResult, any, error) {
			return tools.CodexMonthly(ctx, resolveCodexBin(opts), args)
		})
	return srv
}

// resolveCodexBin returns the configured shim path, resolving
// better-ccusage-codex lazily so servers start without the shim installed.
func resolveCodexBin(opts Opts) string {
	if opts.CodexBin != "" {
		return opts.CodexBin
	}
	p, _ := shim.ResolveBinary("better-ccusage-codex")
	return p
}
