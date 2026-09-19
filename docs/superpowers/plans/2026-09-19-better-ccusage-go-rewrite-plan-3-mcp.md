# better-ccusage Go Rewrite — Plan 3: MCP Server

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the MCP server binary `better-ccusage-mcp` exposing `daily`, `session`, `monthly`, `blocks` tools over stdio and StreamableHTTP by calling `commands.X.Run` directly.

**Architecture:** Single binary `better-ccusage-mcp` built from `apps/better-ccusage/cmd/better-ccusage-mcp/main.go` using cobra. `internal/mcp/tools/` handlers parse MCP args into `commands.CommonOpts` and call `commands.Daily/Monthly/Session/Blocks` directly (no subprocess); `internal/mcp/server/` registers the 4 tools on the official MCP Go SDK server; `internal/mcp/transport/` owns stdio/HTTP wiring and the sentinel-to-MCP error map.

**Tech Stack:** Go 1.24.4 (repo toolchain; 1.22 floor), stdlib `testing`, `github.com/spf13/cobra`, `github.com/modelcontextprotocol/go-sdk@v1.4.0` (newest release supporting go < 1.25; v1.5.0+ requires go ≥ 1.25), `github.com/cobra91/better-ccusage/apps/better-ccusage/internal/commands` + `pkg/pricing` + `pkg/terminal` (Plan 2).

**Spec:** [`docs/superpowers/specs/2026-09-19-better-ccusage-go-rewrite-design.md`](../specs/2026-09-19-better-ccusage-go-rewrite-design.md) (§`apps/better-ccusage/internal/mcp/`, §MCP tool flow, §MCP error mapping)

**Plan series:**
- ✅ Plan 1: Foundation (merged)
- ✅ Plan 2: Core CLI (merged to main at `54c6045`)
- **Plan 3: MCP server** ← this document
- Plan 4: Shims + docs

**Out of scope (deferred):** `codex-daily` / `codex-monthly` tools (TS parity gap; needs a codex runner — revisit in Plan 4 when the codex binary exists). `--jq` on MCP tools (CLI-only flag). `weekly` / `statusline` / `blocks.live` as MCP tools (spec lists 4).

## Global Constraints

- Go version: **1.22** floor (repo uses Go 1.24.4; do NOT bump the `go` directive — SDK v1.4.0 was chosen to fit).
- Module path: **`github.com/cobra91/better-ccusage`**.
- All runtime libs go in `go.mod` as direct deps (run `go mod tidy`; no `// indirect` on directly-imported packages).
- No `testify`, no `ginkgo`, no `gomock`. **stdlib `testing` only.**
- **No floats in cost math.** Cost paths already exist (Plan 2); this plan only marshals results.
- Race detector required: `go test ./... -race`.
- CI runs on **`macos-latest` only**.
- Strict behavior port from TS `apps/mcp`: tool names, descriptions, `{content: [{type: "text", text: <JSON 2-space>}]}` shape, `isError` error shape.
- Logger writes to **`os.Stderr`** via `pkg/terminal.Logger`. stdio transport MUST silence all logging (level 0).
- Sentinel errors live in **`apps/better-ccusage/internal/errs/`** (additive change only: one new sentinel in Task 1).
- Test naming: `TestXxx` for unit, `TestXxx_Integration` for client↔server tests.
- Tool `since`/`until` args are **RFC3339** strings (the only format the Go command layer parses). TS accepted YYYYMMDD — NOT ported; document in tool descriptions.

## Execution Workspace

SDD creates `.worktrees/plan-3-mcp/` on branch `feat/go-rewrite-plan-3` from `main`. All file paths below are relative to the worktree root; all `go` commands run from the worktree root.

---

## File Structure (Plan 3 deliverable)

