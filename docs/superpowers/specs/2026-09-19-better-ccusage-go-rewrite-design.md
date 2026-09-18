# better-ccusage Go Rewrite — Design

**Date:** 2026-09-19
**Status:** Approved (brainstorming complete)
**Author:** brainstorming session

## Summary

Rewrite `better-ccusage` (and its sibling apps + shared packages) from TypeScript to Go. The rewrite is a strict 1:1 behavior port — every TS feature, command, provider adapter, and output format is replicated exactly. User-visible CLI surface, JSON output shape, MCP tool responses, statusline format, and pricing logic must remain byte-equivalent (modulo ANSI escape sequence details) to the TS version. Migration strategy (how to cut over from npm-published TS packages to Go binaries) is **deferred** until the Go rewrite is feature-complete and validated against real-world data.

## Goals

1. Replace the TS implementation with Go across the full monorepo scope.
2. Preserve every user-visible behavior of the TS implementation exactly.
3. Use idiomatic Go patterns (stdlib errors, `internal/` packages, `_test.go` files, stdlib `testing`).
4. Ship native binaries that install via the same npm package names as the TS packages (the package contents become Go binaries).
5. Lay groundwork for a future migration cutover, but defer the cutover decision.

## Non-Goals

- Migration / cutover strategy (npm publishing, Homebrew tap, Windows MSI, etc.) — deferred until post-validation.
- Linux and Windows CI / binary builds — deferred; v1 ships macOS only.
- New CLI commands, new flags, new provider adapters, new output formats.
- Performance optimization beyond "use Go stdlib idiomatically."
- Rewriting the VitePress documentation site.
- Replacing the pricing JSON source (still bundled + optionally fetched from the same upstream URL as TS).

## Decisions (from brainstorming)

| Decision | Choice |
|---|---|
| Scope | Full monorepo mirror (all 4 apps + both packages) |
| Distribution shape | Separate binaries (mirror TS package names) |
| Migration strategy | Spike validation first (defer cutover decision) |
| Feature parity | Strict 1:1 port |
| CLI framework | cobra + pflag |
| Terminal rendering | lipgloss + bubbles/table |
| MCP server | Official MCP Go SDK |
| Pricing data | embed.FS + live fetcher |
| Platform matrix | macOS first, add Linux/Windows later |
| Architecture | Single Go module at repo root, replace TS in place |

## Repository Layout

```
better-ccusage/
├── go.mod                              # single Go module, Go 1.22+
├── go.sum
├── pkg/
│   ├── pricing/                        # was packages/internal
│   │   ├── prices.go                   # embedded model_prices_and_context_window.json
│   │   ├── fetcher.go                  # live fetch + atomic swap
│   │   ├── match.go                    # exact → provider-prefix → fuzzy scorer
│   │   └── prices_test.go
│   └── terminal/                       # was packages/terminal
│       ├── table.go                    # lipgloss-based table renderer
│       ├── colors.go                   # color/style helpers matching consola
│       ├── logger.go                   # leveled logger matching LOG_LEVEL semantics
│       ├── monitor.go                  # live monitor (Charm bubbles)
│       └── *_test.go
├── apps/
│   ├── better-ccusage/
│   │   ├── cmd/better-ccusage/main.go
│   │   └── internal/
│   │       ├── data/                   # JSONL loader (was data-loader.ts)
│   │       ├── cost/                   # token aggregation + cost calc (was calculate-cost.ts)
│   │       ├── adapters/               # 7 provider adapters
│   │       │   ├── claude.go           # base adapter
│   │       │   ├── codex.go
│   │       │   ├── opencode.go
│   │       │   ├── devin.go
│   │       │   ├── pi.go
│   │       │   ├── zcode.go
│   │       │   └── droid.go
│   │       ├── commands/               # daily, monthly, session, blocks, statusline, weekly
│   │       │   ├── common.go           # shared flags
│   │       │   ├── daily.go
│   │       │   ├── monthly.go
│   │       │   ├── session.go
│   │       │   ├── blocks.go
│   │       │   ├── blocks_live.go      # live monitor command
│   │       │   ├── statusline.go
│   │       │   └── weekly.go
│   │       ├── config/                 # JSON config loading + multi-config-dir
│   │       ├── monitor/                # live monitor engine
│   │       ├── jq/                     # gojq wrapper
│   │       ├── statusline/             # statusline loader/renderer
│   │       ├── schema/                 # JSON Schema generation
│   │       ├── output/                 # JSON output types + serialization
│   │       ├── errs/                   # sentinel errors
│   │       └── money/                  # int64-based Money type
│   ├── mcp/
│   │   ├── cmd/better-ccusage-mcp/main.go
│   │   └── internal/
│   │       ├── server/                 # MCP server using official SDK
│   │       ├── transport/              # stdio + StreamableHTTP
│   │       └── tools/                  # daily/session/monthly/blocks tools
│   ├── codex/                          # deprecated shim
│   │   └── cmd/better-ccusage-codex/main.go
│   └── opencode/                       # deprecated shim
│       └── cmd/better-ccusage-opencode/main.go
├── docs/                               # UNCHANGED — stays VitePress
├── scripts/
│   ├── build.sh                        # builds all 4 binaries for current platform
│   ├── test.sh                         # go test ./... -race
│   ├── fmt.sh                          # gofumpt + goimports
│   └── lint.sh                         # golangci-lint run
├── .github/workflows/ci.yml            # macOS-only for v1
└── README.md                           # updated to reference Go binaries
```

