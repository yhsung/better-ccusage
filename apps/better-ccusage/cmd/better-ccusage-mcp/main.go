package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/config"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/mcp/server"
	"github.com/cobra91/better-ccusage/pkg/pricing"
	"github.com/cobra91/better-ccusage/pkg/terminal"
)

const version = "0.0.0-dev"

func main() {
	log := terminal.NewLoggerFromEnv()
	root := newRootCmd(log)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd(log *terminal.Logger) *cobra.Command {
	var mode, mcpType string
	var port int
	root := &cobra.Command{
		Use:     "better-ccusage-mcp",
		Short:   "Serve better-ccusage reports over MCP",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if mcpType == "stdio" {
				log = terminal.NewLogger(int(terminal.Silent), os.Stderr)
			}
			cfg, _ := config.Load(config.DefaultPath())
			dirs := config.ResolveDirs("", cfg)
			if len(dirs) == 0 {
				return fmt.Errorf("No valid Claude data directories found")
			}
			prices, err := pricing.LoadPrices(bytes.NewReader(pricing.EmbeddedPrices))
			if err != nil {
				log.Warn("failed to load embedded pricing: %v", err)
			}
			srv := server.NewServer(server.Opts{
				ConfigDir:   strings.Join(dirs, ","),
				DefaultMode: parseMode(mode),
				Prices:      prices,
				Version:     version,
			})
			switch mcpType {
			case "stdio":
				return srv.Run(cmd.Context(), &mcp.StdioTransport{})
			case "http":
				log.Info("MCP server is running on http://localhost:%d", port)
				return http.ListenAndServe(":"+strconv.Itoa(port), mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
					return srv
				}, &mcp.StreamableHTTPOptions{}))
			default:
				return fmt.Errorf("unsupported MCP type: %q", mcpType)
			}
		},
	}
	root.Flags().StringVarP(&mode, "mode", "m", "auto", "cost calculation mode for usage reports")
	root.Flags().StringVarP(&mcpType, "type", "t", "stdio", "transport type for MCP server")
	root.Flags().IntVarP(&port, "port", "p", 8080, "port for HTTP transport")
	return root
}

func parseMode(s string) cost.CostMode {
	switch s {
	case "calculate":
		return cost.CostCalculate
	case "display":
		return cost.CostDisplay
	default:
		return cost.CostAuto
	}
}