```
apps/better-ccusage/
├── cmd/better-ccusage-mcp/main.go       # cobra root: --mode/-m, --type/-t, --port/-p
└── internal/mcp/
    ├── transport/
    │   ├── errors.go                    # MapError + ErrorResult
    │   └── errors_test.go
    ├── tools/
    │   ├── args.go                      # ReportArgs, Deps, ParseCommon, encodeResult
    │   ├── args_test.go
    │   ├── daily.go                     # Daily handler
    │   ├── daily_test.go
    │   ├── session.go                   # Session handler
    │   ├── session_test.go
    │   ├── monthly.go                   # Monthly handler
    │   ├── monthly_test.go
    │   ├── blocks.go                    # Blocks handler
    │   └── blocks_test.go
    └── server/
        ├── server.go                    # NewServer, 4 tool registrations
        └── server_test.go               # in-memory list-tools + call-daily
```

> **Layout note (SDD ruling):** the design spec sketches this under `apps/mcp/internal/`, but Go's `internal` visibility rule forbids `apps/mcp/...` from importing `apps/better-ccusage/internal/...` (compiler-verified in Task 1). All MCP code therefore lives under `apps/better-ccusage/` (`cmd/better-ccusage-mcp`, `internal/mcp/`). Behavior, tool set, and direct-`Run` calls are unchanged; only the directory prefix differs. Binary name `better-ccusage-mcp` is unaffected.

---

## Task 1: SDK dep + error map + invalid-args sentinel

**Files:**
- Modify: `go.mod`, `go.sum` (after `go get`), `apps/better-ccusage/internal/errs/errs.go` (append one sentinel)
- Create: `apps/better-ccusage/internal/mcp/transport/errors.go`
- Create: `apps/better-ccusage/internal/mcp/transport/errors_test.go`

**Interfaces:**
- Consumes: `apps/better-ccusage/internal/errs` sentinels (`ErrNoData`, `ErrUnknownModel`, `ErrInvalidJSON`, `ErrConfigNotFound`, `ErrIncompatibleMode`, `ErrInvalidMode`).
- Produces: `func MapError(err error) (code, message string)`; `func ErrorResult(code, message, hint string) (*mcp.CallToolResult, any, error)`; new sentinel `errs.ErrInvalidArgs`.

- [ ] **Step 1: Add the SDK dependency**

```bash
go get github.com/modelcontextprotocol/go-sdk@v1.4.0
```

- [ ] **Step 2: Append the sentinel** to the `var (...)` block in `apps/better-ccusage/internal/errs/errs.go`:

```go
	ErrInvalidArgs      = errors.New("invalid tool arguments")
```

Keep the existing alignment (gofmt will enforce it; run `gofmt -l` after).

- [ ] **Step 3: Write `apps/better-ccusage/internal/mcp/transport/errors.go`**

```go
// Package transport owns MCP transport wiring and the sentinel-to-MCP
// error map for the MCP server.
package transport

import (
	"encoding/json"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp",

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
)

// ErrorPayload is the structured MCP tool-error body.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

// MapError maps a sentinel error to an (MCP code, message) pair.
func MapError(err error) (code, message string) {
	switch {
	case errors.Is(err, errs.ErrNoData):
		return "NO_DATA", err.Error()
	case errors.Is(err, errs.ErrUnknownModel):
		return "UNKNOWN_MODEL", err.Error()
	case errors.Is(err, errs.ErrInvalidMode), errors.Is(err, errs.ErrInvalidArgs):
		return "INVALID_ARGS", err.Error()
	default:
		return "INTERNAL", err.Error()
	}
}

// HintFor returns the remediation hint for an MCP error code.
func HintFor(code string) string {
	switch code {
	case "NO_DATA":
		return "no usage entries in range; check --since/--until or data directories"
	case "UNKNOWN_MODEL":
		return "model missing from pricing table; check embedded pricing version"
	case "INVALID_ARGS":
		return "check mode (auto|calculate|display) and RFC3339 since/until"
	default:
		return "unexpected error; retry or inspect server logs"
	}
}

// ErrorResult builds an isError MCP result with a structured payload.
// It returns a nil error so the LLM client sees the failure as tool
// output (per CallToolResult docs) instead of a protocol error.
func ErrorResult(code, message, hint string) (*mcp.CallToolResult, any, error) {
	body, _ := json.Marshal(ErrorPayload{Code: code, Message: message, Hint: hint})
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(body)}},
		IsError: true,
	}, nil, nil
}

// ToolError maps err via MapError and returns it as an isError result.
// Pass the original err so errors.Is chains survive.
func ToolError(err error) (*mcp.CallToolResult, any, error) {
	code, message := MapError(err)
	return ErrorResult(code, message, HintFor(code))
}

- [ ] **Step 4: Write `apps/better-ccusage/internal/mcp/transport/errors_test.go`**

```go
package transport

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
)