**Module strategy:** Single `go.mod` at repo root, import path `github.com/cobra91/better-ccusage`. Apps use `internal/` to keep implementation private; shared code lives in `pkg/` and is exported.

**Deprecated shims:** `apps/codex` and `apps/opencode` each become a single ~30-line `main.go` that calls `os.Exec("better-ccusage", os.Args[1:]...)` with the right subcommand prefix, preserving the user-visible CLI surface.

## Components and Interfaces

### `pkg/pricing`

Public API:

```go
type Price struct {
    InputCostPerToken  float64
    OutputCostPerToken float64
    CacheReadCost      float64
    CacheWriteCost     float64
    // ...other fields from pricing JSON as needed
}

type PriceTable struct { /* unexported */ }

func Load(ctx context.Context, r io.Reader) (*PriceTable, error)
func (t *PriceTable) Lookup(model string) (Price, float64)   // (price, confidence 0.0-1.0)
func (t *PriceTable) Fetch(ctx context.Context, url string, c *http.Client) error
```

- `Load` parses the embedded JSON once at startup.
- `Lookup` runs the three-tier resolution: exact match → provider-prefix suffix match → fuzzy scorer.
- `Fetch` retrieves a newer pricing JSON, validates against the schema, and atomically swaps the in-memory table.
- Confidence < 0.6 returns `(Price{}, 0.0)` so callers can decide whether to skip or warn.
- No I/O at import time — all filesystem and network calls happen behind the public methods.

### `pkg/terminal`

Public API:

```go
type Column struct {
    Header string
    Width  int
    Align  int    // left/center/right
    Color  string // lipgloss style name
}

type Table struct{ /* unexported */ }
func NewTable(w io.Writer, cols ...Column) *Table
func (t *Table) SetRows(rows [][]string) *Table
func (t *Table) Render() error

type Logger struct{ /* unexported */ }
func NewLogger(level int) *Logger
func (l *Logger) Warn(msg string, args ...any)
func (l *Logger) Info(msg string, args ...any)
// ...Debug, Trace, Fatal

func LiveMonitor(ctx context.Context, opts LiveOpts) error   // Charm bubbles event loop
```

- Table output matches `cli-table3` formatting (Unicode borders, column alignment, color via lipgloss).
- Logger writes to `os.Stderr` so JSON stdout output stays clean.
- `LOG_LEVEL` env var: `0`=silent, `1`=warn, `2`=log, `3`=info, `4`=debug, `5`=trace.
- `LiveMonitor` is the Charm bubbles program for the live blocks monitor (replaces `_live-monitor.ts` + `_blocks.live.ts`).

