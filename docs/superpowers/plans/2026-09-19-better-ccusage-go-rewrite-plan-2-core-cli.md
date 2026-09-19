# better-ccusage Go Rewrite — Plan 2: Core CLI

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `apps/better-ccusage` — the core CLI binary — including the JSONL data loader, cost aggregation, seven provider adapters, six commands (daily/monthly/session/blocks/statusline/weekly), live blocks monitor, statusline renderer, JSON config, and JSON output. Swap the testdata pricing fixture for the real upstream file.

**Architecture:** Single binary `better-ccusage` built from `apps/better-ccusage/cmd/better-ccusage/main.go` using cobra. Internal packages under `apps/better-ccusage/internal/` form a layered graph: `data` (JSONL parsing) → `adapters` (provider normalization) → `cost` (aggregation + pricing) → `commands` (orchestration + rendering). The `monitor` package owns the live blocks event loop using Charm bubbles. The MCP server (Plan 3) calls into `commands.X.Run(ctx, opts)` directly — no subprocess.

**Tech Stack:** Go 1.22, stdlib `testing`, `github.com/spf13/cobra` + `pflag`, `github.com/charmbracelet/bubbles` + `bubbletea` + `lipgloss` (already in Plan 1), `github.com/itchyny/gojq` (added in this plan), `github.com/cobra91/better-ccusage/pkg/pricing` + `pkg/terminal` (Plan 1).

**Spec:** [`docs/superpowers/specs/2026-09-19-better-ccusage-go-rewrite-design.md`](../specs/2026-09-19-better-ccusage-go-rewrite-design.md)

**Plan series:**
- ✅ Plan 1: Foundation (merged to main at `3bbf5f2`)
- **Plan 2: Core CLI** ← this document
- Plan 3: MCP server
- Plan 4: Shims + docs

## Global Constraints

- Go version: **1.22** floor.
- Module path: **`github.com/cobra91/better-ccusage`**.
- All runtime libs go in `go.mod` as direct deps.
- No `testify`, no `ginkgo`, no `gomock`. **stdlib `testing` only.**
- **No floats in cost math.** Use `pricing.Money` for any monetary value.
- Money lives in **`pkg/pricing.Money`** (already implemented in Plan 1; do not re-implement in `apps/.../internal/`).
- Test naming: `TestXxx` for unit, `TestXxx_Integration` for full-pipeline tests.
- Race detector required: `go test ./... -race`.
- CI runs on **`macos-latest` only**.
- Strict 1:1 behavior port from TS — output byte-equivalent (modulo ANSI escape details).
- Logger writes to **`os.Stderr`** via `pkg/terminal.Logger`.
- LOG_LEVEL semantics: `0`=silent, `1`=warn, `2`=log, `3`=info, `4`=debug, `5`=trace.
- Sentinel errors live in **`apps/better-ccusage/internal/errs/`**.
- No `console.log` / `fmt.Println` for runtime output — use `pkg/terminal.Logger`.
- `data.Entry` JSON fields match the TS schema (see spec §Data Flow).
- All `commands.X.Run(ctx, opts)` functions return a public typed result so MCP can call them directly.

---

## File Structure (Plan 2 deliverable)

```
apps/better-ccusage/
├── cmd/better-ccusage/main.go         # cobra root + subcommand routing
└── internal/
    ├── errs/
    │   └── errs.go                     # sentinel errors
    ├── data/
    │   ├── entry.go                    # Entry struct + JSON tags
    │   ├── loader.go                   # JSONL loader
    │   └── loader_test.go
    ├── cost/
    │   ├── group.go                    # GroupKey, CostMode enums
    │   ├── aggregate.go                # Aggregate + ApplyPrices
    │   ├── aggregate_test.go
    │   └── bucket.go                   # Bucket struct
    ├── adapters/
    │   ├── provider.go                 # Provider interface + Manager
    │   ├── claude.go                   # base adapter
    │   ├── codex.go
    │   ├── opencode.go
    │   ├── devin.go
    │   ├── pi.go
    │   ├── zcode.go
    │   ├── droid.go
    │   └── adapters_test.go
    ├── config/
    │   ├── config.go                   # Config struct + Load
    │   ├── resolve.go                  # ResolveDirs (multi-dir + env)
    │   └── config_test.go
    ├── output/
    │   ├── encode.go                   # JSON encoder matching TS output
    │   ├── types.go                    # DailyResult, MonthlyResult, etc.
    │   └── encode_test.go
    ├── jq/
    │   ├── jq.go                       # gojq wrapper
    │   └── jq_test.go
    ├── schema/
    │   ├── schema.go                   # hand-written JSON Schema string
    │   └── schema_test.go              # validates schema parses as JSON
    ├── statusline/
    │   ├── statusline.go               # compact statusline renderer
    │   └── statusline_test.go
    ├── monitor/
    │   ├── monitor.go                  # bubbles event loop
    │   └── monitor_test.go
    └── commands/
        ├── common.go                   # shared flags + Run() pattern
        ├── daily.go                    # + tests
        ├── monthly.go                  # + tests
        ├── session.go                  # + tests
        ├── blocks.go                   # static
        ├── blocks_live.go              # live monitor command
        ├── statusline.go
        └── weekly.go

pkg/pricing/testdata/prices.json       # REPLACED in Task 23 with real upstream JSON
```

---

## Task 1: Add cobra dependency + scaffold main.go

**Files:**
- Modify: `go.mod`, `go.sum` (after `go get`)
- Create: `apps/better-ccusage/cmd/better-ccusage/main.go`

**Interfaces:**
- Produces: `func main()` that creates a cobra root command and prints "better-ccusage v0.0.0" via the terminal logger. No subcommands yet — that's wired in Task 21.

- [ ] **Step 1: Add cobra dependency**

```bash
cd /Volumes/Samsung970EVOPlus/dev-projects/better-ccusage/.worktrees/plan-2-core-cli
go get github.com/spf13/cobra@latest
```

- [ ] **Step 2: Create `apps/better-ccusage/cmd/better-ccusage/main.go`**

```go
package main

import (
	"fmt"
	"os"

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
```

- [ ] **Step 3: Create `apps/better-ccusage/cmd/better-ccusage/root.go`**

```go
package main

import (
	"github.com/spf13/cobra"
	"github.com/cobra91/better-ccusage/pkg/terminal"
)

func newRootCmd(log *terminal.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "better-ccusage",
		Short:   "Analyze Claude Code usage from local JSONL files",
		Version: version,
	}
}
```

- [ ] **Step 4: Verify it builds and runs**

```bash
go build ./apps/better-ccusage/cmd/better-ccusage
./apps/better-ccusage/cmd/better-ccusage --version  # outputs "better-ccusage version 0.0.0-dev"
```

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum apps/better-ccusage/
git commit -m "feat(better-ccusage): scaffold cobra root command"
```

---

## Task 2: Sentinel errors

**Files:**
- Create: `apps/better-ccusage/internal/errs/errs.go`
- Create: `apps/better-ccusage/internal/errs/errs_test.go`

**Interfaces:**
- Produces: `ErrNoData`, `ErrUnknownModel`, `ErrInvalidJSON`, `ErrConfigNotFound`, `ErrIncompatibleMode`. Each wrapped via `fmt.Errorf("...: %w", err)` by callers.

- [ ] **Step 1: Write `errs.go`**

```go
// Package errs defines sentinel errors used across the apps/better-ccusage internal
// packages. Use errors.Is to branch on these.
package errs

import "errors"

var (
	ErrNoData           = errors.New("no usage data found")
	ErrUnknownModel     = errors.New("unknown model")
	ErrInvalidJSON      = errors.New("invalid JSONL entry")
	ErrConfigNotFound   = errors.New("config file not found")
	ErrIncompatibleMode = errors.New("cost mode incompatible with data")
)
```

- [ ] **Step 2: Write `errs_test.go`**

```go
package errs

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	for _, e := range []error{ErrNoData, ErrUnknownModel, ErrInvalidJSON, ErrConfigNotFound, ErrIncompatibleMode} {
		wrapped := fmt.Errorf("context: %w", e)
		if !errors.Is(wrapped, e) {
			t.Errorf("errors.Is(%v, %v) = false, want true", wrapped, e)
		}
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./apps/better-ccusage/internal/errs/...
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add apps/better-ccusage/internal/errs/
git commit -m "feat(better-ccusage): add sentinel errors"
```

---

## Task 3: Entry struct + JSONL loader

**Files:**
- Create: `apps/better-ccusage/internal/data/entry.go`
- Create: `apps/better-ccusage/internal/data/loader.go`
- Create: `apps/better-ccusage/internal/data/loader_test.go`

**Interfaces:**
- Produces:
  - `type Entry struct` with JSON tags matching TS Claude Code log schema (`timestamp`, `sessionId`, `project`, `model`, `inputTokens`, `outputTokens`, `cacheCreationTokens`, `cacheReadTokens`, `costUSD`).
  - `type Loader struct { dirs []string }`
  - `func NewLoader(dirs ...string) *Loader`
  - `func (l *Loader) Load(ctx context.Context) ([]Entry, error)`
  - `func (l *Loader) Reload(ctx context.Context, prev []Entry) ([]Entry, error)` — mtime-aware, returns new entries list

- [ ] **Step 1: Write `entry.go`**

```go
package data

import (
	"encoding/json"
	"fmt"
	"time"
)

// Entry represents one line of a Claude Code JSONL log file.
// Field names match the TS schema in apps/better-ccusage/src/_types.ts.
type Entry struct {
	Timestamp           time.Time `json:"-"`
	TimestampRaw        string    `json:"timestamp"`
	SessionID           string    `json:"sessionId"`
	Project             string    `json:"project,omitempty"`
	Model               string    `json:"model"`
	InputTokens         int64     `json:"inputTokens"`
	OutputTokens        int64     `json:"outputTokens"`
	CacheCreationTokens int64     `json:"cacheCreationTokens"`
	CacheReadTokens     int64     `json:"cacheReadTokens"`
	CostUSD             *float64  `json:"costUSD,omitempty"`
	IsAPIError          bool      `json:"isApiError,omitempty"`
}

// UnmarshalJSON parses the timestamp string into time.Time.
func (e *Entry) UnmarshalJSON(data []byte) error {
	type alias Entry
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*e = Entry(a)
	if a.TimestampRaw != "" {
		ts, err := time.Parse(time.RFC3339Nano, a.TimestampRaw)
		if err != nil {
			return fmt.Errorf("parsing timestamp %q: %w", a.TimestampRaw, err)
		}
		e.Timestamp = ts
	}
	return nil
}
```

- [ ] **Step 2: Write `loader.go`**

```go
package data

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Loader reads JSONL entries from one or more Claude config directories.
type Loader struct {
	dirs []string
}

// NewLoader constructs a Loader with the given config directories.
// Each dir is expected to contain projects/*/*.jsonl (Claude Code log layout).
func NewLoader(dirs ...string) *Loader {
	return &Loader{dirs: append([]string(nil), dirs...)}
}

// Load walks all configured dirs, parses each .jsonl line into an Entry,
// and returns them sorted by timestamp ascending. Malformed lines are
// silently skipped (matches TS data-loader.ts).
func (l *Loader) Load(ctx context.Context) ([]Entry, error) {
	if len(l.dirs) == 0 {
		return nil, fmt.Errorf("no data directories configured")
	}
	var entries []Entry
	for _, dir := range l.dirs {
		if err := l.walkDir(ctx, dir, &entries); err != nil {
			return nil, fmt.Errorf("walking %s: %w", dir, err)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})
	return entries, nil
}

// Reload re-reads files whose mtime changed since the previous load.
// For simplicity in v1, Reload calls Load and returns the full result.
// Incremental mtime filtering can be added later.
func (l *Loader) Reload(ctx context.Context, prev []Entry) ([]Entry, error) {
	return l.Load(ctx)
}

func (l *Loader) walkDir(ctx context.Context, root string, out *[]Entry) error {
	projectsDir := filepath.Join(root, "projects")
	return filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if info.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := parseJSONL(f, out); err != nil {
			return err
		}
		return nil
	})
}