func TestMapError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code string
	}{
		{"nodata", errs.ErrNoData, "NO_DATA"},
		{"unknownmodel", errs.ErrUnknownModel, "UNKNOWN_MODEL"},
		{"invalidmode", errs.ErrInvalidMode, "INVALID_ARGS"},
		{"invalidargs", errs.ErrInvalidArgs, "INVALID_ARGS"},
		{"wrapped", errors.Join(errs.ErrNoData, errors.New("x")), "NO_DATA"},
		{"default", errors.New("boom"), "INTERNAL"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, msg := MapError(tc.err)
			if code != tc.code {
				t.Errorf("code: got %q, want %q", code, tc.code)
			}
			if msg == "" {
				t.Error("message must not be empty")
			}
		})
	}
}

func TestErrorResult_Shape(t *testing.T) {
	res, _, err := ErrorResult("NO_DATA", "no usage data found", "hint")
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("IsError must be true")
	}
	if len(res.Content) != 1 {
		t.Fatalf("content length: got %d, want 1", len(res.Content))
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content type: got %T, want *mcp.TextContent", res.Content[0])
	}
	var p ErrorPayload
	if err := json.Unmarshal([]byte(tc.Text), &p); err != nil {
		t.Fatal(err)
	}
	if p.Code != "NO_DATA" || p.Message == "" || p.Hint == "" {
		t.Errorf("payload: %+v", p)
	}
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./apps/better-ccusage/internal/mcp/transport/... -race -count=1
go vet ./apps/better-ccusage/...
```

Expected: PASS (6 subtests + 1 shape test), vet clean.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum apps/better-ccusage/internal/mcp/transport/ apps/better-ccusage/internal/errs/errs.go
git commit -m "feat(mcp): add SDK dep and MCP error map"
```

---

## Task 2: Tool args parsing + JSON encoding shared helpers

**Files:**
- Create: `apps/better-ccusage/internal/mcp/tools/args.go`
- Create: `apps/better-ccusage/internal/mcp/tools/args_test.go`

**Interfaces:**
- Consumes: `transport.ToolError`; `commands.CommonOpts`; `cost.CostAuto/CostCalculate/CostDisplay`; `errs.ErrInvalidMode/ErrInvalidArgs`.
- Produces: `type Deps struct { ConfigDir string; DefaultMode cost.CostMode; Prices *pricing.PriceTable }`; `type ReportArgs struct { Since, Until, Mode string }`; `func ParseCommon(d Deps, args ReportArgs) (commands.CommonOpts, error)`; `func EncodeResult(v any) (*mcp.CallToolResult, any, error)`.

- [ ] **Step 1: Write `apps/better-ccusage/internal/mcp/tools/args.go`**

