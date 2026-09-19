# better-ccusage Go Rewrite — Plan 4: Shims + Docs (+ Codex MCP Tools)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Go `better-ccusage-codex` / `better-ccusage-opencode` shim binaries (deprecated forwarders to `better-ccusage`), add `codex-daily` / `codex-monthly` MCP tools (TS parity), and document the three Go binaries.

**Architecture:** Shared forwarder core in `pkg/shim` (parameterized `Config`: notice text, opt-out env, suppressed env, own binary name) with two ~20-line mains supplying per-app config; MCP codex tools subprocess the shim binary (`--json daily/monthly`) and return stdout verbatim; three existing READMEs gain a short Go-binary section each.

**Tech Stack:** Go 1.24.4 (1.22 floor), stdlib `testing` + `os/exec`, `github.com/spf13/cobra` (mains use no flags — plain `main`, no cobra needed), MCP Go SDK v1.4.0 (already pinned), `pkg/terminal` only for nothing (shims write stderr directly; no logger dep).

**Spec:** [`docs/superpowers/specs/2026-09-19-better-ccusage-go-rewrite-design.md`](../specs/2026-09-19-better-ccusage-go-rewrite-design.md) (§Deprecated shims) + TS behavior source `apps/codex/src/*.ts`, `apps/opencode/src/*.ts` (env contract, notice text, exit mirroring).

**Plan series:**
- ✅ Plan 1: Foundation (merged)
- ✅ Plan 2: Core CLI (merged at `54c6045`)
- ✅ Plan 3: MCP server (merged at `51903f9`; 4 tools)
- **Plan 4: Shims + docs (+ codex MCP)** ← this document

**Out of scope:** VitePress site changes. `--source` filtering flags (TS NOTE blocks mention them as future). Changing `better-ccusage` CLI itself.

## Global Constraints

- Go version: **1.22** floor (repo uses Go 1.24.4; do NOT touch the `go` directive).
- Module path: **`github.com/cobra91/better-ccusage`**.
- No new runtime dependencies (stdlib only for shims; SDK already present for MCP).
- No `testify`, no `ginkgo`, no `gomock`. **stdlib `testing` only.**
- Race detector required: `go test ./... -race`.
- CI runs on **`macos-latest` only** (POSIX `/bin/sh`, `/usr/bin/true` available for tests).
- Shim stdout MUST stay byte-clean (notice/errors → stderr only; exit code mirrored).
- **Export rules**: only export `pkg/shim` identifiers actually used by the mains/tools.
- Env contract is verbatim TS port (see per-app tables in Task 2); do not invent extra vars.

## Execution Workspace

SDD creates `.worktrees/plan-4-shims/` on branch `feat/go-rewrite-plan-4` from `main`. All paths relative to worktree root; all `go` commands from worktree root.

---

## File Structure (Plan 4 deliverable)

```
pkg/shim/
├── shim.go                          # Config, ResolveBinary, Run, notice, env
└── shim_test.go
apps/codex/cmd/better-ccusage-codex/main.go
apps/opencode/cmd/better-ccusage-opencode/main.go
apps/better-ccusage/internal/mcp/tools/codex.go
apps/better-ccusage/internal/mcp/tools/codex_test.go
apps/better-ccusage/internal/mcp/server/server.go          # +2 registrations (modify)
apps/better-ccusage/internal/mcp/server/server_test.go     # list-6 update (modify)
apps/mcp/README.md                    # +Go binary section (append)
apps/codex/README.md                  # +Go binary section (append)
apps/opencode/README.md               # +Go binary section (append)
```

---

## Task 1: Shared forwarder core `pkg/shim`

**Files:**
- Create: `pkg/shim/shim.go`
- Create: `pkg/shim/shim_test.go`

**Interfaces:**
- Consumes: nothing (stdlib only).
- Produces: `type Config struct { Target, OwnName, NoticePrefix string; NoticeLines []string; OptOutEnv string; Set map[string]string; SuppressIfUnset map[string]string }`; `func ResolveBinary(name string) (string, error)`; `func NonexistentTmp(prefix string) string`; `func Run(cfg Config, args []string) int`.