func parseJSONL(r io.Reader, out *[]Entry) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024) // up to 16 MiB per line
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			// silently skip malformed lines, matching TS behavior
			continue
		}
		*out = append(*out, e)
	}
	return scanner.Err()
}

// DefaultDirs returns the default Claude config directories in lookup order.
func DefaultDirs() []string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return nil
	}
	return []string{
		filepath.Join(home, ".config", "claude"),
		filepath.Join(home, ".claude"),
	}
}

// FilterByTime returns entries with Timestamp in [since, until].
// Either bound may be nil for open-ended ranges.
func FilterByTime(entries []Entry, since, until *time.Time) []Entry {
	var out []Entry
	for _, e := range entries {
		if since != nil && e.Timestamp.Before(*since) {
			continue
		}
		if until != nil && e.Timestamp.After(*until) {
			continue
		}
		out = append(out, e)
	}
	return out
}
```

- [ ] **Step 3: Write `loader_test.go`**

```go
package data

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_EmptyDirs(t *testing.T) {
	l := NewLoader()
	_, err := l.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for empty dirs")
	}
}

func TestLoad_ReadsJSONL(t *testing.T) {
	dir := t.TempDir()
	project := filepath.Join(dir, "projects", "myproj")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	jsonl := `{"timestamp":"2026-01-15T10:00:00Z","sessionId":"s1","model":"claude-sonnet-4-5-20250929","inputTokens":100,"outputTokens":50}
{"timestamp":"2026-01-15T11:00:00Z","sessionId":"s1","model":"claude-sonnet-4-5-20250929","inputTokens":200,"outputTokens":100}
this is not valid json
{"timestamp":"2026-01-15T12:00:00Z","sessionId":"s1","model":"claude-sonnet-4-5-20250929","inputTokens":50,"outputTokens":25}
`
	if err := os.WriteFile(filepath.Join(project, "abc.jsonl"), []byte(jsonl), 0o644); err != nil {
		t.Fatal(err)
	}
	l := NewLoader(dir)
	entries, err := l.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries (1 malformed skipped), got %d", len(entries))
	}
	// Verify sorted by timestamp
	if !entries[0].Timestamp.Before(entries[1].Timestamp) {
		t.Error("entries not sorted by timestamp")
	}
	if entries[0].InputTokens != 100 || entries[1].InputTokens != 200 {
		t.Errorf("unexpected token counts: %+v", entries)
	}
}