```go
// Package tools implements the MCP tool handlers for the MCP server.
// One file per tool; shared arg parsing and encoding lives here.
package tools

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/commands"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/mcp/transport"
	"github.com/cobra91/better-ccusage/pkg/pricing"
)

// Deps are the server-owned inputs every tool handler needs.
type Deps struct {
	ConfigDir   string
	DefaultMode cost.CostMode
	Prices      *pricing.PriceTable
}

// ReportArgs are the MCP arguments shared by daily/session/monthly/blocks.
// Dates are RFC3339 (the only format the Go command layer parses).
type ReportArgs struct {
	Since string `json:"since,omitempty" jsonschema:"Filter entries since this RFC3339 timestamp"`
	Until string `json:"until,omitempty" jsonschema:"Filter entries until this RFC3339 timestamp"`
	Mode  string `json:"mode,omitempty" jsonschema:"Cost calculation mode: auto, calculate, display"`
}

// ParseCommon converts MCP args into commands.CommonOpts.
// Empty mode falls back to the server default mode.
func ParseCommon(d Deps, args ReportArgs) (commands.CommonOpts, error) {
	var out commands.CommonOpts
	out.ConfigDir = d.ConfigDir
	switch args.Mode {
	case "":
		out.Mode = d.DefaultMode
	case "auto":
		out.Mode = cost.CostAuto
	case "calculate":
		out.Mode = cost.CostCalculate
	case "display":
		out.Mode = cost.CostDisplay
	default:
		return commands.CommonOpts{}, fmt.Errorf("%w: %q", errs.ErrInvalidMode, args.Mode)
	}
	if args.Since != "" {
		t, err := time.Parse(time.RFC3339, args.Since)
		if err != nil {
			return commands.CommonOpts{}, fmt.Errorf("%w: since %q", errs.ErrInvalidArgs, args.Since)
		}
		out.Since = &t
	}
	if args.Until != "" {
		t, err := time.Parse(time.RFC3339, args.Until)
		if err != nil {
			return commands.CommonOpts{}, fmt.Errorf("%w: until %q", errs.ErrInvalidArgs, args.Until)
		}
		out.Until = &t
	}
	return out, nil
}

// EncodeResult marshals a command result as 2-space JSON text content.
func EncodeResult(v any) (*mcp.CallToolResult, any, error) {
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return transport.ToolError(err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(body)}},
	}, nil, nil
}
```

Remove the `context` import and the `var _` seam if unused — do not ship dead code (drop both lines if no identifier in this file uses `context`).

- [ ] **Step 2: Write `apps/better-ccusage/internal/mcp/tools/args_test.go`**

```go
package tools

import (
	"testing"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"
)

func TestParseCommon(t *testing.T) {
	d := Deps{ConfigDir: "/tmp/x", DefaultMode: cost.CostAuto}
	got, err := ParseCommon(d, ReportArgs{Mode: "calculate", Since: "2026-01-01T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != cost.CostCalculate {
		t.Errorf("mode: got %v, want calculate", got.Mode)
	}
	if got.ConfigDir != "/tmp/x" {
		t.Errorf("configdir: got %q", got.ConfigDir)
	}
	if got.Since == nil || got.Until != nil {
		t.Error("since must parse, until must stay nil")
	}

	def, err := ParseCommon(d, ReportArgs{})
	if err != nil {
		t.Fatal(err)
	}
	if def.Mode != cost.CostAuto {
		t.Errorf("default mode: got %v", def.Mode)
	}

	if _, err := ParseCommon(d, ReportArgs{Mode: "bogus"}); err == nil {
		t.Error("bad mode must error")
	}
	if _, err := ParseCommon(d, ReportArgs{Since: "not-a-date"}); err == nil {
		t.Error("bad since must error")
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./apps/better-ccusage/internal/mcp/tools/... -race -count=1
go vet ./apps/better-ccusage/...
```

Expected: PASS, vet clean.

- [ ] **Step 4: Commit**

```bash
git add apps/better-ccusage/internal/mcp/tools/
git commit -m "feat(mcp): add shared tool arg parsing"
```

---

## Task 3: `daily` tool

**Files:**
- Create: `apps/better-ccusage/internal/mcp/tools/daily.go`
- Create: `apps/better-ccusage/internal/mcp/tools/daily_test.go`