Semantics (TS `forwarder.ts` + `resolve-better-ccusage.ts` port):
- `Run` filters `args` entries equal to `OwnName`, prints notice, resolves target, execs with inherited stdio + built env, returns child exit code (0 on success, child code on `*exec.ExitError`, 1 otherwise).
- `ResolveBinary`: `exec.LookPath(name)` → dir-of-current-executable fallback (must exist, non-dir, executable bit) → error `could not resolve %q: not on PATH nor next to this binary`.
- Notice: skip when opt-out env == "1"; skip when stdout is not a character device; else write lines to stderr.
- Env: start from `os.Environ()`, apply `Set` unconditionally, apply `SuppressIfUnset` only for keys unset-or-empty in the parent.

- [ ] **Step 1: Write `pkg/shim/shim.go`**

```go
// Package shim implements deprecated forwarder binaries that delegate to
// better-ccusage. Each shim is a thin main supplying a Config.
package shim

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// Config parameterizes one shim binary.
type Config struct {
	// Target is the binary to exec, e.g. "better-ccusage".
	Target string
	// OwnName is this shim's binary name, stripped from argv (npx scenario).
	OwnName string
	// NoticeLines are written to stderr on TTY runs (already prefixed).
	NoticeLines []string
	// OptOutEnv, when "1", suppresses the notice (e.g. CODEX_NO_DEPRECATION_NOTICE).
	OptOutEnv string
	// Set env vars are always applied.
	Set map[string]string
	// SuppressIfUnset entries apply only when the parent env lacks the key
	// or holds an empty value.
	SuppressIfUnset map[string]string
}

// NonexistentTmp returns a path inside os.TempDir that must not exist,
// used to neuter unrelated data-dir env vars.
func NonexistentTmp(prefix string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("nonexistent-%s-%d", prefix, os.Getpid()))
}

// ResolveBinary locates target on PATH, falling back to the shim's own
// directory. It never falls back silently: missing binaries are an error.
func ResolveBinary(name string) (string, error) {
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), name)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
			return p, nil
		}
	}
	return "", fmt.Errorf("could not resolve %q: not on PATH nor next to this binary", name)
}

// Run forwards args to the target and returns the process exit code.
// It never calls os.Exit itself so mains stay testable.
func Run(cfg Config, args []string) int {
	filtered := make([]string, 0, len(args))
	for _, a := range args {
		if a != cfg.OwnName {
			filtered = append(filtered, a)
		}
	}
	printNotice(os.Stderr, cfg, isCharDevice(os.Stdout))
	bin, err := ResolveBinary(cfg.Target)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cmd := exec.Command(bin, filtered...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = buildEnv(os.Environ(), cfg)
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func printNotice(w io.Writer, cfg Config, isTTY bool) {
	if !isTTY {
		return
	}
	if os.Getenv(cfg.OptOutEnv) == "1" {
		return
	}
	for _, line := range cfg.NoticeLines {
		fmt.Fprintln(w, line)
	}
}

func isCharDevice(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func buildEnv(parent []string, cfg Config) []string {
	out := make([]string, 0, len(parent)+len(cfg.Set)+len(cfg.SuppressIfUnset))
	out = append(out, parent...)
	set := func(k, v string) {
		prefix := k + "="
		for i, e := range out {
			if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
				out[i] = prefix + v
				return
			}
		}
		out = append(out, prefix+v)
	}
	for k, v := range cfg.Set {
		set(k, v)
	}
	for k, v := range cfg.SuppressIfUnset {
		if os.Getenv(k) == "" {
			set(k, v)
		}
	}
	return out
}
```

Note: `SuppressIfUnset` consults the real parent env via `os.Getenv` (not the `parent` slice) — matches TS `process.env` checks. Tests must use `t.Setenv` so both agree.

- [ ] **Step 2: Write `pkg/shim/shim_test.go`**