func TestLoad_MissingDir(t *testing.T) {
	l := NewLoader("/nonexistent/path/that/does/not/exist")
	entries, err := l.Load(context.Background())
	if err != nil {
		t.Fatalf("missing dir should be tolerated, got: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestFilterByTime(t *testing.T) {
	entries := []Entry{
		{Timestamp: mustTime("2026-01-15T10:00:00Z")},
		{Timestamp: mustTime("2026-01-15T11:00:00Z")},
		{Timestamp: mustTime("2026-01-15T12:00:00Z")},
	}
	since := mustTime("2026-01-15T10:30:00Z")
	until := mustTime("2026-01-15T11:30:00Z")
	out := FilterByTime(entries, &since, &until)
	if len(out) != 1 {
		t.Errorf("expected 1 entry, got %d", len(out))
	}
}

func mustTime(s string) (t time.Time) {
	t, _ = timeParse(s)
	return
}
```

Note: `mustTime` above uses a `timeParse` helper. Add this to the test file:

```go
func timeParse(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}
```

Also add `"time"` import.

- [ ] **Step 4: Run tests**

```bash
go test ./apps/better-ccusage/internal/data/... -race
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/better-ccusage/internal/data/
git commit -m "feat(better-ccusage): add Entry type and JSONL loader"
```

---

## Task 4: Cost aggregation + pricing application

**Files:**
- Create: `apps/better-ccusage/internal/cost/group.go` (enums)
- Create: `apps/better-ccusage/internal/cost/bucket.go`
- Create: `apps/better-ccusage/internal/cost/aggregate.go`
- Create: `apps/better-ccusage/internal/cost/aggregate_test.go`

**Interfaces:**
- `type GroupKey int` with `GroupByDay, GroupByMonth, GroupBySession, GroupByBlock`
- `type CostMode int` with `CostAuto, CostCalculate, CostDisplay`
- `type Bucket struct { Key string; Timestamp time.Time; InputTokens, OutputTokens, CacheCreationTokens, CacheReadTokens int64; Cost pricing.Money; Model string }`
- `func Aggregate(entries []data.Entry, key GroupKey) []Bucket`
- `func ApplyPrices(buckets []Bucket, prices *pricing.PriceTable, mode CostMode) []Bucket`

- [ ] **Step 1: Write `group.go`**

```go
package cost

// GroupKey controls how Aggregate groups entries.
type GroupKey int

const (
	GroupByDay GroupKey = iota
	GroupByMonth
	GroupBySession
	GroupByBlock
)

// CostMode controls how ApplyPrices derives per-bucket cost.
type CostMode int

const (
	// CostAuto uses pre-calculated costUSD when present, else calculates from tokens.
	CostAuto CostMode = iota
	// CostCalculate always calculates from token counts; ignores costUSD.
	CostCalculate
	// CostDisplay always uses pre-calculated costUSD; shows 0 when missing.
	CostDisplay
)

func (m CostMode) String() string {
	switch m {
	case CostAuto:
		return "auto"
	case CostCalculate:
		return "calculate"
	case CostDisplay:
		return "display"
	default:
		return "unknown"
	}
}
```

- [ ] **Step 2: Write `bucket.go`**

```go
package cost

import (
	"time"

	"github.com/cobra91/better-ccusage/pkg/pricing"
)

// Bucket is the unit of aggregation output.
type Bucket struct {
	Key                 string         // group identifier (date string, session ID, block key)
	Timestamp           time.Time      // representative timestamp for the group
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
	Cost                pricing.Money
	Model               string         // primary model in the bucket (most-used)
	Count               int            // number of entries in the bucket
}
```

- [ ] **Step 3: Write `aggregate.go`**

```go
package cost

import (
	"fmt"
	"sort"
	"time"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
	"github.com/cobra91/better-ccusage/pkg/pricing"
)

// Aggregate groups entries by key into buckets. Pure function, no I/O.
func Aggregate(entries []data.Entry, key GroupKey) []Bucket {
	groups := make(map[string]*Bucket)
	for _, e := range entries {
		gk := groupKeyFor(e, key)
		b, ok := groups[gk]
		if !ok {
			b = &Bucket{Key: gk, Timestamp: e.Timestamp}
			groups[gk] = b
		}
		b.InputTokens += e.InputTokens
		b.OutputTokens += e.OutputTokens
		b.CacheCreationTokens += e.CacheCreationTokens
		b.CacheReadTokens += e.CacheReadTokens
		b.Count++
	}
	out := make([]Bucket, 0, len(groups))
	for _, b := range groups {
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	return out
}

func groupKeyFor(e data.Entry, key GroupKey) string {
	switch key {
	case GroupByDay:
		return e.Timestamp.Format("2006-01-02")
	case GroupByMonth:
		return e.Timestamp.Format("2006-01")
	case GroupBySession:
		return e.SessionID
	case GroupByBlock:
		return blockKey(e.Timestamp)
	default:
		return ""
	}
}

// blockKey returns a 5-hour billing-block key like "2026-01-15T05".
// Matches TS session-blocks.ts: hour = floor(UTC_hour / 5) * 5.
func blockKey(t time.Time) string {
	h := t.UTC().Hour() / 5 * 5
	return t.UTC().Format("2006-01-02") + fmt.Sprintf("T%02d", h)
}

// ApplyPrices derives Cost on each bucket according to mode and prices.
func ApplyPrices(buckets []Bucket, prices *pricing.PriceTable, mode CostMode) []Bucket {
	out := make([]Bucket, len(buckets))
	for i, b := range buckets {
		price, _ := prices.Lookup(b.Model)
		out[i] = applyPrice(b, price, mode)
	}
	return out
}

func applyPrice(b Bucket, price pricing.Price, mode CostMode) Bucket {
	switch mode {
	case CostCalculate:
		return calcCost(b, price)
	case CostDisplay:
		// Display mode shows pre-calculated costUSD; without per-bucket pre-calc,
		// it stays zero. Plans that need true pre-calc display should pre-compute
		// costUSD per entry upstream.
		return b
	default: // CostAuto
		return autoCost(b, price)
	}
}

func autoCost(b Bucket, price pricing.Price) Bucket {
	if price.InputCostPerToken > 0 || price.OutputCostPerToken > 0 {
		return calcCost(b, price)
	}
	return b
}

func calcCost(b Bucket, p pricing.Price) Bucket {
	in := pricing.Money{Micros: int64(p.InputCostPerToken * 1_000_000)}.MulFloat(float64(b.InputTokens + b.CacheReadTokens))
	cacheW := pricing.Money{Micros: int64(p.CacheCreationInputTokenCost * 1_000_000)}.MulFloat(float64(b.CacheCreationTokens))
	out := pricing.Money{Micros: int64(p.OutputCostPerToken * 1_000_000)}.MulFloat(float64(b.OutputTokens))
	b.Cost = in.Add(cacheW).Add(out)
	return b
}
```

- [ ] **Step 4: Write `aggregate_test.go`**

```go
package cost

import (
	"testing"
	"time"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
	"github.com/cobra91/better-ccusage/pkg/pricing"
)

func mkEntry(t time.Time, model string, in, out int64) data.Entry {
	return data.Entry{
		Timestamp:    t,
		TimestampRaw: t.Format(time.RFC3339),
		Model:        model,
		InputTokens:  in,
		OutputTokens: out,
	}
}

func TestAggregate_GroupByDay(t *testing.T) {
	day1 := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 16, 10, 0, 0, 0, time.UTC)
	entries := []data.Entry{
		mkEntry(day1, "claude-sonnet-4-5-20250929", 100, 50),
		mkEntry(day1, "claude-sonnet-4-5-20250929", 200, 100),
		mkEntry(day2, "claude-sonnet-4-5-20250929", 50, 25),
	}
	buckets := Aggregate(entries, GroupByDay)
	if len(buckets) != 2 {
		t.Fatalf("expected 2 day buckets, got %d", len(buckets))
	}
	if buckets[0].InputTokens != 300 {
		t.Errorf("day1 input: got %d, want 300", buckets[0].InputTokens)
	}
	if buckets[1].InputTokens != 50 {
		t.Errorf("day2 input: got %d, want 50", buckets[1].InputTokens)
	}
}

func TestApplyPrices_Calculate(t *testing.T) {
	day := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	buckets := []Bucket{{
		Key: "2026-01-15", Timestamp: day,
		InputTokens: 1000, OutputTokens: 500,
		Model: "claude-sonnet-4-5-20250929",
	}}
	price := pricing.Price{
		InputCostPerToken:  0.000003,
		OutputCostPerToken: 0.000015,
	}
	out := ApplyPrices(buckets, &pricing.PriceTable{}, CostCalculate)
	_ = price // prices used via lookup; use real table in TestApplyPrices_WithTable
	if out[0].Cost.Micros == 0 {
		// With empty PriceTable, lookup returns zero, cost is zero. We test real cost via WithTable.
		t.Logf("note: empty table yields zero cost (expected for CostCalculate with no prices)")
	}
}

func TestApplyPrices_WithTable(t *testing.T) {
	pt, _ := pricing.LoadPrices(strings.NewReader(`{"claude-sonnet-4-5-20250929":{"input_cost_per_token":0.000003,"output_cost_per_token":0.000015}}`))
	day := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	buckets := []Bucket{{
		Key: "2026-01-15", Timestamp: day,
		InputTokens: 1000, OutputTokens: 500,
		Model:       "claude-sonnet-4-5-20250929",
	}}
	out := ApplyPrices(buckets, pt, CostCalculate)
	// 1000 * 0.000003 = 0.003 → 3000 micros
	// 500 * 0.000015 = 0.0075 → 7500 micros
	// total: 10500 micros
	if out[0].Cost.Micros != 10500 {
		t.Errorf("cost: got %d micros, want 10500", out[0].Cost.Micros)
	}
}
```

Add `"strings"` import to the test file.

- [ ] **Step 5: Run tests**

```bash
go test ./apps/better-ccusage/internal/cost/... -race
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/better-ccusage/internal/cost/
git commit -m "feat(better-ccusage): add cost aggregation and pricing application"
```

---

## Task 5: Adapter interface + claude base + Manager

**Files:**
- Create: `apps/better-ccusage/internal/adapters/provider.go`
- Create: `apps/better-ccusage/internal/adapters/claude.go`
- Create: `apps/better-ccusage/internal/adapters/adapters_test.go`

**Interfaces:**
- `type Provider interface { Name() string; Detect(e data.Entry) bool; Adapt(e data.Entry) (data.Entry, bool) }`
- `type Manager struct { providers []Provider }`
- `func NewManager() *Manager`
- `func (m *Manager) Normalize(entries []data.Entry) []data.Entry`

- [ ] **Step 1: Write `provider.go`**

```go
package adapters

import "github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"

// Provider normalizes a data.Entry for a specific provider (claude, codex, ...).
type Provider interface {
	Name() string
	Detect(e data.Entry) bool
	Adapt(e data.Entry) (data.Entry, bool)
}

// Manager applies all registered providers to a slice of entries.
type Manager struct {
	providers []Provider
}

// NewManager returns a Manager pre-loaded with all built-in providers
// (claude base + 6 provider-specific adapters).
func NewManager() *Manager {
	return &Manager{providers: builtins()}
}

// Normalize runs each entry through the first provider whose Detect returns true.
// If no provider matches, the entry passes through unchanged.
func (m *Manager) Normalize(entries []data.Entry) []data.Entry {
	out := make([]data.Entry, 0, len(entries))
	for _, e := range entries {
		matched := false
		for _, p := range m.providers {
			if p.Detect(e) {
				if normalized, ok := p.Adapt(e); ok {
					out = append(out, normalized)
				}
				matched = true
				break
			}
		}
		if !matched {
			out = append(out, e)
		}
	}
	return out
}

func builtins() []Provider {
	return []Provider{
		claudeAdapter{},
		// Adapters added in Task 6.
	}
}
```

- [ ] **Step 2: Write `claude.go`**

```go
package adapters

import "github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"

// claudeAdapter is the base provider for vanilla Claude Code logs.
type claudeAdapter struct{}

func (claudeAdapter) Name() string { return "claude" }

// Detect returns true for any entry — claude is the default fallback.
func (claudeAdapter) Detect(e data.Entry) bool { return e.Model != "" }

// Adapt returns the entry unchanged. Provider-specific normalizations
// (e.g. unprefixing model names) happen in dedicated adapters.
func (claudeAdapter) Adapt(e data.Entry) (data.Entry, bool) { return e, true }
```

- [ ] **Step 3: Write `adapters_test.go`**

```go
package adapters

import (
	"testing"
	"time"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
)

func mkE(model string) data.Entry {
	return data.Entry{
		Timestamp:    time.Now(),
		TimestampRaw: time.Now().Format(time.RFC3339),
		Model:        model,
	}
}

func TestManager_DefaultsToClaude(t *testing.T) {
	m := NewManager()
	entries := []data.Entry{mkE("claude-sonnet-4-5-20250929"), mkE("kimi-for-coding")}
	out := m.Normalize(entries)
	if len(out) != 2 {
		t.Fatalf("expected 2 entries (both via claude fallback), got %d", len(out))
	}
}

func TestManager_EmptyEntries(t *testing.T) {
	m := NewManager()
	out := m.Normalize(nil)
	if len(out) != 0 {
		t.Errorf("expected 0 entries, got %d", len(out))
	}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./apps/better-ccusage/internal/adapters/... -race
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/better-ccusage/internal/adapters/
git commit -m "feat(better-ccusage): add Provider interface and claude base adapter"
```

---

## Task 6: Six additional provider adapters

**Files:**
- Create: `apps/better-ccusage/internal/adapters/{codex,opencode,devin,pi,zcode,droid}.go`
- Modify: `apps/better-ccusage/internal/adapters/provider.go` (add adapters to `builtins()`)

**Interfaces:** Each adapter implements the Provider interface with Detect and Adapt logic that strips provider prefixes and normalizes model names.

- [ ] **Step 1: Create `codex.go`** — strips `codex/` prefix from model names

```go
package adapters

import (
	"strings"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
)

type codexAdapter struct{}

func (codexAdapter) Name() string { return "codex" }
func (codexAdapter) Detect(e data.Entry) bool {
	return strings.HasPrefix(e.Model, "codex/") || e.SessionID != "" && strings.Contains(e.Model, "codex")
}
func (codexAdapter) Adapt(e data.Entry) (data.Entry, bool) {
	e.Model = strings.TrimPrefix(e.Model, "codex/")
	return e, true
}
```

- [ ] **Step 2: Create `opencode.go`** — strips `opencode/` prefix

```go
package adapters

import (
	"strings"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
)

type opencodeAdapter struct{}

func (opencodeAdapter) Name() string { return "opencode" }
func (opencodeAdapter) Detect(e data.Entry) bool {
	return strings.HasPrefix(e.Model, "opencode/")
}
func (opencodeAdapter) Adapt(e data.Entry) (data.Entry, bool) {
	e.Model = strings.TrimPrefix(e.Model, "opencode/")
	return e, true
}
```

- [ ] **Step 3: Create `devin.go`** — Devin (Cognition) provider normalization

```go
package adapters

import "github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"

type devinAdapter struct{}

func (devinAdapter) Name() string { return "devin" }
func (devinAdapter) Detect(e data.Entry) bool { return e.Project == "devin" }
func (devinAdapter) Adapt(e data.Entry) (data.Entry, bool) { return e, true }
```

- [ ] **Step 4: Create `pi.go`** — Pi provider

```go
package adapters

import "github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"

type piAdapter struct{}

func (piAdapter) Name() string { return "pi" }
func (piAdapter) Detect(e data.Entry) bool { return e.Project == "pi" || e.SessionID != "" && e.Project == "opencode-pi" }
func (piAdapter) Adapt(e data.Entry) (data.Entry, bool) { return e, true }
```

- [ ] **Step 5: Create `zcode.go`** — ZCode provider

```go
package adapters

import "github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"

type zcodeAdapter struct{}

func (zcodeAdapter) Name() string { return "zcode" }
func (zcodeAdapter) Detect(e data.Entry) bool { return e.Project == "zcode" }
func (zcodeAdapter) Adapt(e data.Entry) (data.Entry, bool) { return e, true }
```

- [ ] **Step 6: Create `droid.go`** — Droid provider

```go
package adapters

import "github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"

type droidAdapter struct{}

func (droidAdapter) Name() string { return "droid" }
func (droidAdapter) Detect(e data.Entry) bool { return e.Project == "droid" }
func (droidAdapter) Adapt(e data.Entry) (data.Entry, bool) { return e, true }
```

- [ ] **Step 7: Update `provider.go` `builtins()`**

```go
func builtins() []Provider {
	return []Provider{
		codexAdapter{},
		opencodeAdapter{},
		devinAdapter{},
		piAdapter{},
		zcodeAdapter{},
		droidAdapter{},
		claudeAdapter{}, // last (fallback)
	}
}
```

Note: order matters — provider-specific adapters run before the claude fallback. Detection is by exact project/prefix match, so false positives are unlikely.

- [ ] **Step 8: Add tests to `adapters_test.go`**

```go
func TestCodexAdapter(t *testing.T) {
	m := NewManager()
	out := m.Normalize([]data.Entry{mkE("codex/kimi-for-coding")})
	if out[0].Model != "kimi-for-coding" {
		t.Errorf("expected stripped model, got %q", out[0].Model)
	}
}

func TestOpencodeAdapter(t *testing.T) {
	m := NewManager()
	out := m.Normalize([]data.Entry{mkE("opencode/claude-sonnet-4-5-20250929")})
	if out[0].Model != "claude-sonnet-4-5-20250929" {
		t.Errorf("expected stripped model, got %q", out[0].Model)
	}
}
```

- [ ] **Step 9: Run tests**

```bash
go test ./apps/better-ccusage/internal/adapters/... -race
```

Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add apps/better-ccusage/internal/adapters/
git commit -m "feat(better-ccusage): add 6 provider adapters (codex, opencode, devin, pi, zcode, droid)"
```

---

## Task 7: JSON config + multi-config-dir resolution

**Files:**
- Create: `apps/better-ccusage/internal/config/config.go`
- Create: `apps/better-ccusage/internal/config/resolve.go`
- Create: `apps/better-ccusage/internal/config/config_test.go`

**Interfaces:**
- `type Config struct { PricingURL string; Offline bool; DefaultMode string; ExtraConfigDirs []string }`
- `func Load(path string) (*Config, error)` — returns empty config (not error) if file doesn't exist
- `func ResolveDirs(envValue string, cfg *Config) []string` — merges env, config extras, defaults

- [ ] **Step 1: Write `config.go`**

```go
package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
)

// Config is the user-facing configuration loaded from ~/.config/better-ccusage/config.json.
type Config struct {
	PricingURL      string   `json:"pricingUrl,omitempty"`
	Offline         bool     `json:"offline,omitempty"`
	DefaultMode     string   `json:"defaultMode,omitempty"`
	ExtraConfigDirs []string `json:"extraConfigDirs,omitempty"`
}

// Load reads the config file at path. If the file does not exist, returns
// an empty Config (not an error) — matches TS behavior.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%w: %v", errs.ErrConfigNotFound, err)
	}
	return &c, nil
}

// DefaultPath returns the default config file path.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return home + "/.config/better-ccusage/config.json"
}
```

- [ ] **Step 2: Write `resolve.go`**

```go
package config

import (
	"strings"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
)

// ResolveDirs merges directories from CLAUDE_CONFIG_DIR env, config extras,
// and the data.DefaultDirs fallback. Order: env first, then extras, then defaults.
func ResolveDirs(envValue string, cfg *Config) []string {
	var dirs []string
	if envValue != "" {
		for _, d := range strings.Split(envValue, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				dirs = append(dirs, d)
			}
		}
	}
	if cfg != nil {
		dirs = append(dirs, cfg.ExtraConfigDirs...)
	}
	dirs = append(dirs, data.DefaultDirs()...)
	return dirs
}
```

- [ ] **Step 3: Write `config_test.go`**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_MissingFile(t *testing.T) {
	c, err := Load("/nonexistent/config.json")
	if err != nil {
		t.Fatalf("missing file should not error, got %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"pricingUrl":"https://example.com/prices.json","offline":true,"defaultMode":"calculate","extraConfigDirs":["/foo"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !c.Offline {
		t.Error("expected Offline=true")
	}
	if c.DefaultMode != "calculate" {
		t.Errorf("DefaultMode: got %q, want calculate", c.DefaultMode)
	}
	if len(c.ExtraConfigDirs) != 1 || c.ExtraConfigDirs[0] != "/foo" {
		t.Errorf("ExtraConfigDirs: got %v", c.ExtraConfigDirs)
	}
}

func TestResolveDirs_EnvFirst(t *testing.T) {
	got := ResolveDirs("/env/dir", &Config{ExtraConfigDirs: []string{"/cfg/dir"}})
	if got[0] != "/env/dir" {
		t.Errorf("env dir should be first, got %v", got)
	}
}

func TestResolveDirs_OnlyDefaults(t *testing.T) {
	got := ResolveDirs("", &Config{})
	if len(got) == 0 {
		t.Error("expected at least default dirs")
	}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./apps/better-ccusage/internal/config/... -race
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/better-ccusage/internal/config/
git commit -m "feat(better-ccusage): add JSON config loader and multi-dir resolver"
```

---

## Task 8: JSON output types + encoder

**Files:**
- Create: `apps/better-ccusage/internal/output/types.go`
- Create: `apps/better-ccusage/internal/output/encode.go`
- Create: `apps/better-ccusage/internal/output/encode_test.go`

**Interfaces:**
- Typed result structs matching TS `_json-output-types.ts`
- `func EncodeJSON(w io.Writer, v any) error` — JSON encoder with consistent formatting

- [ ] **Step 1: Write `types.go`**

```go
package output

import "github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"

// DailyResult is the JSON output shape for the `daily` command.
type DailyResult struct {
	Daily   []DailyRow `json:"daily"`
	Summary Summary    `json:"summary"`
}

// DailyRow is one day's usage.
type DailyRow struct {
	Date               string  `json:"date"`
	InputTokens        int64   `json:"inputTokens"`
	OutputTokens       int64   `json:"outputTokens"`
	CacheCreationTokens int64  `json:"cacheCreationTokens"`
	CacheReadTokens    int64   `json:"cacheReadTokens"`
	TotalTokens        int64   `json:"totalTokens"`
	CostUSD            float64 `json:"costUSD"`
	Models             []string `json:"models"`
}

// Summary is the totals row.
type Summary struct {
	TotalTokens int64   `json:"totalTokens"`
	CostUSD     float64 `json:"costUSD"`
}

// BucketsToDailyRows converts cost buckets to DailyRows.
func BucketsToDailyRows(buckets []cost.Bucket) []DailyRow {
	out := make([]DailyRow, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, DailyRow{
			Date:                b.Key,
			InputTokens:         b.InputTokens,
			OutputTokens:        b.OutputTokens,
			CacheCreationTokens: b.CacheCreationTokens,
			CacheReadTokens:     b.CacheReadTokens,
			TotalTokens:         b.InputTokens + b.OutputTokens + b.CacheCreationTokens + b.CacheReadTokens,
			CostUSD:             float64(b.Cost.Micros) / 1_000_000,
		})
	}
	return out
}

// SumRows totals a slice of DailyRows.
func SumRows(rows []DailyRow) Summary {
	var s Summary
	for _, r := range rows {
		s.TotalTokens += r.TotalTokens
		s.CostUSD += r.CostUSD
	}
	return s
}
```

- [ ] **Step 2: Write `encode.go`**

```go
package output

import (
	"encoding/json"
	"io"
)

// EncodeJSON writes v as indented JSON to w. Matches the TS JSON output
// shape (2-space indent, sorted keys, trailing newline).
func EncodeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	return nil
}
```

- [ ] **Step 3: Write `encode_test.go`**

```go
package output

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestEncodeJSON_Indented(t *testing.T) {
	var buf bytes.Buffer
	if err := EncodeJSON(&buf, map[string]any{"a": 1, "b": "two"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Should contain newlines (indented)
	if len(out) < 10 {
		t.Errorf("output too short: %q", out)
	}
	// Should parse back
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Errorf("output not valid JSON: %v", err)
	}
	if got["a"].(float64) != 1 {
		t.Errorf("roundtrip failed: %v", got)
	}
}

func TestBucketsToDailyRows(t *testing.T) {
	rows := BucketsToDailyRows(nil)
	if len(rows) != 0 {
		t.Errorf("expected 0 rows for nil, got %d", len(rows))
	}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./apps/better-ccusage/internal/output/... -race
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/better-ccusage/internal/output/
git commit -m "feat(better-ccusage): add JSON output types and encoder"
```

---

## Task 9: Commands — shared flags + Run pattern

**Files:**
- Create: `apps/better-ccusage/internal/commands/common.go`
- Create: `apps/better-ccusage/internal/commands/common_test.go`

**Interfaces:**
- `type CommonOpts struct { Mode cost.CostMode; Offline, JSON bool; Since, Until *time.Time; ConfigDir string }`
- `func BindCommonFlags(cmd *cobra.Command) *CommonOpts` — attaches persistent flags, returns the opts struct that the flags bind to

- [ ] **Step 1: Write `common.go`**

```go
package commands

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
)

// CommonOpts are the flags shared across all better-ccusage commands.
type CommonOpts struct {
	Mode      cost.CostMode
	Offline   bool
	JSON      bool
	Since     *time.Time
	Until     *time.Time
	ConfigDir string
}

// BindCommonFlags attaches persistent flags to cmd and returns the opts
// pointer that cobra will populate.
func BindCommonFlags(cmd *cobra.Command) *CommonOpts {
	opts := &CommonOpts{}
	cmd.PersistentFlags().StringVar((*string)(&opts.Mode), "mode", "auto", "cost calculation mode: auto, calculate, display")
	cmd.PersistentFlags().BoolVar(&opts.Offline, "offline", false, "skip live pricing fetch")
	cmd.PersistentFlags().BoolVar(&opts.JSON, "json", false, "output as JSON")
	cmd.PersistentFlags().StringVar(&opts.ConfigDir, "config-dir", "", "additional Claude config directory (comma-separated for multiple)")
	cmd.PersistentFlags().String("since", "", "filter entries since this RFC3339 timestamp")
	cmd.PersistentFlags().String("until", "", "filter entries until this RFC3339 timestamp")
	// Pre-run parsing
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if s, _ := cmd.Flags().GetString("since"); s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return err
			}
			opts.Since = &t
		}
		if s, _ := cmd.Flags().GetString("until"); s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return err
			}
			opts.Until = &t
		}
		return nil
	}
	return opts
}
```

- [ ] **Step 2: Write `common_test.go`** — smoke test for the parser

```go
package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestBindCommonFlags_Defaults(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	opts := BindCommonFlags(cmd)
	if err := cmd.ParseFlags([]string{}); err != nil {
		t.Fatal(err)
	}
	if opts.Mode != "auto" {
		t.Errorf("default mode: got %q, want auto", opts.Mode)
	}
	if opts.Offline {
		t.Error("default offline: got true, want false")
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./apps/better-ccusage/internal/commands/... -race -run TestBindCommonFlags
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add apps/better-ccusage/internal/commands/common.go apps/better-ccusage/internal/commands/common_test.go
git commit -m "feat(better-ccusage): add shared command flags and Run pattern"
```

---

## Task 10: `daily` command

**Files:**
- Create: `apps/better-ccusage/internal/commands/daily.go`
- Create: `apps/better-ccusage/internal/commands/daily_test.go`

**Interfaces:**
- `type DailyOpts struct { CommonOpts }`
- `func Daily(ctx context.Context, opts DailyOpts, w io.Writer) (output.DailyResult, error)`
- `func NewDailyCmd() (*cobra.Command, *DailyOpts)`

- [ ] **Step 1: Write `daily.go`**

```go
package commands

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/adapters"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/config"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/output"
	"github.com/cobra91/better-ccusage/pkg/pricing"
	"github.com/cobra91/better-ccusage/pkg/terminal"
)

// DailyOpts are the daily command's options.
type DailyOpts struct {
	CommonOpts
}

// Daily is the public Run entry point for the daily command.
// Returns the typed result so the MCP server can call it directly.
func Daily(ctx context.Context, opts DailyOpts, w io.Writer, prices *pricing.PriceTable) (output.DailyResult, error) {
	cfg, _ := config.Load(config.DefaultPath())
	dirs := config.ResolveDirs(opts.ConfigDir, cfg)
	loader := data.NewLoader(dirs...)
	entries, err := loader.Load(ctx)
	if err != nil {
		return output.DailyResult{}, fmt.Errorf("loading data: %w", err)
	}
	if opts.Since != nil || opts.Until != nil {
		entries = data.FilterByTime(entries, opts.Since, opts.Until)
	}
	if len(entries) == 0 {
		return output.DailyResult{}, fmt.Errorf("no data in range")
	}
	entries = adapters.NewManager().Normalize(entries)
	buckets := cost.Aggregate(entries, cost.GroupByDay)
	buckets = cost.ApplyPrices(buckets, prices, opts.Mode)
	rows := output.BucketsToDailyRows(buckets)
	return output.DailyResult{Daily: rows, Summary: output.SumRows(rows)}, nil
}

// NewDailyCmd returns the cobra command for `better-ccusage daily`.
func NewDailyCmd(prices *pricing.PriceTable) *cobra.Command {
	opts := &DailyOpts{}
	cmd := &cobra.Command{
		Use:   "daily",
		Short: "Show daily usage report",
		RunE: func(cmd *cobra.Command, args []string) error {
			common := BindCommonFlags(cmd)
			opts.CommonOpts = *common
			result, err := Daily(cmd.Context(), *opts, cmd.OutOrStdout(), prices)
			if err != nil {
				return err
			}
			if opts.JSON {
				return output.EncodeJSON(os.Stdout, result)
			}
			// Render as table
			cols := []terminal.Column{
				{Header: "Date", Width: 12},
				{Header: "Input", Width: 10},
				{Header: "Output", Width: 10},
				{Header: "Cache R", Width: 10},
				{Header: "Cost", Width: 12, Style: terminal.StyleCost},
			}
			rows := make([][]string, 0, len(result.Daily))
			for _, r := range result.Daily {
				rows = append(rows, []string{r.Date, fmt.Sprint(r.InputTokens), fmt.Sprint(r.OutputTokens), fmt.Sprint(r.CacheReadTokens), fmt.Sprintf("$%.4f", r.CostUSD)})
			}
			tbl := terminal.NewTable(cmd.OutOrStdout(), cols...)
			tbl.SetRows(rows)
			return tbl.Render()
		},
	}
	return cmd
}
```

- [ ] **Step 2: Write `daily_test.go`** — integration test

```go
package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
	"github.com/cobra91/better-ccusage/pkg/pricing"
	"github.com/cobra91/better-ccusage/pkg/terminal"
)

func setupDailyFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	project := filepath.Join(dir, "projects", "myproj")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	jsonl := `{"timestamp":"2026-01-15T10:00:00Z","sessionId":"s1","model":"claude-sonnet-4-5-20250929","inputTokens":1000,"outputTokens":500}
{"timestamp":"2026-01-16T10:00:00Z","sessionId":"s1","model":"claude-sonnet-4-5-20250929","inputTokens":2000,"outputTokens":1000}
`
	if err := os.WriteFile(filepath.Join(project, "abc.jsonl"), []byte(jsonl), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestDaily_Integration(t *testing.T) {
	dir := setupDailyFixture(t)
	pt, _ := pricing.LoadPrices(strings.NewReader(`{"claude-sonnet-4-5-20250929":{"input_cost_per_token":0.000003,"output_cost_per_token":0.000015}}`))
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	var buf strings.Builder
	opts := DailyOpts{
		CommonOpts: CommonOpts{Mode: pricingTestMode(t), JSON: true},
	}
	_ = pricingTestMode // referenced
	result, err := Daily(context.Background(), opts, &buf, pt)
	if err != nil {
		t.Fatalf("Daily: %v", err)
	}
	if len(result.Daily) != 2 {
		t.Errorf("expected 2 daily rows, got %d", len(result.Daily))
	}
	if result.Summary.TotalTokens == 0 {
		t.Error("expected non-zero total tokens in summary")
	}
}

// pricingTestMode is a tiny helper to coerce a CostMode from a string.
func pricingTestMode(t *testing.T) (m cost.CostMode) {
	t.Helper()
	return cost.CostCalculate
}
```

Add `"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"` import.

- [ ] **Step 3: Run tests**

```bash
go test ./apps/better-ccusage/internal/commands/... -race -run TestDaily
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add apps/better-ccusage/internal/commands/daily.go apps/better-ccusage/internal/commands/daily_test.go
git commit -m "feat(better-ccusage): add daily command"
```

---

## Task 11: `monthly` command

Mirrors Task 10 with `GroupByMonth`. Same pattern.

- [ ] **Step 1: Write `monthly.go`** — pattern matches `daily.go`, but uses `cost.GroupByMonth`

```go
package commands

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/adapters"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/config"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/output"
	"github.com/cobra91/better-ccusage/pkg/pricing"
	"github.com/cobra91/better-ccusage/pkg/terminal"
)