**Interfaces:**
- Consumes: `ParseCommon`, `EncodeResult`, `transport.ToolError`; `commands.Daily(ctx, commands.DailyOpts, io.Discard, prices) (output.DailyResult, error)`.
- Produces: `func Daily(ctx context.Context, d Deps, args ReportArgs) (*mcp.CallToolResult, any, error)`.

For the success test fixture, mirror `apps/better-ccusage/internal/commands/daily_test.go` (fixture JSONL shape + `t.Setenv("CLAUDE_CONFIG_DIR", dir)` + `isolateHome`-style HOME shadowing — read that file first; `commands` tests shadow HOME because `ResolveDirs` appends default dirs).

- [ ] **Step 1: Write `apps/better-ccusage/internal/mcp/tools/daily.go`**

```go
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
```

- [ ] **Step 2: Write `apps/better-ccusage/internal/mcp/tools/daily_test.go`** with two tests:
  - `TestDaily_Integration`: fixture dir with one `projects/p/session1.jsonl` entry (copy the entry shape from `apps/better-ccusage/internal/commands/daily_test.go`), `t.Setenv("CLAUDE_CONFIG_DIR", dir)`, shadow HOME to an empty temp dir, `Deps{ConfigDir: dir, DefaultMode: cost.CostAuto, Prices: nil}` (nil prices: `ApplyPrices` must tolerate nil via `Lookup` nil-receiver guard — if it panics, pass a table from `pricing.LoadPrices` on the embedded JSON instead and note it in the report). Call `Daily(ctx, d, ReportArgs{})`, assert `err == nil`, `res.IsError == false`, content[0] is `*mcp.TextContent` whose text parses as JSON containing a `"daily"` key.
  - `TestDaily_InvalidMode`: `Daily(ctx, d, ReportArgs{Mode: "bogus"})` → `res.IsError == true`, text contains `INVALID_ARGS`.

- [ ] **Step 3: Run tests**

```bash
go test ./apps/better-ccusage/internal/mcp/tools/... -race -count=1 -run 'TestDaily|TestParseCommon'
go vet ./apps/better-ccusage/...
```

Expected: PASS, vet clean.

- [ ] **Step 4: Commit**

```bash
git add apps/better-ccusage/internal/mcp/tools/daily.go apps/better-ccusage/internal/mcp/tools/daily_test.go
git commit -m "feat(mcp): add daily tool"
```

---

## Task 4: `session` tool

**Files:**
- Create: `apps/better-ccusage/internal/mcp/tools/session.go`
- Create: `apps/better-ccusage/internal/mcp/tools/session_test.go`

**Interfaces:**
- Consumes: same seams as Task 3; `commands.Session(ctx, commands.SessionOpts, io.Discard, prices) (output.DailyResult, error)`.
- Produces: `func Session(ctx context.Context, d Deps, args ReportArgs) (*mcp.CallToolResult, any, error)`.

Mirror Task 3 exactly with `GroupBySession` behavior: the integration fixture uses two entries with the same session so they collapse to one row (mirror `apps/better-ccusage/internal/commands/session_test.go`).

- [ ] **Step 1: Write `apps/better-ccusage/internal/mcp/tools/session.go`**

```go
package tools

import (
	"context"
	"io"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/commands"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/mcp/transport"
)

// Session implements the MCP `session` tool: usage grouped by conversation session.
func Session(ctx context.Context, d Deps, args ReportArgs) (*mcp.CallToolResult, any, error) {
	common, err := ParseCommon(d, args)
	if err != nil {
		return transport.ToolError(err)
	}
	result, err := commands.Session(ctx, commands.SessionOpts{CommonOpts: common}, io.Discard, d.Prices)
	if err != nil {
		return transport.ToolError(err)
	}
	return EncodeResult(result)
}
```

- [ ] **Step 2: Write `apps/better-ccusage/internal/mcp/tools/session_test.go`** (`TestSession_Integration` asserting one collapsed row / JSON `"daily"` key present; `TestSession_NoData` with empty temp dir asserting `IsError == true` and text contains `NO_DATA`).