### `apps/better-ccusage/internal/`

**`data/` — JSONL loader**

```go
type Loader struct{ /* unexported */ }
func NewLoader(dirs ...string) *Loader
func (l *Loader) Load(ctx context.Context) ([]Entry, error)
func (l *Loader) Reload(ctx context.Context, prev []Entry) ([]Entry, error)   // incremental, mtime-aware
```

- Walks `{dir}/projects/*/*.jsonl` for each configured dir.
- Silently skips malformed JSONL lines (matches TS behavior).
- Supports `CLAUDE_CONFIG_DIR` env var accepting comma-separated paths (matches TS multi-dir support).
- Default dirs: `~/.config/claude/projects/` + `~/.claude/projects/`.

**`cost/` — token aggregation + cost calculation**

```go
type GroupKey int
const (
    GroupByDay GroupKey = iota
    GroupByMonth
    GroupBySession
    GroupByBlock
)

type CostMode int
const (
    CostAuto CostMode = iota
    CostCalculate
    CostDisplay
)

func Aggregate(entries []data.Entry, key GroupKey) []Bucket
func ApplyPrices(buckets []Bucket, prices *pricing.PriceTable, mode CostMode, opt costOpts) []Bucket
```

- Pure functions, no I/O.
- Bucket ordering matches TS: sort by date ascending within group, then by model name.

**`adapters/` — provider normalization**

```go
type Provider interface {
    Name() string
    Detect(entry data.Entry) bool
    Adapt(entry data.Entry) (data.Entry, bool)   // (normalized, ok)
}

type Manager struct{ /* unexported */ }
func NewManager() *Manager
func (m *Manager) Normalize(entries []data.Entry) []data.Entry
```

- One file per provider: `claude.go` (base), `codex.go`, `opencode.go`, `devin.go`, `pi.go`, `zcode.go`, `droid.go`.
- The fuzzy model matching logic from TS `_pricing-fetcher.ts` is shared via `pkg/pricing/match.go`.

**`commands/` — cobra subcommands**

Each command is a struct implementing `cobra.Command` with `Use`, `Short`, `RunE`. Shared flags (`--json`, `--mode`, `--offline`, `--config-dir`, `--since`, `--until`) defined once in `common.go` and attached via `PersistentFlags`.

```go
type DailyOpts struct {
    Mode    cost.CostMode
    Offline bool
    JSON    bool
    Since   *time.Time
    Until   *time.Time
    // ...
}

func Daily(ctx context.Context, opts DailyOpts) (DailyResult, error)
```

- Each command exposes a public `Run(ctx, opts) (Result, error)` API so the MCP server can call directly (no subprocess).

**`config/` — JSON config loading**

```go
type Config struct {
    PricingURL     string   `json:"pricingUrl,omitempty"`
    Offline        bool     `json:"offline,omitempty"`
    DefaultMode    string   `json:"defaultMode,omitempty"`
    ExtraConfigDirs []string `json:"extraConfigDirs,omitempty"`
}

func Load(path string) (*Config, error)
func ResolveDirs(envValue string) []string   // merges env with config defaults
```

- Loads `~/.config/better-ccusage/config.json` if present.
- Validates against an embedded JSON Schema.
- Multi-config-dir resolution happens here, so commands just call `config.ResolveDirs()`.

**`monitor/`, `jq/`, `statusline/`, `schema/`, `output/`, `errs/`, `money/`**

Narrow packages, each porting one TS file's behavior:
- `monitor/` — Charm bubbles event loop for the live blocks monitor.
- `jq/` — thin wrapper over `github.com/itchyny/gojq` (pinned version).
- `statusline/` — loader/renderer for the compact statusline.
- `schema/` — generates `config-schema.json` at build time. **Hand-written** JSON Schema string in a Go file (small, stable schema; no reflection dep needed).
- `output/` — typed structs + JSON serialization matching TS `_json-output-types.ts` exactly.
- `errs/` — sentinel errors (`ErrNoData`, `ErrUnknownModel`, `ErrInvalidJSON`, `ErrConfigNotFound`, `ErrIncompatibleMode`).
- `money/` — `Money` type as `struct{ Micros int64 }` with custom JSON marshal/unmarshal that handles the string format from the pricing JSON. No floats in cost paths.