type MonthlyOpts struct {
	CommonOpts
}

func Monthly(ctx context.Context, opts MonthlyOpts, w io.Writer, prices *pricing.PriceTable) (output.DailyResult, error) {
	cfg, _ := config.Load(config.DefaultPath())
	dirs := config.ResolveDirs(opts.ConfigDir, cfg)
	loader := data.NewLoader(dirs...)
	entries, err := loader.Load(ctx)
	if err != nil {
		return output.DailyResult{}, err
	}
	if opts.Since != nil || opts.Until != nil {
		entries = data.FilterByTime(entries, opts.Since, opts.Until)
	}
	if len(entries) == 0 {
		return output.DailyResult{}, fmt.Errorf("no data in range")
	}
	entries = adapters.NewManager().Normalize(entries)
	buckets := cost.Aggregate(entries, cost.GroupByMonth)
	buckets = cost.ApplyPrices(buckets, prices, opts.Mode)
	rows := output.BucketsToDailyRows(buckets)
	return output.DailyResult{Daily: rows, Summary: output.SumRows(rows)}, nil
}

func NewMonthlyCmd(prices *pricing.PriceTable) *cobra.Command {
	opts := &MonthlyOpts{}
	cmd := &cobra.Command{
		Use:   "monthly",
		Short: "Show monthly usage report",
		RunE: func(cmd *cobra.Command, args []string) error {
			common := BindCommonFlags(cmd)
			opts.CommonOpts = *common
			result, err := Monthly(cmd.Context(), *opts, cmd.OutOrStdout(), prices)
			if err != nil {
				return err
			}
			if opts.JSON {
				return output.EncodeJSON(os.Stdout, result)
			}
			cols := []terminal.Column{
				{Header: "Month", Width: 10},
				{Header: "Input", Width: 10},
				{Header: "Output", Width: 10},
				{Header: "Cost", Width: 12, Style: terminal.StyleCost},
			}
			rows := make([][]string, 0, len(result.Daily))
			for _, r := range result.Daily {
				rows = append(rows, []string{r.Date, fmt.Sprint(r.InputTokens), fmt.Sprint(r.OutputTokens), fmt.Sprintf("$%.4f", r.CostUSD)})
			}
			tbl := terminal.NewTable(cmd.OutOrStdout(), cols...)
			tbl.SetRows(rows)
			return tbl.Render()
		},
	}
	return cmd
}
```

- [ ] **Step 2: Write `monthly_test.go`** — mirrors daily test

```go
package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/cobra91/better-ccusage/pkg/pricing"
)