- [ ] **Step 3: Run tests**

```bash
go test ./apps/better-ccusage/internal/mcp/tools/... -race -count=1 -run 'TestSession'
go vet ./apps/better-ccusage/...
```

Expected: PASS, vet clean.

- [ ] **Step 4: Commit**

```bash
git add apps/better-ccusage/internal/mcp/tools/session.go apps/better-ccusage/internal/mcp/tools/session_test.go
git commit -m "feat(mcp): add session tool"
```

---

## Task 5: `monthly` tool

**Files:**
- Create: `apps/better-ccusage/internal/mcp/tools/monthly.go`
- Create: `apps/better-ccusage/internal/mcp/tools/monthly_test.go`

**Interfaces:**
- Consumes: same seams; `commands.Monthly(ctx, commands.MonthlyOpts, io.Discard, prices) (output.DailyResult, error)`.
- Produces: `func Monthly(ctx context.Context, d Deps, args ReportArgs) (*mcp.CallToolResult, any, error)`.

Mirror Task 3 with `GroupByMonth` behavior (mirror `apps/better-ccusage/internal/commands/monthly_test.go`: two fixture dates in the same month collapse to one row).

- [ ] **Step 1: Write `apps/better-ccusage/internal/mcp/tools/monthly.go`**

```go
package tools

import (
	"context"
	"io"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/commands"
	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/mcp/transport"
)

// Monthly implements the MCP `monthly` tool: usage report grouped by month.
func Monthly(ctx context.Context, d Deps, args ReportArgs) (*mcp.CallToolResult, any, error) {
	common, err := ParseCommon(d, args)
	if err != nil {
		return transport.ToolError(err)
	}
	result, err := commands.Monthly(ctx, commands.MonthlyOpts{CommonOpts: common}, io.Discard, d.Prices)
	if err != nil {
		return transport.ToolError(err)
	}
	return EncodeResult(result)
}
```

- [ ] **Step 2: Write `apps/better-ccusage/internal/mcp/tools/monthly_test.go`** (`TestMonthly_Integration` + `TestMonthly_InvalidDate` with `Since: "not-a-date"` asserting `INVALID_ARGS`).

- [ ] **Step 3: Run tests**

```bash
go test ./apps/better-ccusage/internal/mcp/tools/... -race -count=1 -run 'TestMonthly'
go vet ./apps/better-ccusage/...
```

Expected: PASS, vet clean.

- [ ] **Step 4: Commit**

```bash
git add apps/better-ccusage/internal/mcp/tools/monthly.go apps/better-ccusage/internal/mcp/tools/monthly_test.go
git commit -m "feat(mcp): add monthly tool"
```

---

## Task 6: `blocks` tool

**Files:**
- Create: `apps/better-ccusage/internal/mcp/tools/blocks.go`
- Create: `apps/better-ccusage/internal/mcp/tools/blocks_test.go`

**Interfaces:**
- Consumes: same seams; `commands.Blocks(ctx, commands.BlocksOpts, io.Discard, prices) (output.DailyResult, error)` where `type BlocksOpts struct { CommonOpts; Active bool; Recent bool; TokenLimit int64 }`.
- Produces: `func Blocks(ctx context.Context, d Deps, args ReportArgs) (*mcp.CallToolResult, any, error)`.

MCP exposes no active/recent/token-limit args (TS parity: same 3-arg schema) — construct `BlocksOpts{CommonOpts: common}` with zero values.

- [ ] **Step 1: Write `apps/better-ccusage/internal/mcp/tools/blocks.go`**

```go
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
```

- [ ] **Step 2: Write `apps/better-ccusage/internal/mcp/tools/blocks_test.go`** (`TestBlocks_Integration` + `TestBlocks_NoData` asserting `NO_DATA`).

- [ ] **Step 3: Run tests**