### `apps/mcp/internal/`

**`server/`**

- Uses `github.com/modelcontextprotocol/go-sdk` (official MCP Go SDK).
- Registers 4 tools: `daily`, `session`, `monthly`, `blocks`.
- Each tool's handler is ~50 lines: parse MCP args → call `commands.X.Run(ctx, opts)` → return JSON response.

**`transport/`**

- stdio transport (default) and StreamableHTTP transport (selected via `--type http --port N`).
- HTTP transport uses stdlib `net/http` for routing.

**`tools/`**

- One file per tool. Each defines the MCP tool schema and delegates to the corresponding `commands.X.Run` function via direct Go call (no subprocess).

### Deprecated shims

- `apps/codex/cmd/better-ccusage-codex/main.go` — ~30 lines, `os.Exec("better-ccusage", append([]string{"codex"}, os.Args[1:]...)...)`.
- `apps/opencode/cmd/better-ccusage-opencode/main.go` — same shape with `"opencode"` prefix.
- Both print the TS shim's deprecation notice if the upstream `better-ccusage` binary cannot be found.

## Data Flow

### Per-command CLI flow

```
cobra root parses args
    └─> config.ResolveDirs(env CLAUDE_CONFIG_DIR, defaults)
    └─> commands.X.Run(ctx, XOpts{...})
            └─> pricing.Load(ctx, embedReader)         // parse embedded JSON
            └─> if !offline && mode==Calculate:
                    └─> prices.Fetch(ctx, upstreamURL) // 5s timeout, atomic swap
            └─> data.NewLoader(dirs...).Load(ctx)
                    └─> walks {dir}/projects/*/*.jsonl
                    └─> json.Unmarshal each line → Entry; skip malformed
            └─> adapters.Manager.Normalize(entries)
            └─> cost.Aggregate(entries, groupKey)
            └─> cost.ApplyPrices(buckets, prices, mode)
            └─> if opts.JSON: output.EncodeJSON(os.Stdout, result)
                else:        terminal.NewTable(...).SetRows(...).Render()
```

**Invariants:**
- `cost.Aggregate` is pure — same input, same buckets.
- Pricing loads once per process; live fetch happens once at startup if conditions met.
- Aggregation order matches TS implementation exactly.

### MCP tool flow

```
MCP client → JSON-RPC (stdio or HTTP)
    └─> server handler resolves tool name
    └─> tools/X.Handle(ctx, req)
            └─> commands.X.Run(ctx, parsedOpts)        // direct Go call
            └─> return {content: [{type: "text", text: JSON}]}
```

- Errors propagated as MCP `isError: true` with structured payload `{code, message, hint}`.
- Error code mapping in `transport/errors.go`:
  - `ErrNoData` → `NO_DATA`
  - `ErrUnknownModel` → `UNKNOWN_MODEL`
  - default → `INTERNAL`

### Live monitor

```
cobra command starts blocks.live
    └─> tea.NewProgram(monitor.New(loader, prices, opts))
            └─> Init(): synchronous first load + render
            └─> Update loop (bubbles):
                    ├─> TickMsg (every refresh-interval): loader.Reload + re-aggregate + re-render
                    ├─> WindowSizeMsg: re-layout columns
                    └─> KeyMsg (q/Ctrl-C): quit
```

- Reloads are incremental: only re-parse files whose mtime changed since last load.

### Pricing loader

```
init (once at startup):
    └─> embed.FS.ReadFile("model_prices_and_context_window.json")
    └─> json.Unmarshal → PriceTable (atomic.Value.Store)
    └─> if !offline && has-network:
            └─> pricing.Fetch(ctx, upstreamURL, httpClient)
                    └─> 5s timeout
                    └─> validate response schema
                    └─> atomic.Value.Store(newTable) on success
                    └─> log warning + keep old table on error

per-lookup:
    └─> PriceTable.Lookup(model):
            1. exact match
            2. provider-prefix match (split "/" → try suffix)
            3. fuzzy scorer (top score > 0.6)
            return (Price, confidence)
```