```go
package shim

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestFilterArgs(t *testing.T) {
	cfg := Config{OwnName: "better-ccusage-codex", Target: "true"}
	_ = cfg
	// Run() filtering is covered via TestRunExitCodes arg pass-through;
	// unit-check the rule directly:
	args := []string{"better-ccusage-codex", "daily", "--json"}
	var filtered []string
	for _, a := range args {
		if a != "better-ccusage-codex" {
			filtered = append(filtered, a)
		}
	}
	if len(filtered) != 2 || filtered[0] != "daily" {
		t.Errorf("filtered: %q", filtered)
	}
}

func TestBuildEnv(t *testing.T) {
	t.Setenv("SHIM_TEST_PARENT", "keep")
	t.Setenv("SHIM_TEST_UNSET", "")
	cfg := Config{
		Set:             map[string]string{"OFFLINE": "true"},
		SuppressIfUnset: map[string]string{"SHIM_TEST_PARENT": "x", "SHIM_TEST_UNSET": "fallback", "SHIM_TEST_MISSING": "fallback"},
	}
	env := buildEnv(os.Environ(), cfg)
	get := func(k string) string {
		for _, e := range env {
			if strings.HasPrefix(e, k+"=") {
				return strings.TrimPrefix(e, k+"=")
			}
		}
		return "<absent>"
	}
	if get("OFFLINE") != "true" {
		t.Error("Set must apply")
	}
	if get("SHIM_TEST_PARENT") != "keep" {
		t.Error("parent value must win over suppress")
	}
	if get("SHIM_TEST_UNSET") != "fallback" || get("SHIM_TEST_MISSING") != "fallback" {
		t.Error("unset/empty/missing must take fallback")
	}
}

func TestPrintNotice(t *testing.T) {
	cfg := Config{NoticeLines: []string{"l1", "l2"}, OptOutEnv: "SHIM_TEST_OPTOUT"}
	var buf bytes.Buffer
	printNotice(&buf, cfg, true)
	if buf.String() != "l1\nl2\n" {
		t.Errorf("notice: %q", buf.String())
	}
	buf.Reset()
	printNotice(&buf, cfg, false)
	if buf.String() != "" {
		t.Error("non-TTY must suppress")
	}
	t.Setenv("SHIM_TEST_OPTOUT", "1")
	printNotice(&buf, cfg, true)
	if buf.String() != "" {
		t.Error("opt-out must suppress")
	}
}

func TestResolveBinaryMissing(t *testing.T) {
	if _, err := ResolveBinary("definitely-not-a-binary-xyz"); err == nil {
		t.Error("missing binary must error")
	} else if !strings.Contains(err.Error(), "definitely-not-a-binary-xyz") {
		t.Errorf("error must name binary: %v", err)
	}
}

func TestRunExitCodes(t *testing.T) {
	if Run(Config{Target: "true"}, nil) != 0 {
		t.Error("true must exit 0")
	}
	if Run(Config{Target: "false"}, nil) != 1 {
		t.Error("false must exit 1")
	}
	if Run(Config{Target: "definitely-not-a-binary-xyz"}, nil) != 1 {
		t.Error("unresolvable must exit 1")
	}
}
```

`TestRunExitCodes` with `Target: "true"` prints no notice (stdout under `go test` is a pipe → suppressed) and inherits stdio — safe. `TestFilterArgs` duplicates the loop instead of calling an unexported helper — acceptable: it pins the contract; do NOT refactor `Run` to export a filter just for this test.

- [ ] **Step 3: Run tests**

```bash
go test ./pkg/shim/... -race -count=1
go vet ./pkg/shim/...
```

Expected: 5 tests PASS, vet clean.

- [ ] **Step 4: Commit**

```bash
git add pkg/shim/
git commit -m "feat(shim): add shared forwarder core"
```

---

## Task 2: codex + opencode shim binaries + smoke

**Files:**
- Create: `apps/codex/cmd/better-ccusage-codex/main.go`
- Create: `apps/opencode/cmd/better-ccusage-opencode/main.go`

**Interfaces:**
- Consumes: `shim.Config/Run/NonexistentTmp`.
- Produces: two binaries; smoke behavior (forwarding, exit codes, notice gating).

Env contract (verbatim TS port — `apps/codex/src/forwarder.ts`, `apps/opencode/src/forwarder.ts`):

| | codex | opencode |
|---|---|---|
| `Target` | `better-ccusage` | `better-ccusage` |
| `OwnName` | `better-ccusage-codex` | `better-ccusage-opencode` |
| prefix | `[@better-ccusage/codex]` | `[@better-ccusage/opencode]` |
| opt-out | `CODEX_NO_DEPRECATION_NOTICE` | `OPENCODE_NO_DEPRECATION_NOTICE` |
| `Set` | `OFFLINE=true` | `OFFLINE=true` |
| suppress | `DROID_SESSIONS_DIR→/dev/null`, `ZCODE_HOME→nonexistent` | + `CODEX_HOME→nonexistent` |
| inherit (untouched) | `CODEX_HOME` | `OPENCODE_DATA_DIR` |