```bash
go test ./apps/better-ccusage/internal/mcp/tools/... -race -count=1 -run 'TestBlocks'
go vet ./apps/better-ccusage/...
```

Expected: PASS, vet clean.

- [ ] **Step 4: Commit**

```bash
git add apps/better-ccusage/internal/mcp/tools/blocks.go apps/better-ccusage/internal/mcp/tools/blocks_test.go
git commit -m "feat(mcp): add blocks tool"
```

---

## Task 7: Server with 4 tool registrations + in-memory tests

**Files:**
- Create: `apps/better-ccusage/internal/mcp/server/server.go`
- Create: `apps/better-ccusage/internal/mcp/server/server_test.go`

**Interfaces:**
- Consumes: `tools.Deps/Daily/Session/Monthly/Blocks`; `mcp.NewServer`, `mcp.AddTool[ReportArgs, any]`, `mcp.NewInMemoryTransports`, `mcp.NewClient`.
- Produces: `type Opts struct { ConfigDir string; DefaultMode cost.CostMode; Prices *pricing.PriceTable; Version string }`; `func NewServer(opts Opts) *mcp.Server`.

Server name const: `better-ccusage-mcp`. Tool descriptions (TS parity, from `apps/mcp/src/mcp.ts`): daily `Show usage report grouped by date`, session `Show usage report grouped by conversation session`, monthly `Show usage report grouped by month`, blocks `Show usage report grouped by session billing blocks`.

- [ ] **Step 1: Write `apps/better-ccusage/internal/mcp/server/server.go`**

```go
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
```

- [ ] **Step 2: Write `apps/better-ccusage/internal/mcp/server/server_test.go`** with two tests:
  - `TestNewServer_ListsFourTools_Integration`: `NewServer(Opts{Version: "test"})`, `ct, st := mcp.NewInMemoryTransports()`, `mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)`. Connect the SERVER first, then the client (SDK requires server-first: the client initializes the session during connection): `srvSess, err := srv.Connect(ctx, st, nil)` then `cliSess, err := client.Connect(ctx, ct, nil)`. `res, err := cliSess.ListTools(ctx, &mcp.ListToolsParams{})`, assert `err == nil` and tool names are exactly `[blocks daily monthly session]` (collect `res.Tools[i].Name`, sort before comparing). Close both sessions at the end.
  - `TestServer_CallDaily_Integration`: fixture dir (same shape as Task 3), `NewServer(Opts{ConfigDir: dir, DefaultMode: cost.CostAuto, Prices: nil, Version: "test"})` (same nil-prices rule as Task 3), in-memory pair (server first, then client), `session.CallTool(ctx, &mcp.CallToolParams{Name: "daily", Arguments: map[string]any{"mode": "auto"}})`, assert `err == nil`, `res.IsError == false`, one `*mcp.TextContent` with JSON containing `"daily"`. `Server.Connect` takes `(ctx, transport, *ServerSessionOptions)` — pass `nil` options.

For `ListTools`/`CallTool` signatures, run `go doc github.com/modelcontextprotocol/go-sdk/mcp ClientSession` if the test does not compile on first try.

- [ ] **Step 3: Run tests**

```bash
go test ./apps/better-ccusage/internal/mcp/server/... -race -count=1
go vet ./apps/better-ccusage/...
```

Expected: PASS, vet clean.

- [ ] **Step 4: Commit**

```bash
git add apps/better-ccusage/internal/mcp/server/
git commit -m "feat(mcp): add server with four tools"
```

---

## Task 8: CLI binary with stdio + HTTP transports

**Files:**
- Create: `apps/better-ccusage/cmd/better-ccusage-mcp/main.go`

**Interfaces:**
- Consumes: `server.NewServer/Opts/Name`; `config.Load/DefaultPath/ResolveDirs`; `pricing.LoadPrices/EmbeddedPrices`; `terminal.NewLogger/NewLoggerFromEnv/Silent`; `mcp.StdioTransport`, `mcp.NewStreamableHTTPHandler`.
- Produces: `func main()` + `func newRootCmd(log *terminal.Logger) *cobra.Command` (same shape as Plan 2 root for testability).