- Atomic swap means partial fetches never corrupt in-memory state.

## Error Handling

### Philosophy

Standard Go `(value, error)`, no Result type. Strict 1:1 behavior, not syntax — the TS code uses `@praha/byethrow` Result, but Go readers expect stdlib errors.

### Wrapping

Every non-trivial return wraps with context:
```go
return nil, fmt.Errorf("loading %s: %w", path, err)
```
Callers branch on sentinels via `errors.Is(err, errs.ErrNoData)` without losing the chain.

### Sentinel errors

In `apps/better-ccusage/internal/errs/`:
```go
var (
    ErrNoData           = errors.New("no usage data found")
    ErrUnknownModel     = errors.New("unknown model")
    ErrInvalidJSON      = errors.New("invalid JSONL entry")
    ErrConfigNotFound   = errors.New("config file not found")
    ErrIncompatibleMode = errors.New("cost mode incompatible with data")
)
```

### Logging

`pkg/terminal/logger.go` exposes `Warn`, `Info`, `Debug`, `Trace` (plus untraced `Fatal`). Writes to `os.Stderr`. `LOG_LEVEL` env var mirrors TS: `0`=silent, `1`=warn, `2`=log, `3`=info, `4`=debug, `5`=trace.

### User-visible CLI errors

`pkg/terminal.FriendlyError(err)` renders:
```
Error: <root message>
  hint: <suggestion if available>
```
in consola-style red. For JSON mode: `{"error": "...", "code": "..."}` to stdout, exit 1.

### MCP error mapping

In `apps/mcp/internal/transport/errors.go`:
```go
func MapError(err error) *mcp.Error {
    switch {
    case errors.Is(err, errs.ErrNoData):       return mcpErr("NO_DATA", err)
    case errors.Is(err, errs.ErrUnknownModel): return mcpErr("UNKNOWN_MODEL", err)
    default:                                    return mcpErr("INTERNAL", err)
    }
}
```

## Testing Strategy

### Idiomatic Go

