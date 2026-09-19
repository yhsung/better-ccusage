package statusline

import (
	"fmt"
	"strings"

	"github.com/cobra91/better-ccusage/pkg/pricing"
	"github.com/cobra91/better-ccusage/pkg/terminal"
)

// Input is the data needed to render a statusline.
type Input struct {
	TotalCost   pricing.Money
	TotalTokens int64
	Models      []string
}

// Render produces a single-line statusline like:
// "claude-sonnet-4-5 (1234 tokens) $0.05"
func Render(in Input) string {
	costStr := fmt.Sprintf("$%.4f", float64(in.TotalCost.Micros)/1_000_000)
	tokensStr := fmt.Sprintf("%d tokens", in.TotalTokens)
	modelStr := "unknown"
	if len(in.Models) > 0 {
		modelStr = in.Models[0]
	}
	parts := []string{
		terminal.StyleText(terminal.StyleHeader, modelStr),
		terminal.StyleText(terminal.StyleMuted, tokensStr),
		terminal.StyleText(terminal.StyleCost, costStr),
	}
	return strings.Join(parts, " ")
}