Flags (TS `command.ts` parity): `--mode/-m` default `auto`, `--type/-t` default `stdio`, `--port/-p` default `8080`. Unknown `--type` → error `unsupported MCP type: %q`. No `--config-dir` flag (TS has none; server resolves at startup).

Startup: `cfg, _ := config.Load(config.DefaultPath())`; `dirs := config.ResolveDirs("", cfg)`; if `len(dirs) == 0` → return error `No valid Claude data directories found` (TS message parity). `ConfigDir: strings.Join(dirs, ",")` (commands accept comma-separated). Pricing: `pricing.LoadPrices(bytes.NewReader(pricing.EmbeddedPrices))`, warn-and-continue on error (Plan 2 root pattern). stdio: silence logger via `terminal.NewLogger(int(terminal.Silent), os.Stderr)` before `srv.Run`. HTTP: `http.ListenAndServe(":"+strconv.Itoa(port), mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, &mcp.StreamableHTTPOptions{}))`.

- [ ] **Step 1: Write `apps/better-ccusage/cmd/better-ccusage-mcp/main.go`**

```go
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
```

The reassignment at the top of `RunE` makes the passed-in `log` silent in stdio mode before any `Warn`/`Info` call. The mode flag maps via `parseMode` below (same vocabulary as Plan 2): unknown values fall back to `cost.CostAuto` with no error.

`parseMode` helper (same file):

```go
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
```

Needs import `"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/cost"`. Add it; drop unused imports after the logger rewrite.

- [ ] **Step 2: Verify it builds and lists help**

```bash
go build ./apps/better-ccusage/cmd/better-ccusage-mcp
go vet ./apps/better-ccusage/...
./better-ccusage-mcp --help
```

Expected: build OK, vet clean, help shows `--mode/-m`, `--type/-t`, `--port/-p`. (Binary lands in repo root as `better-ccusage-mcp` — delete it after the smoke check: `rm -f better-ccusage-mcp`.)

- [ ] **Step 3: Commit**

```bash
git add apps/better-ccusage/cmd/
git commit -m "feat(mcp): add MCP server binary with stdio and HTTP"
```

---

## Task 9: End-to-end verification

**Files:** none (verification only; fix anything found as separate commits, then re-run).

- [ ] **Step 1: Build everything**

```bash
go build ./...
```

Expected: clean.

- [ ] **Step 2: Full test suite with race detector**

```bash
go test ./... -race -count=1
```

Expected: all packages green (Plan 2: 132 passed / 14 pkgs + new `apps/better-ccusage/internal/mcp/...` packages).

- [ ] **Step 3: Vet**

```bash
go vet ./...
```

Expected: no diagnostics.

- [ ] **Step 4: Binary smoke**

```bash
go build -o /tmp/better-ccusage-mcp ./apps/better-ccusage/cmd/better-ccusage-mcp
/tmp/better-ccusage-mcp --help
/tmp/better-ccusage-mcp --type bogus 2>&1 | head -3
```

Expected: help lists flags; bogus type errors (exit non-zero) — note: it will first resolve dirs; on a machine with Claude data it proceeds to the type switch and errors `unsupported MCP type`; on a machine without data it errors `No valid Claude data directories found`. Either error proves wiring; record which in the report.

- [ ] **Step 5: Report to user**

Tell the user: Plan 3 complete — tasks, commits, `go test ./... -race` result, binary behavior, known TS deviations (4 tools not 6; RFC3339 dates; empty data → `NO_DATA` isError instead of empty arrays; `summary` key instead of `totals`), ready for final review.

---

## End of Plan 3

Plan 4 (shims + docs) will add `apps/codex` + `apps/opencode` Go binaries and revisit `codex-daily`/`codex-monthly` MCP parity.