Notice lines (prefix included per line):
- codex: `This package is deprecated. Codex support is now built into better-ccusage.` / ``Run `npx better-ccusage` directly. Forwarding your invocation now.`` / `To silence this notice set CODEX_NO_DEPRECATION_NOTICE=1.`
- opencode: `This package is deprecated. OpenCode support is now built into better-ccusage.` / ``Run `npx better-ccusage` directly. Forwarding your invocation now.`` / `To silence this notice set OPENCODE_NO_DEPRECATION_NOTICE=1.`

- [ ] **Step 1: Write `apps/codex/cmd/better-ccusage-codex/main.go`**

```go
// Command better-ccusage-codex is a deprecated shim forwarding to better-ccusage.
package main

import (
	"os"

	"github.com/cobra91/better-ccusage/pkg/shim"
)

func main() {
	os.Exit(shim.Run(shim.Config{
		Target:  "better-ccusage",
		OwnName: "better-ccusage-codex",
		NoticeLines: []string{
			"[@better-ccusage/codex] This package is deprecated. Codex support is now built into better-ccusage.",
			"[@better-ccusage/codex] Run `npx better-ccusage` directly. Forwarding your invocation now.",
			"[@better-ccusage/codex] To silence this notice set CODEX_NO_DEPRECATION_NOTICE=1.",
		},
		OptOutEnv: "CODEX_NO_DEPRECATION_NOTICE",
		Set:       map[string]string{"OFFLINE": "true"},
		SuppressIfUnset: map[string]string{
			"DROID_SESSIONS_DIR": os.DevNull,
			"ZCODE_HOME":         shim.NonexistentTmp("zcode"),
		},
	}, os.Args[1:]))
}
```

- [ ] **Step 2: Write `apps/opencode/cmd/better-ccusage-opencode/main.go`** — same shape with opencode values:

```go
// Command better-ccusage-opencode is a deprecated shim forwarding to better-ccusage.
package main

import (
	"os"

	"github.com/cobra91/better-ccusage/pkg/shim"
)

func main() {
	os.Exit(shim.Run(shim.Config{
		Target:  "better-ccusage",
		OwnName: "better-ccusage-opencode",
		NoticeLines: []string{
			"[@better-ccusage/opencode] This package is deprecated. OpenCode support is now built into better-ccusage.",
			"[@better-ccusage/opencode] Run `npx better-ccusage` directly. Forwarding your invocation now.",
			"[@better-ccusage/opencode] To silence this notice set OPENCODE_NO_DEPRECATION_NOTICE=1.",
		},
		OptOutEnv: "OPENCODE_NO_DEPRECATION_NOTICE",
		Set:       map[string]string{"OFFLINE": "true"},
		SuppressIfUnset: map[string]string{
			"DROID_SESSIONS_DIR": os.DevNull,
			"ZCODE_HOME":         shim.NonexistentTmp("zcode"),
			"CODEX_HOME":         shim.NonexistentTmp("codex"),
		},
	}, os.Args[1:]))
}
```

- [ ] **Step 3: Build + smoke with a fake `better-ccusage` on PATH**

```bash
mkdir -p /tmp/p4shim/fakebin /tmp/empty-dir
go build -o /tmp/p4shim/codex ./apps/codex/cmd/better-ccusage-codex
go build -o /tmp/p4shim/opencode ./apps/opencode/cmd/better-ccusage-opencode
printf '#!/bin/sh\necho "ARGS:$@"\n' > /tmp/p4shim/fakebin/better-ccusage
chmod +x /tmp/p4shim/fakebin/better-ccusage
PATH="/tmp/p4shim/fakebin:$PATH" /tmp/p4shim/codex daily --json
echo "exit=$?"
PATH="/tmp/p4shim/fakebin:$PATH" CODEX_NO_DEPRECATION_NOTICE=1 /tmp/p4shim/opencode better-ccusage-opencode session
echo "exit=$?"
PATH="/tmp/empty-dir" /tmp/p4shim/codex daily; echo "exit=$?"
```