- Stdlib `testing` package only — no testify, no ginkgo, no gomock.
- `*_test.go` files alongside code (the TS in-source pattern doesn't translate to Go).
- Table-driven tests for every multi-case function.
- `t.TempDir()` for filesystem isolation.

### Fixtures

A small `internal/testfixtures` package builds temp dir trees on `t.TempDir()` and writes JSONL from inline byte literals. Each fixture is named and lives next to its test file. No external fixture-loading library.

### Coverage targets

- Unit: 80%+ on `pkg/pricing`, `pkg/terminal`, `cost/`, `data/`.
- Integration: one full-pipeline test per command (`daily`, `monthly`, `session`, `blocks`, `statusline`, `weekly`) with `bytes.Equal` against golden output files committed to `apps/better-ccusage/testdata/`.
- MCP: one test per tool covering success path + at least one error path.
- Live monitor: snapshot test using Charm bubbles' `tea.WithInput`/`tea.WithOutput` test harnesses.

### Table output snapshotting

Golden files are brittle to terminal width and color codes. Helper `pkg/terminal.NormalizeForTest(s)` strips ANSI escapes and width-pads columns before comparison.

### Race detector

`scripts/test.sh` runs `go test ./... -race` — required, no exceptions.

## Cross-Cutting Concerns

### Configuration file

JSON at `~/.config/better-ccusage/config.json`. Parsed via `encoding/json`, validated against an embedded hand-written JSON Schema. The `better-ccusage.example.json` reference file stays at repo root.

### Build & release scripts

```bash
# scripts/build.sh
go build -o dist/better-ccusage ./apps/better-ccusage/cmd/better-ccusage
go build -o dist/better-ccusage-mcp ./apps/mcp/cmd/better-ccusage-mcp
go build -o dist/better-ccusage-codex ./apps/codex/cmd/better-ccusage-codex
go build -o dist/better-ccusage-opencode ./apps/opencode/cmd/better-ccusage-opencode

# scripts/test.sh
go test ./... -race

# scripts/fmt.sh
gofumpt -w .
goimports -w .

# scripts/lint.sh
golangci-lint run
```

### CI workflow

`.github/workflows/ci.yml` (macOS only for v1):
```yaml
name: ci
on: [push, pull_request]
jobs:
  test:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.22" }
      - run: ./scripts/test.sh
  build:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.22" }
      - run: ./scripts/build.sh
      - uses: actions/upload-artifact@v4
        with: { name: binaries, path: dist/ }
```

### Module path and Go version

- Module path: `github.com/cobra91/better-ccusage` (matches GitHub remote).
- Go version: 1.22 (supports `range over int` and `for range` over functions used in this design).

### Dependencies

| Package | Purpose | Source |
|---|---|---|
| `github.com/spf13/cobra` | CLI framework | stdlib-adjacent |
| `github.com/spf13/pflag` | POSIX flags | via cobra |
| `github.com/charmbracelet/lipgloss` | Terminal styling | Charm stack |
| `github.com/charmbracelet/bubbles` | TUI components | Charm stack |
| `github.com/charmbracelet/bubbletea` | TUI runtime | Charm stack |
| `github.com/charmbracelet/x/exp/charmtone` | Tones (optional) | Charm stack |
| `github.com/itchyny/gojq` | jq processor | standard Go jq |
| `github.com/modelcontextprotocol/go-sdk` | MCP server | official MCP |
| `github.com/mvdan/unparam` | linter helper | lint only |

No `testify`, no `decimal`, no `jsonschema` reflection libraries — stdlib handles everything else.

## Statusline Output

The statusline command outputs a single line of ANSI-styled text for Claude Code's statusline integration. The Go version uses lipgloss to render the same line. Width-adaptive, colors match consola. Output is byte-equivalent to TS for the same input on a 120-column terminal within lipgloss's ANSI escape sequence details.

## Documentation

`docs/` (VitePress) stays unchanged. Add one new page: `docs/guide/go-migration.md` documenting that better-ccusage is now Go-native under the hood, with install instructions and a note that the CLI surface is unchanged.

## Out of Scope (Deferred)

- Migration / cutover strategy: npm publishing of Go binaries, Homebrew tap, Windows MSI.
- Linux and Windows CI matrix.
- Goreleaser-based release pipeline.
- VitePress docs rewrite.
- New CLI flags, new commands, new provider adapters.
- Performance optimization beyond idiomatic Go.
- Fuzz testing.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Pricing JSON format changes upstream break parsing | Schema validation in `Fetch`; bundled fallback always present |
| Lipgloss ANSI sequences differ from `cli-table3` chalk output | Document delta; statusline consumers (Claude Code) normalize ANSI |
| JSONL files are huge for some users | Incremental reload (mtime-aware); `Loader.Reload` avoids re-parsing unchanged files |
| MCP SDK version churn | Pin SDK version; revisit on minor bumps |
| `gojq` behavior differs from `jq` for edge cases | Tests against the TS jq processor's output for common patterns |
| Decimal precision loss in cost math | `Money` type uses `int64` micros; no floats in cost paths |

## Acceptance Criteria

The rewrite is complete when:

1. Every TS command produces byte-equivalent output to the TS version for the same input fixture (golden files match).
2. Every TS provider adapter handles its known input fixtures correctly (table-driven tests pass).
3. The MCP server exposes all 4 tools with matching schemas and response shapes.
4. The statusline output matches the TS version for representative inputs.
5. The live blocks monitor produces visually equivalent output and updates correctly.
6. CI passes on macOS (build + test + lint).
7. `go test ./... -race` passes with zero failures.
8. The deprecated shims (`codex`, `opencode`) exec into the main binary and produce the same output as the TS shims.
9. All four binaries build and run on macOS (darwin/arm64, darwin/amd64).

After all criteria pass, the project enters a spike-validation phase: run the Go binaries against real user data, compare outputs to TS outputs, and only then decide on a migration/cutover strategy.
