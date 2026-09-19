package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/mcp/transport"
)

// CodexArgs are the MCP arguments for codex-daily / codex-monthly.
// TS also accepts timezone/locale; the Go CLI has no such flags, so they
// are intentionally absent here.
type CodexArgs struct {
	Since string `json:"since,omitempty" jsonschema:"Filter entries since this date"`
	Until string `json:"until,omitempty" jsonschema:"Filter entries until this date"`
}

// CodexDaily implements the MCP `codex-daily` tool via the codex shim binary.
func CodexDaily(ctx context.Context, bin string, args CodexArgs) (*mcp.CallToolResult, any, error) {
	return runCodexCli(ctx, bin, "daily", args)
}

// CodexMonthly implements the MCP `codex-monthly` tool via the codex shim binary.
func CodexMonthly(ctx context.Context, bin string, args CodexArgs) (*mcp.CallToolResult, any, error) {
	return runCodexCli(ctx, bin, "monthly", args)
}

func runCodexCli(ctx context.Context, bin, command string, args CodexArgs) (*mcp.CallToolResult, any, error) {
	if bin == "" {
		return transport.ToolError(fmt.Errorf("could not resolve %q: codex shim not installed", "better-ccusage-codex"))
	}
	cliArgs := []string{command, "--json"}
	if args.Since != "" {
		cliArgs = append(cliArgs, "--since", args.Since)
	}
	if args.Until != "" {
		cliArgs = append(cliArgs, "--until", args.Until)
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, cliArgs...)
	cmd.Env = os.Environ()
	var se bytes.Buffer
	cmd.Stderr = &se
	out, err := cmd.Output()
	if err != nil {
		snippet := ""
		if se.Len() > 0 {
			snippet = ": " + string(bytes.TrimSpace(se.Bytes()[:min(se.Len(), 500)]))
		}
		return transport.ToolError(fmt.Errorf("codex %s: %w%s", command, err, snippet))
	}
	if len(bytes.TrimSpace(out)) == 0 {
		return transport.ToolError(fmt.Errorf("codex %s returned empty output", command))
	}
	if !json.Valid(out) {
		return transport.ToolError(fmt.Errorf("codex %s returned invalid JSON", command))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(out)}},
	}, nil, nil
}