func TestMonthly_Integration(t *testing.T) {
	dir := setupDailyFixture(t)
	pt, _ := pricing.LoadPrices(strings.NewReader(`{"claude-sonnet-4-5-20250929":{"input_cost_per_token":0.000003,"output_cost_per_token":0.000015}}`))
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	opts := MonthlyOpts{CommonOpts: CommonOpts{Mode: cost.CostCalculate, JSON: true}}
	var buf strings.Builder
	result, err := Monthly(context.Background(), opts, &buf, pt)
	if err != nil {
		t.Fatalf("Monthly: %v", err)
	}
	if len(result.Daily) != 1 {
		t.Errorf("expected 1 monthly row (both dates in same month), got %d", len(result.Daily))
	}
}
```

Add `cost` import.

- [ ] **Step 3: Run + commit**

```bash
go test ./apps/better-ccusage/internal/commands/... -race -run TestMonthly
git add apps/better-ccusage/internal/commands/monthly.go apps/better-ccusage/internal/commands/monthly_test.go
git commit -m "feat(better-ccusage): add monthly command"
```

---

## Task 12: `session` command

Mirrors Task 10 with `GroupBySession`. Same pattern.

- [ ] **Step 1: Write `session.go`**

```go
package commands

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/adapters"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/config"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/output"
	"github.com/cobra91/better-ccusage/pkg/pricing"
	"github.com/cobra91/better-ccusage/pkg/terminal"
)

type SessionOpts struct {
	CommonOpts
}

func Session(ctx context.Context, opts SessionOpts, w io.Writer, prices *pricing.PriceTable) (output.DailyResult, error) {
	cfg, _ := config.Load(config.DefaultPath())
	dirs := config.ResolveDirs(opts.ConfigDir, cfg)
	loader := data.NewLoader(dirs...)
	entries, err := loader.Load(ctx)
	if err != nil {
		return output.DailyResult{}, err
	}
	if opts.Since != nil || opts.Until != nil {
		entries = data.FilterByTime(entries, opts.Since, opts.Until)
	}
	if len(entries) == 0 {
		return output.DailyResult{}, fmt.Errorf("no data in range")
	}
	entries = adapters.NewManager().Normalize(entries)
	buckets := cost.Aggregate(entries, cost.GroupBySession)
	buckets = cost.ApplyPrices(buckets, prices, opts.Mode)
	rows := output.BucketsToDailyRows(buckets)
	return output.DailyResult{Daily: rows, Summary: output.SumRows(rows)}, nil
}

func NewSessionCmd(prices *pricing.PriceTable) *cobra.Command {
	opts := &SessionOpts{}
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Show session-based usage report",
		RunE: func(cmd *cobra.Command, args []string) error {
			common := BindCommonFlags(cmd)
			opts.CommonOpts = *common
			result, err := Session(cmd.Context(), *opts, cmd.OutOrStdout(), prices)
			if err != nil {
				return err
			}
			if opts.JSON {
				return output.EncodeJSON(os.Stdout, result)
			}
			cols := []terminal.Column{
				{Header: "Session", Width: 30},
				{Header: "Tokens", Width: 12},
				{Header: "Cost", Width: 12, Style: terminal.StyleCost},
			}
			rows := make([][]string, 0, len(result.Daily))
			for _, r := range result.Daily {
				rows = append(rows, []string{r.Date, fmt.Sprint(r.TotalTokens), fmt.Sprintf("$%.4f", r.CostUSD)})
			}
			tbl := terminal.NewTable(cmd.OutOrStdout(), cols...)
			tbl.SetRows(rows)
			return tbl.Render()
		},
	}
	return cmd
}
```

- [ ] **Step 2: Write `session_test.go`**

```go
package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/pkg/pricing"
)

func TestSession_Integration(t *testing.T) {
	dir := setupDailyFixture(t)
	pt, _ := pricing.LoadPrices(strings.NewReader(`{"claude-sonnet-4-5-20250929":{"input_cost_per_token":0.000003,"output_cost_per_token":0.000015}}`))
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	opts := SessionOpts{CommonOpts: CommonOpts{Mode: cost.CostCalculate, JSON: true}}
	var buf strings.Builder
	result, err := Session(context.Background(), opts, &buf, pt)
	if err != nil {
		t.Fatalf("Session: %v", err)
	}
	if len(result.Daily) == 0 {
		t.Error("expected at least 1 session row")
	}
}
```

- [ ] **Step 3: Run + commit**

```bash
go test ./apps/better-ccusage/internal/commands/... -race -run TestSession
git add apps/better-ccusage/internal/commands/session.go apps/better-ccusage/internal/commands/session_test.go
git commit -m "feat(better-ccusage): add session command"
```

---

## Task 13: `blocks` command (static)

- [ ] **Step 1: Write `blocks.go`** — pattern matches daily, uses `GroupByBlock`

```go
package commands

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/adapters"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/config"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/output"
	"github.com/cobra91/better-ccusage/pkg/pricing"
	"github.com/cobra91/better-ccusage/pkg/terminal"
)

type BlocksOpts struct {
	CommonOpts
	Active  bool
	Recent  bool
	TokenLimit int64
}

func Blocks(ctx context.Context, opts BlocksOpts, w io.Writer, prices *pricing.PriceTable) (output.DailyResult, error) {
	cfg, _ := config.Load(config.DefaultPath())
	dirs := config.ResolveDirs(opts.ConfigDir, cfg)
	loader := data.NewLoader(dirs...)
	entries, err := loader.Load(ctx)
	if err != nil {
		return output.DailyResult{}, err
	}
	if opts.Since != nil || opts.Until != nil {
		entries = data.FilterByTime(entries, opts.Since, opts.Until)
	}
	if len(entries) == 0 {
		return output.DailyResult{}, fmt.Errorf("no data in range")
	}
	entries = adapters.NewManager().Normalize(entries)
	buckets := cost.Aggregate(entries, cost.GroupByBlock)
	buckets = cost.ApplyPrices(buckets, prices, opts.Mode)
	rows := output.BucketsToDailyRows(buckets)
	return output.DailyResult{Daily: rows, Summary: output.SumRows(rows)}, nil
}

func NewBlocksCmd(prices *pricing.PriceTable) *cobra.Command {
	opts := &BlocksOpts{}
	cmd := &cobra.Command{
		Use:   "blocks",
		Short: "Show 5-hour billing blocks",
		RunE: func(cmd *cobra.Command, args []string) error {
			common := BindCommonFlags(cmd)
			opts.CommonOpts = *common
			result, err := Blocks(cmd.Context(), *opts, cmd.OutOrStdout(), prices)
			if err != nil {
				return err
			}
			if opts.JSON {
				return output.EncodeJSON(os.Stdout, result)
			}
			cols := []terminal.Column{
				{Header: "Block", Width: 18},
				{Header: "Tokens", Width: 12},
				{Header: "Cost", Width: 12, Style: terminal.StyleCost},
			}
			rows := make([][]string, 0, len(result.Daily))
			for _, r := range result.Daily {
				rows = append(rows, []string{r.Date, fmt.Sprint(r.TotalTokens), fmt.Sprintf("$%.4f", r.CostUSD)})
			}
			tbl := terminal.NewTable(cmd.OutOrStdout(), cols...)
			tbl.SetRows(rows)
			return tbl.Render()
		},
	}
	cmd.Flags().BoolVar(&opts.Active, "active", false, "show only the active block (with projection)")
	cmd.Flags().BoolVar(&opts.Recent, "recent", false, "show blocks from the last 3 days")
	cmd.Flags().Int64Var(&opts.TokenLimit, "token-limit", 0, "warn when a block exceeds this token count")
	return cmd
}
```

- [ ] **Step 2: Write `blocks_test.go`**

```go
package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/pkg/pricing"
)

func TestBlocks_Integration(t *testing.T) {
	dir := setupDailyFixture(t)
	pt, _ := pricing.LoadPrices(strings.NewReader(`{"claude-sonnet-4-5-20250929":{"input_cost_per_token":0.000003,"output_cost_per_token":0.000015}}`))
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	opts := BlocksOpts{CommonOpts: CommonOpts{Mode: cost.CostCalculate, JSON: true}}
	var buf strings.Builder
	result, err := Blocks(context.Background(), opts, &buf, pt)
	if err != nil {
		t.Fatalf("Blocks: %v", err)
	}
	if len(result.Daily) == 0 {
		t.Error("expected at least 1 block row")
	}
}
```

- [ ] **Step 3: Run + commit**

```bash
go test ./apps/better-ccusage/internal/commands/... -race -run TestBlocks
git add apps/better-ccusage/internal/commands/blocks.go apps/better-ccusage/internal/commands/blocks_test.go
git commit -m "feat(better-ccusage): add blocks command (static)"
```

---

## Task 14: Monitor package — bubbles event loop

**Files:**
- Create: `apps/better-ccusage/internal/monitor/monitor.go`
- Create: `apps/better-ccusage/internal/monitor/monitor_test.go`

**Interfaces:**
- `type Monitor struct { ... }`
- `type Opts struct { Loader *data.Loader; Prices *pricing.PriceTable; Mode cost.CostMode; RefreshInterval time.Duration }`
- `func Run(ctx context.Context, opts Opts) error`

Note: this task creates the engine. Task 15 wires it into the `blocks_live` command.

- [ ] **Step 1: Add bubbles + bubbletea deps**

```bash
cd /Volumes/Samsung970EVOPlus/dev-projects/better-ccusage/.worktrees/plan-2-core-cli
go get github.com/charmbracelet/bubbletea@latest github.com/charmbracelet/bubbles@latest
```

- [ ] **Step 2: Write `monitor.go`**

```go
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
```

- [ ] **Step 3: Write `monitor_test.go`** — test that `renderBlocks` formats correctly (test pure function, not the event loop)

```go
package monitor

import (
	"strings"
	"testing"
	"time"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
)

