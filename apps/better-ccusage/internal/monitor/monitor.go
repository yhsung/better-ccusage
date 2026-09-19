package monitor

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/adapters"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
	"github.com/cobra91/better-ccusage/pkg/pricing"
	"github.com/cobra91/better-ccusage/pkg/terminal"
)

// Opts configures the live monitor.
type Opts struct {
	Loader          *data.Loader
	Prices          *pricing.PriceTable
	Mode            cost.CostMode
	RefreshInterval time.Duration
}

type model struct {
	opts    Opts
	log     *terminal.Logger
	content string
}

type tickMsg time.Time

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m model) Init() tea.Cmd { return tickCmd(m.opts.RefreshInterval) }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tickMsg:
		m.refresh()
		return m, tickCmd(m.opts.RefreshInterval)
	}
	return m, nil
}

func (m *model) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	entries, err := m.opts.Loader.Load(ctx)
	if err != nil {
		m.content = fmt.Sprintf("error: %v", err)
		return
	}
	entries = adapters.NewManager().Normalize(entries)
	buckets := cost.Aggregate(entries, cost.GroupByBlock)
	buckets = cost.ApplyPrices(buckets, m.opts.Prices, m.opts.Mode)
	m.content = renderBlocks(buckets)
}

func renderBlocks(buckets []cost.Bucket) string {
	var s string
	for _, b := range buckets {
		s += fmt.Sprintf("%s  tokens=%d  cost=$%.4f\n", b.Key, b.InputTokens+b.OutputTokens+b.CacheCreationTokens+b.CacheReadTokens, float64(b.Cost.Micros)/1_000_000)
	}
	return s
}

func (m model) View() string {
	return m.content + "\n(press q to quit)\n"
}

// Run starts the live monitor with the given options.
func Run(ctx context.Context, opts Opts) error {
	if opts.RefreshInterval == 0 {
		opts.RefreshInterval = 30 * time.Second
	}
	m := model{opts: opts, log: terminal.NewLoggerFromEnv()}
	m.refresh()
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err := p.Run()
	return err
}