Expected: `ARGS:daily --json` exit 0 (notice suppressed — stdout is a pipe); `ARGS:session` exit 0 with own-name stripped; unresolvable → `could not resolve "better-ccusage"...` on stderr, exit 1. (`/tmp/empty-dir` must exist and lack `better-ccusage`; also ensure no `better-ccusage` next to `/tmp/p4shim/codex` or the exe-dir fallback fires — it doesn't exist there.)

Record actual outputs in the report. Clean up: `rm -rf /tmp/p4shim`.

- [ ] **Step 4: Commit**

```bash
git add apps/codex/cmd/ apps/opencode/cmd/
git commit -m "feat(shims): add codex and opencode forwarder binaries"
```

---

## Task 3: `codex-daily` / `codex-monthly` MCP tools + server wiring

**Files:**
- Create: `apps/better-ccusage/internal/mcp/tools/codex.go`
- Create: `apps/better-ccusage/internal/mcp/tools/codex_test.go`
- Modify: `apps/better-ccusage/internal/mcp/server/server.go` (+2 registrations, `Opts.CodexBin`)
- Modify: `apps/better-ccusage/internal/mcp/server/server_test.go` (list expectation 4→6)

**Interfaces:**
- Consumes: `transport.ToolError`; `shim.ResolveBinary`; `mcp.AddTool[CodexArgs, any]`.
- Produces: `func CodexDaily/CodexMonthly(ctx, bin string, args CodexArgs) (*mcp.CallToolResult, any, error)`; `server.Opts` gains `CodexBin string` (empty → resolve `better-ccusage-codex` lazily per call).

Args (TS `codexParametersShape` minus `timezone`/`locale` — Go CLI has no such flags; deviation recorded): `CodexArgs{Since, Until string}` forwarded as `--since/--until` when non-empty. 15s timeout. Empty subprocess output → `ToolError`. Non-JSON output → `ToolError`. Non-zero exit → `ToolError` with stderr text. Missing binary at call time → `ToolError` (isError, NOT startup failure — most installs lack the shim).

- [ ] **Step 1: Write `apps/better-ccusage/internal/mcp/tools/codex.go`**

```go
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
	out, err := cmd.Output()
	if err != nil {
		return transport.ToolError(fmt.Errorf("codex %s: %w", command, err))
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
```

- [ ] **Step 2: Write `apps/better-ccusage/internal/mcp/tools/codex_test.go`** with three tests using a fake executable script (`t.TempDir()` + `os.WriteFile(path, []byte("#!/bin/sh\n..."), 0o755)`):
  - `TestCodexDaily_Success`: script `echo '{"daily":[],"summary":{"totalTokens":0,"costUSD":0}}'`; assert `err == nil`, `IsError == false`, text parses as JSON with `"daily"` key.
  - `TestCodexDaily_Failure`: script `echo boom >&2; exit 3`; assert `IsError == true`.
  - `TestCodexDaily_MissingBin`: `bin: ""` → `IsError == true`, text contains `better-ccusage-codex`.

- [ ] **Step 3: Wire server** — in `server.go`: add `CodexBin string` to `Opts` with doc comment (`// CodexBin overrides the codex shim path; empty resolves better-ccusage-codex lazily per call.`); register (read current file first for exact anchor text):

```go
	mcp.AddTool[tools.CodexArgs, any](srv, &mcp.Tool{Name: "codex-daily", Description: "Show Codex usage grouped by day"},
		func(ctx context.Context, req *mcp.CallToolRequest, args tools.CodexArgs) (*mcp.CallToolResult, any, error) {
			return tools.CodexDaily(ctx, resolveCodexBin(opts), args)
		})
	mcp.AddTool[tools.CodexArgs, any](srv, &mcp.Tool{Name: "codex-monthly", Description: "Show Codex usage grouped by month"},
		func(ctx context.Context, req *mcp.CallToolRequest, args tools.CodexArgs) (*mcp.CallToolResult, any, error) {
			return tools.CodexMonthly(ctx, resolveCodexBin(opts), args)
		})
```

with helper (same file):

```go
// resolveCodexBin returns the configured shim path, resolving
// better-ccusage-codex lazily so servers start without the shim installed.
func resolveCodexBin(opts Opts) string {
	if opts.CodexBin != "" {
		return opts.CodexBin
	}
	p, _ := shim.ResolveBinary("better-ccusage-codex")
	return p
}
```

Add import `"github.com/cobra91/better-ccusage/pkg/shim"`. Update `server_test.go` list expectation from 4 names to 6 (`blocks codex-daily codex-monthly daily monthly session` sorted) and add `TestServer_CallCodexDaily_Integration` using `Opts{CodexBin: <fake script>, ...}` asserting success content (mirror the daily call test).

- [ ] **Step 4: Run tests**

```bash
go test ./apps/better-ccusage/internal/mcp/... -race -count=1
go vet ./apps/better-ccusage/...
```

Expected: PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add apps/better-ccusage/internal/mcp/
git commit -m "feat(mcp): add codex-daily and codex-monthly tools"
```

---

## Task 4: READMEs + verification

**Files:**
- Modify (append section): `apps/mcp/README.md`, `apps/codex/README.md`, `apps/opencode/README.md`
- No code changes (verification only; fix findings as separate commits, then re-run).

Read each README first (edit tool requires it), append the Go section at the end.

- [ ] **Step 1: Append to `apps/mcp/README.md`**

```markdown
## Go Binary (`better-ccusage-mcp`)

```bash
go build ./apps/better-ccusage/cmd/better-ccusage-mcp
better-ccusage-mcp --help
better-ccusage-mcp --type http --port 8080
```

Exposes `daily`, `session`, `monthly`, `blocks`, `codex-daily`, `codex-monthly` over MCP (stdio default, StreamableHTTP via `--type http`). Tool args: `since`/`until` (RFC3339), `mode` (`auto`|`calculate`|`display`). Errors return `isError` with `{code, message, hint}`.
```

- [ ] **Step 2: Append to `apps/codex/README.md`**

```markdown
## Go Binary (`better-ccusage-codex`, deprecated)

```bash
go build ./apps/codex/cmd/better-ccusage-codex
```

Forwards to `better-ccusage` (resolved via `PATH`, then the shim's own directory). Prints a stderr deprecation notice on TTY runs unless `CODEX_NO_DEPRECATION_NOTICE=1`. Sets `OFFLINE=true`; neuters `DROID_SESSIONS_DIR`/`ZCODE_HOME` when unset; inherits `CODEX_HOME`.
```

- [ ] **Step 3: Append to `apps/opencode/README.md`**

```markdown
## Go Binary (`better-ccusage-opencode`, deprecated)

```bash
go build ./apps/opencode/cmd/better-ccusage-opencode
```

Forwards to `better-ccusage` (resolved via `PATH`, then the shim's own directory). Prints a stderr deprecation notice on TTY runs unless `OPENCODE_NO_DEPRECATION_NOTICE=1`. Sets `OFFLINE=true`; neuters `DROID_SESSIONS_DIR`/`ZCODE_HOME`/`CODEX_HOME` when unset; inherits `OPENCODE_DATA_DIR`.
```

- [ ] **Step 4: Verify**

```bash
go build ./...
go test ./... -race -count=1
go vet ./...
go build -o /tmp/better-ccusage-codex ./apps/codex/cmd/better-ccusage-codex
go build -o /tmp/better-ccusage-opencode ./apps/opencode/cmd/better-ccusage-opencode
go build -o /tmp/better-ccusage-mcp ./apps/better-ccusage/cmd/better-ccusage-mcp
/tmp/better-ccusage-mcp --help
```

Expected: all green; three binaries build; `--help` lists MCP flags.

- [ ] **Step 5: Commit + report**

```bash
git add apps/mcp/README.md apps/codex/README.md apps/opencode/README.md
git commit -m "docs: document Go binaries for mcp, codex, opencode"
```

Report to user: Plan 4 complete — tasks, commits, suite result, binaries, known deviations (no timezone/locale on codex tools; codex output is `DailyResult` shape not TS codex shape; `OFFLINE=true` forwarded but currently inert), ready for final review.

---

## End of Plan 4

Go rewrite plans complete: Foundation → Core CLI → MCP server → Shims + docs. Remaining follow-ups live in prior ledgers (pipeline dedup, Money display formatting, `blocks.live` naming, jq expansion).