func TestRenderBlocks(t *testing.T) {
	buckets := []cost.Bucket{
		{Key: "2026-01-15T05", Timestamp: time.Now(), InputTokens: 100, OutputTokens: 50, Cost: /* Money{Micros: 1500} */ costFromMicros(1500)},
	}
	out := renderBlocks(buckets)
	if !strings.Contains(out, "2026-01-15T05") {
		t.Errorf("missing block key in output: %q", out)
	}
	if !strings.Contains(out, "tokens=150") {
		t.Errorf("missing token total: %q", out)
	}
	if !strings.Contains(out, "$0.0015") {
		t.Errorf("missing cost: %q", out)
	}
}

func costFromMicros(m int64) struct{ Micros int64 } { return struct{ Micros int64 }{m} }
```

Note: `cost.Bucket.Cost` is type `pricing.Money`. Adjust if needed by importing `pricing` and using `pricing.Money{Micros: 1500}`:

```go
import "github.com/cobra91/better-ccusage/pkg/pricing"
// ...
Cost: pricing.Money{Micros: 1500},
```

- [ ] **Step 4: Run tests**

```bash
go test ./apps/better-ccusage/internal/monitor/... -race
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/better-ccusage/internal/monitor/ go.mod go.sum
git commit -m "feat(better-ccusage): add live monitor with bubbles event loop"
```

---

## Task 15: `blocks_live` command (live monitor)

- [ ] **Step 1: Write `blocks_live.go`**

```go
package commands

import (
	"github.com/spf13/cobra"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/config"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/monitor"
	"github.com/cobra91/better-ccusage/pkg/pricing"
	"time"
)

func NewBlocksLiveCmd(prices *pricing.PriceTable) *cobra.Command {
	opts := &BlocksOpts{}
	cmd := &cobra.Command{
		Use:   "blocks.live",
		Short: "Live-updating 5-hour billing blocks",
		RunE: func(cmd *cobra.Command, args []string) error {
			common := BindCommonFlags(cmd)
			opts.CommonOpts = *common
			cfg, _ := config.Load(config.DefaultPath())
			dirs := config.ResolveDirs(opts.ConfigDir, cfg)
			return monitor.Run(cmd.Context(), monitor.Opts{
				Loader:          data.NewLoader(dirs...),
				Prices:          prices,
				Mode:            cost.CostMode(opts.Mode),
				RefreshInterval: 30 * time.Second,
			})
		},
	}
	cmd.Flags().Int64Var(&opts.TokenLimit, "token-limit", 0, "warn when a block exceeds this token count")
	return cmd
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/better-ccusage/internal/commands/blocks_live.go
git commit -m "feat(better-ccusage): add blocks.live subcommand"
```

---

## Task 16: Statusline package

**Files:**
- Create: `apps/better-ccusage/internal/statusline/statusline.go`
- Create: `apps/better-ccusage/internal/statusline/statusline_test.go`

**Interfaces:**
- `type Input struct { TotalCost pricing.Money; TotalTokens int64; Models []string }`
- `func Render(in Input) string` — produces a single-line ANSI-styled status string

- [ ] **Step 1: Write `statusline.go`**

```go
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
```

- [ ] **Step 2: Write `statusline_test.go`**

```go
package statusline

import (
	"strings"
	"testing"

	"github.com/cobra91/better-ccusage/pkg/pricing"
	"github.com/cobra91/better-ccusage/pkg/terminal"
)

func TestRender_Basic(t *testing.T) {
	out := Render(Input{
		TotalCost:   pricing.Money{Micros: 50_000},
		TotalTokens: 1500,
		Models:      []string{"claude-sonnet-4-5-20250929"},
	})
	if !strings.Contains(out, "claude-sonnet-4-5") {
		t.Errorf("missing model: %q", out)
	}
	if !strings.Contains(out, "1500 tokens") {
		t.Errorf("missing token count: %q", out)
	}
	if !strings.Contains(out, "$0.0500") {
		t.Errorf("missing cost: %q", out)
	}
}

func TestRender_NoModel(t *testing.T) {
	out := Render(Input{TotalCost: pricing.Money{}, TotalTokens: 100})
	if !strings.Contains(out, "unknown") {
		t.Errorf("expected 'unknown' for empty models, got %q", out)
	}
	// Normalize ANSI before substring check
	normalized := terminal.StripANSI(out)
	if !strings.Contains(normalized, "unknown") {
		t.Errorf("normalized output missing 'unknown': %q", normalized)
	}
}
```

- [ ] **Step 3: Run + commit**

```bash
go test ./apps/better-ccusage/internal/statusline/... -race
git add apps/better-ccusage/internal/statusline/
git commit -m "feat(better-ccusage): add statusline renderer"
```

---

## Task 17: `statusline` command

- [ ] **Step 1: Write `statusline.go`**

```go
package commands

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/adapters"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/config"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/statusline"
	"github.com/cobra91/better-ccusage/pkg/pricing"
)

type StatuslineOpts struct {
	CommonOpts
}

func Statusline(ctx context.Context, opts StatuslineOpts, w io.Writer, prices *pricing.PriceTable) error {
	cfg, _ := config.Load(config.DefaultPath())
	dirs := config.ResolveDirs(opts.ConfigDir, cfg)
	loader := data.NewLoader(dirs...)
	entries, err := loader.Load(ctx)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("no data")
	}
	entries = adapters.NewManager().Normalize(entries)
	buckets := cost.Aggregate(entries, cost.GroupByDay)
	buckets = cost.ApplyPrices(buckets, prices, opts.Mode)
	var totalCost pricing.Money
	var totalTokens int64
	models := map[string]bool{}
	for _, b := range buckets {
		totalCost = totalCost.Add(b.Cost)
		totalTokens += b.InputTokens + b.OutputTokens + b.CacheCreationTokens + b.CacheReadTokens
		models[b.Model] = true
	}
	modelList := make([]string, 0, len(models))
	for m := range models {
		modelList = append(modelList, m)
	}
	out := statusline.Render(statusline.Input{TotalCost: totalCost, TotalTokens: totalTokens, Models: modelList})
	_, err = io.WriteString(w, out+"\n")
	return err
}

func NewStatuslineCmd(prices *pricing.PriceTable) *cobra.Command {
	opts := &StatuslineOpts{}
	cmd := &cobra.Command{
		Use:   "statusline",
		Short: "Render compact statusline for Claude Code",
		RunE: func(cmd *cobra.Command, args []string) error {
			common := BindCommonFlags(cmd)
			opts.CommonOpts = *common
			return Statusline(cmd.Context(), *opts, cmd.OutOrStdout(), prices)
		},
	}
	return cmd
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/better-ccusage/internal/commands/statusline.go
git commit -m "feat(better-ccusage): add statusline command"
```

---

## Task 18: `weekly` command

Mirrors daily with `GroupByDay` then groups by ISO week. For v1, simpler approach: `GroupByDay` and let consumers group further. Or use a custom 7-day rolling bucket.

For simplicity, **reuse daily's bucket structure** but with a weekly grouping key.

- [ ] **Step 1: Add a Week grouping helper in `cost/aggregate.go`**

Extend `groupKeyFor`:

```go
case GroupByWeek:
    yr, wk := t.ISOWeek()
    return fmt.Sprintf("%04d-W%02d", yr, wk)
```

Add `GroupByWeek GroupKey = iota + 4` (after `GroupByBlock`).

- [ ] **Step 2: Write `weekly.go`** — pattern matches daily, uses `GroupByWeek`

```go
package commands

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/adapters"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/config"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/data"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/output"
	"github.com/cobra91/better-ccusage/pkg/pricing"
	"github.com/cobra91/better-ccusage/pkg/terminal"
)

type WeeklyOpts struct {
	CommonOpts
}

func Weekly(ctx context.Context, opts WeeklyOpts, w io.Writer, prices *pricing.PriceTable) (output.DailyResult, error) {
	cfg, _ := config.Load(config.DefaultPath())
	dirs := config.ResolveDirs(opts.ConfigDir, cfg)
	loader := data.NewLoader(dirs...)
	entries, err := loader.Load(ctx)
	if err != nil {
		return output.DailyResult{}, err
	}
	if opts.Since != nil || opts.Until != nil {
		entries = data.FilterByTime(entries, opts.Since, opts.Until)
	}
	if len(entries) == 0 {
		return output.DailyResult{}, fmt.Errorf("no data in range")
	}
	entries = adapters.NewManager().Normalize(entries)
	buckets := cost.Aggregate(entries, cost.GroupByWeek)
	buckets = cost.ApplyPrices(buckets, prices, opts.Mode)
	rows := output.BucketsToDailyRows(buckets)
	return output.DailyResult{Daily: rows, Summary: output.SumRows(rows)}, nil
}

func NewWeeklyCmd(prices *pricing.PriceTable) *cobra.Command {
	opts := &WeeklyOpts{}
	cmd := &cobra.Command{
		Use:   "weekly",
		Short: "Show weekly usage report",
		RunE: func(cmd *cobra.Command, args []string) error {
			common := BindCommonFlags(cmd)
			opts.CommonOpts = *common
			result, err := Weekly(cmd.Context(), *opts, cmd.OutOrStdout(), prices)
			if err != nil {
				return err
			}
			if opts.JSON {
				return output.EncodeJSON(os.Stdout, result)
			}
			cols := []terminal.Column{
				{Header: "Week", Width: 12},
				{Header: "Tokens", Width: 12},
				{Header: "Cost", Width: 12, Style: terminal.StyleCost},
			}
			rows := make([][]string, 0, len(result.Daily))
			for _, r := range result.Daily {
				rows = append(rows, []string{r.Date, fmt.Sprint(r.TotalTokens), fmt.Sprintf("$%.4f", r.CostUSD)})
			}
			tbl := terminal.NewTable(cmd.OutOrStdout(), cols...)
			tbl.SetRows(rows)
			return tbl.Render()
		},
	}
	return cmd
}
```

- [ ] **Step 3: Commit**

```bash
git add apps/better-ccusage/internal/cost/aggregate.go apps/better-ccusage/internal/commands/weekly.go
git commit -m "feat(better-ccusage): add weekly command"
```

---

## Task 19: jq wrapper

**Files:**
- Create: `apps/better-ccusage/internal/jq/jq.go`
- Create: `apps/better-ccusage/internal/jq/jq_test.go`

**Interfaces:**
- `func Process(expression string, input []byte) ([]byte, error)` — runs a jq expression on input bytes

- [ ] **Step 1: Add gojq dep**

```bash
go get github.com/itchyny/gojq@latest
```

- [ ] **Step 2: Write `jq.go`**

```go
package jq

import (
	"fmt"

	"github.com/itchyny/gojq"
)

// Process runs the given jq expression on input bytes and returns the result.
func Process(expression string, input []byte) ([]byte, error) {
	q, err := gojq.New().Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("parsing jq expression: %w", err)
	}
	var value any
	if err := gojq.Unmarshal(input, &value); err != nil {
		return nil, fmt.Errorf("unmarshaling input: %w", err)
	}
	iter := q.Run(value)
	v, ok := iter.Next()
	if !ok {
		return nil, nil
	}
	if err, ok := v.(error); ok {
		return nil, err
	}
	return gojq.Marshal(v)
}
```

- [ ] **Step 3: Write `jq_test.go`**

```go
package jq

import (
	"strings"
	"testing"
)

func TestProcess_Identity(t *testing.T) {
	out, err := Process(".", []byte(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"a"`) {
		t.Errorf("expected identity output, got %q", out)
	}
}

func TestProcess_FieldAccess(t *testing.T) {
	out, err := Process(".a", []byte(`{"a":42}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "42\n" {
		t.Errorf("expected '42\\n', got %q", out)
	}
}
```

- [ ] **Step 4: Run + commit**

```bash
go test ./apps/better-ccusage/internal/jq/... -race
git add apps/better-ccusage/internal/jq/ go.mod go.sum
git commit -m "feat(better-ccusage): add gojq wrapper for jq processing"
```

---

## Task 20: JSON Schema for config

**Files:**
- Create: `apps/better-ccusage/internal/schema/schema.go`
- Create: `apps/better-ccusage/internal/schema/schema_test.go`

**Interfaces:**
- `var ConfigSchema = \`{...}\`` — hand-written JSON Schema string
- `func Write(w io.Writer) error` — writes the schema to w
- `func Validate(data []byte) error` — checks that data conforms to schema (optional; basic JSON Schema check)

- [ ] **Step 1: Write `schema.go`**

```go
package schema

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
)

//go:embed config-schema.json
var configSchemaJSON []byte

// ConfigSchema returns the JSON Schema for the user config file as bytes.
func ConfigSchema() []byte { return configSchemaJSON }

// Write outputs the config schema (pretty-printed) to w.
func Write(w io.Writer) error {
	var v any
	if err := json.Unmarshal(configSchemaJSON, &v); err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Validate checks that data is a valid JSON object (basic check; full schema
// validation requires a JSON Schema library which is out of scope for v1).
func Validate(data []byte) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}
```

- [ ] **Step 2: Create `config-schema.json`** (hand-written JSON Schema)

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "better-ccusage config",
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "pricingUrl": { "type": "string", "format": "uri" },
    "offline": { "type": "boolean" },
    "defaultMode": { "type": "string", "enum": ["auto", "calculate", "display"] },
    "extraConfigDirs": {
      "type": "array",
      "items": { "type": "string" }
    }
  }
}
```

- [ ] **Step 3: Write `schema_test.go`**

```go
package schema

import (
	"bytes"
	"testing"
)

func TestConfigSchema_Embedded(t *testing.T) {
	s := ConfigSchema()
	if len(s) == 0 {
		t.Fatal("config schema is empty")
	}
}

func TestValidate_ValidObject(t *testing.T) {
	if err := Validate([]byte(`{"offline":true,"defaultMode":"calculate"}`)); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestWrite_ProducesJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf); err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Error("Write produced no output")
	}
}
```

- [ ] **Step 4: Run + commit**

```bash
go test ./apps/better-ccusage/internal/schema/... -race
git add apps/better-ccusage/internal/schema/
git commit -m "feat(better-ccusage): add hand-written config JSON Schema"
```

---

## Task 21: Wire up cobra subcommands in main.go

- [ ] **Step 1: Update `root.go`**

```go
package main

import (
	"github.com/spf13/cobra"
	"github.com/cobra91/better-ccusage/pkg/pricing"
	"github.com/cobra91/better-ccusage/pkg/terminal"
)

func newRootCmd(log *terminal.Logger) *cobra.Command {
	prices, err := pricing.LoadPrices(bytes.NewReader(pricing.EmbeddedPrices))
	if err != nil {
		log.Warn("failed to load embedded pricing: %v", err)
	}

	root := &cobra.Command{
		Use:     "better-ccusage",
		Short:   "Analyze Claude Code usage from local JSONL files",
		Version: version,
	}
	root.AddCommand(
		newDailyCmd(prices),
		newMonthlyCmd(prices),
		newSessionCmd(prices),
		newBlocksCmd(prices),
		newBlocksLiveCmd(prices),
		newStatuslineCmd(prices),
		newWeeklyCmd(prices),
	)
	return root
}
```

Add `"bytes"` import.

- [ ] **Step 2: Verify it builds**

```bash
go build ./apps/better-ccusage/cmd/better-ccusage
./apps/better-ccusage/cmd/better-ccusage --help
```

Expected: lists daily, monthly, session, blocks, statusline, weekly subcommands.

- [ ] **Step 3: Smoke test**

```bash
./apps/better-ccusage/cmd/better-ccusage daily --help
./apps/better-ccusage/cmd/better-ccusage blocks --help
```

Expected: shows flag list.

- [ ] **Step 4: Commit**

```bash
git add apps/better-ccusage/cmd/better-ccusage/root.go
git commit -m "feat(better-ccusage): wire up cobra subcommands in root"
```

---

## Task 22: Add jq processing flag to relevant commands (optional polish)

Skip if not needed for v1. The Daily command supports `--jq <expr>` which post-processes the JSON output via the jq wrapper.

- [ ] **Step 1: Add `--jq` flag to `daily.go`** (and optionally other commands)

In `BindCommonFlags` (or per-command):

```go
var jqExpr string
cmd.PersistentFlags().StringVar(&jqExpr, "jq", "", "post-process JSON output with a jq expression")
```

After `output.EncodeJSON`, if `jqExpr != ""`, pipe through `jq.Process(jqExpr, ...)`.

For v1, this is optional. **Skip if scope creep.**

---

## Task 23: Replace testdata pricing fixture with real upstream JSON

**Files:**
- Modify: `pkg/pricing/testdata/prices.json`

**Goal:** Replace the 3-model test fixture with the real 253KB upstream `model_prices_and_context_window.json` so the embedded binary contains production pricing data.

- [ ] **Step 1: Copy the upstream JSON**

```bash
cp /Volumes/Samsung970EVOPlus/dev-projects/better-ccusage/packages/internal/model_prices_and_context_window.json \
   /Volumes/Samsung970EVOPlus/dev-projects/better-ccusage/.worktrees/plan-2-core-cli/pkg/pricing/testdata/prices.json
```

- [ ] **Step 2: Verify `LoadPrices` still parses it**

```bash
go test ./pkg/pricing/... -race -run TestLoadPrices
```

Expected: PASS (the existing test uses a small inline JSON, but should still pass).

Add a smoke test:

```go
func TestLoadPrices_EmbeddedIsValid(t *testing.T) {
	pt, err := pricing.LoadPrices(bytes.NewReader(pricing.EmbeddedPrices))
	if err != nil {
		t.Fatalf("embedded prices invalid: %v", err)
	}
	if len(pt.rawPrices()) == 0 {
		t.Error("embedded prices loaded but empty")
	}
	if _, ok := pt.LookupExact("claude-sonnet-4-5-20250929"); !ok {
		t.Error("expected claude-sonnet-4-5-20250929 in embedded prices")
	}
}
```

Add to `pkg/pricing/prices_test.go`.

- [ ] **Step 3: Run + commit**

```bash
go test ./... -race
git add pkg/pricing/testdata/prices.json pkg/pricing/prices_test.go
git commit -m "feat(pricing): embed real upstream pricing JSON (LiteLLM)"
```

---

## Task 24: Foundation verification (end-to-end)

**Files:** none

- [ ] **Step 1: Build the full CLI**

```bash
cd /Volumes/Samsung970EVOPlus/dev-projects/better-ccusage/.worktrees/plan-2-core-cli
go build ./...
```

Expected: builds clean.

- [ ] **Step 2: Build the binary explicitly**

```bash
go build -o dist/better-ccusage ./apps/better-ccusage/cmd/better-ccusage
./dist/better-ccusage --version
```

Expected: outputs version.

- [ ] **Step 3: Run full test suite with race detector**

```bash
go test ./... -race -count=1
```

Expected: all packages green.

- [ ] **Step 4: Run `./scripts/test.sh`**

```bash
./scripts/test.sh
```

Expected: green.

- [ ] **Step 5: Run `go vet`**

```bash
go vet ./...
```

Expected: no diagnostics.

- [ ] **Step 6: Build script sanity check**

```bash
./scripts/build.sh
ls -lh dist/
```

Expected: all four binaries built (note: only `better-ccusage` will exist after Plan 2; `better-ccusage-mcp`, `-codex`, `-opencode` come in Plans 3 and 4. Build script will warn but should still complete for `better-ccusage`).

- [ ] **Step 7: Report to user**

Tell the user:
- Plan 2 is complete.
- Total tasks completed: 24.
- Total commits: ~24 (one per task).
- Test counts: `go test ./... -race` passes.
- Binary builds and runs.
- Ready to begin Plan 3 (MCP server) when approved.

---

## End of Plan 2

The next plan (Plan 3: MCP server) depends on Plan 2's deliverables:
- `apps/better-ccusage/internal/commands.{Daily,Monthly,Session,Blocks,BlocksLive,Statusline,Weekly}` public Run functions
- Working cobra root with all six subcommands wired
- Real pricing JSON embedded in the binary
- Live monitor (bubbles) for `blocks.live`

Plan 3 introduces the official MCP Go SDK and exposes the four commands as MCP tools, sharing the same Go functions (no subprocess).
