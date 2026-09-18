# better-ccusage Go Rewrite — Plan 1: Foundation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bootstrap the Go module, set up scripts and CI, and implement the two shared packages (`pkg/pricing`, `pkg/terminal`) that every Go app in the rewrite depends on.

**Architecture:** Single Go module at repo root, Go 1.22+, import path `github.com/cobra91/better-ccusage`. Two shared packages in `pkg/`: `pricing` (embedded JSON + live fetcher + 3-tier fuzzy lookup + Money type) and `terminal` (Charm-stack table renderer + leveled logger + ANSI normalize helper). All I/O behind public methods; pure functions for math.

**Tech Stack:** Go 1.22, stdlib `testing`, `github.com/spf13/cobra` + `pflag` (added in Plan 2), `github.com/charmbracelet/lipgloss` + `bubbles` + `bubbletea`, `github.com/itchyny/gojq` (added in Plan 2), `github.com/modelcontextprotocol/go-sdk` (added in Plan 4). Foundation plan adds only `lipgloss` (for `pkg/terminal/colors.go`).

**Spec:** [`docs/superpowers/specs/2026-09-19-better-ccusage-go-rewrite-design.md`](../specs/2026-09-19-better-ccusage-go-rewrite-design.md)

**Plan series:**
- **Plan 1: Foundation** ← this document
- Plan 2: Core CLI (`apps/better-ccusage` — data loader, cost calc, adapters, commands, live monitor)
- Plan 3: MCP server (`apps/mcp` — server, transport, tools)
- Plan 4: Shims + docs (`apps/codex`, `apps/opencode`, VitePress docs page, full end-to-end verification)

## Global Constraints

These are extracted verbatim from the spec and apply to every task in this plan:

- Go version: **1.22** (`range over int`, `for range` over functions).
- Module path: **`github.com/cobra91/better-ccusage`**.
- All runtime libs go in **`go.mod` as direct deps** (not sub-module deps). The bundler-style "deps in devDependencies" rule from TS CLAUDE.md does not apply to Go.
- No `testify`, no `ginkgo`, no `gomock`. **stdlib `testing` only.**
- **No floats in cost math.** Money is `struct{ Micros int64 }`. All cost arithmetic uses int64.
- Test naming: `TestXxx` for unit, `TestXxx_Integration` for full-pipeline tests.
- Race detector required: `go test ./... -race`.
- CI runs on **`macos-latest` only** for v1.
- Strict 1:1 behavior port from TS — pricing lookup order is `exact → provider-prefix suffix → fuzzy scorer`; confidence < 0.6 returns `(Price{}, 0.0)`.
- Logger writes to **`os.Stderr`** (keeps stdout clean for JSON output).
- `LOG_LEVEL` semantics: `0`=silent, `1`=warn, `2`=log, `3`=info, `4`=debug, `5`=trace.
- No `console.log` / `fmt.Println` for runtime output — use `pkg/terminal.Logger`.
- Sentinel errors live in **`apps/better-ccusage/internal/errs/`** (added in Plan 2). Foundation uses ad-hoc errors wrapped with `fmt.Errorf("...: %w", err)`.

---

## File Structure (Plan 1 deliverable)

```
better-ccusage/
├── go.mod                              # CREATE
├── go.sum                              # generated
├── .gitignore                          # MODIFY (append Go artifacts)
├── scripts/                            # CREATE
│   ├── build.sh
│   ├── test.sh
│   ├── fmt.sh
│   └── lint.sh
├── .github/workflows/ci.yml            # CREATE
└── pkg/
    ├── pricing/
    │   ├── embed.go                    # go:embed directive
    │   ├── prices.go                   # Price, PriceTable, Load
    │   ├── match.go                    # Lookup (3-tier)
    │   ├── fetcher.go                  # Fetch + atomic swap
    │   ├── money.go                    # Money type (int64 Micros)
    │   ├── prices_test.go
    │   ├── match_test.go
    │   ├── fetcher_test.go
    │   ├── money_test.go
    │   └── testdata/
    │       └── prices.json             # small fixture
    └── terminal/
        ├── logger.go                   # Logger + LOG_LEVEL
        ├── table.go                    # lipgloss table renderer
        ├── colors.go                   # consola-equivalent styles
        ├── normalize.go                # NormalizeForTest helper
        ├── logger_test.go
        ├── table_test.go
        ├── colors_test.go
        └── normalize_test.go
```

---

## Task 1: Bootstrap Go module + scripts + CI

**Files:**
- Create: `go.mod`
- Create: `scripts/build.sh`, `scripts/test.sh`, `scripts/fmt.sh`, `scripts/lint.sh`
- Create: `.github/workflows/ci.yml`
- Modify: `.gitignore` (append)

**Interfaces:**
- Produces: working `go.mod`, `go build ./...` succeeds (with no `.go` files yet), `go test ./...` succeeds with no tests, CI workflow file syntactically valid

- [ ] **Step 1: Initialize Go module**

```bash
cd /Volumes/Samsung970EVOPlus/dev-projects/better-ccusage
go mod init github.com/cobra91/better-ccusage
```

Expected: `go.mod` created with `module github.com/cobra91/better-ccusage`, `go 1.22` (or current). If `go` is not 1.22, edit the file's `go` directive to `1.22` after init.

- [ ] **Step 2: Append Go build artifacts to `.gitignore`**

Append (do NOT overwrite) the following to `.gitignore`:

```
# Go
/dist/
/vendor/
*.test
*.out
coverage.out
coverage.html
```

Verify by running:
```bash
tail -10 .gitignore
```

Expected: appended lines visible.

- [ ] **Step 3: Create `scripts/build.sh`**

```bash
#!/usr/bin/env bash
set -euo pipefail

mkdir -p dist

echo "Building better-ccusage..."
go build -trimpath -o dist/better-ccusage ./apps/better-ccusage/cmd/better-ccusage
echo "Building better-ccusage-mcp..."
go build -trimpath -o dist/better-ccusage-mcp ./apps/mcp/cmd/better-ccusage-mcp
echo "Building better-ccusage-codex..."
go build -trimpath -o dist/better-ccusage-codex ./apps/codex/cmd/better-ccusage-codex
echo "Building better-ccusage-opencode..."
go build -trimpath -o dist/better-ccusage-opencode ./apps/opencode/cmd/better-ccusage-opencode

ls -lh dist/
```

Note: the four `./apps/.../cmd/...` paths do not exist yet — that's expected; they're added in Plans 2-4. This script will succeed once those plans land.

- [ ] **Step 4: Create `scripts/test.sh`**

```bash
#!/usr/bin/env bash
set -euo pipefail
go test ./... -race -count=1
```

- [ ] **Step 5: Create `scripts/fmt.sh`**

```bash
#!/usr/bin/env bash
set -euo pipefail
gofumpt -w .
goimports -w .
```

- [ ] **Step 6: Create `scripts/lint.sh`**

```bash
#!/usr/bin/env bash
set -euo pipefail
golangci-lint run ./...
```

- [ ] **Step 7: Make scripts executable**

```bash
chmod +x scripts/*.sh
```

- [ ] **Step 8: Create `.github/workflows/ci.yml`**

```yaml
name: ci
on:
  push:
    branches: [main]
  pull_request:
jobs:
  test:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
          cache: true
      - run: ./scripts/test.sh
  build:
    runs-on: macos-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
          cache: true
      - run: ./scripts/build.sh
      - uses: actions/upload-artifact@v4
        with:
          name: binaries
          path: dist/
```

- [ ] **Step 9: Verify Go module compiles (no source yet)**

```bash
go build ./...
go test ./...
```

Expected: both succeed with no output (empty module).

- [ ] **Step 10: Commit**

```bash
git add go.mod .gitignore scripts/ .github/workflows/ci.yml
git commit -m "chore: bootstrap Go module, scripts, and macOS CI"
```

---

## Task 2: Money type

**Files:**
- Create: `pkg/pricing/money.go`
- Create: `pkg/pricing/money_test.go`

**Interfaces:**
- Produces: `type Money struct { Micros int64 }` with methods `Add`, `Sub`, `MulFloat`, `String`, `MarshalJSON`, `UnmarshalJSON`. JSON form: string with up to 6 decimal places (matches TS pricing JSON's `costUSD` shape).

- [ ] **Step 1: Write the failing test file `pkg/pricing/money_test.go`**

```go
package pricing

import (
	"encoding/json"
	"testing"
)

func TestMoney_Add(t *testing.T) {
	a := Money{Micros: 1_000_000} // $1.00
	b := Money{Micros: 250_000}   // $0.25
	got := a.Add(b)
	if got.Micros != 1_250_000 {
		t.Errorf("Add: got %d, want 1_250_000", got.Micros)
	}
}

func TestMoney_Sub(t *testing.T) {
	a := Money{Micros: 1_000_000}
	b := Money{Micros: 250_000}
	got := a.Sub(b)
	if got.Micros != 750_000 {
		t.Errorf("Sub: got %d, want 750_000", got.Micros)
	}
}

func TestMoney_MulFloat(t *testing.T) {
	// $0.000003 per token, 1500 tokens → $0.0045
	perToken := Money{Micros: 3}
	got := perToken.MulFloat(1500)
	if got.Micros != 4_500 {
		t.Errorf("MulFloat: got %d, want 4_500", got.Micros)
	}
}

func TestMoney_String(t *testing.T) {
	cases := []struct {
		in   Money
		want string
	}{
		{Money{Micros: 1_000_000}, "1.000000"},
		{Money{Micros: 250_000}, "0.250000"},
		{Money{Micros: 0}, "0.000000"},
		{Money{Micros: 1}, "0.000001"},
		{Money{Micros: 123_456_789}, "123.456789"},
	}
	for _, tc := range cases {
		if got := tc.in.String(); got != tc.want {
			t.Errorf("String(%d): got %q, want %q", tc.in.Micros, got, tc.want)
		}
	}
}

func TestMoney_JSONRoundtrip(t *testing.T) {
	cases := []Money{
		{Micros: 0},
		{Micros: 1},
		{Micros: 1_000_000},
		{Micros: 123_456_789},
	}
	for _, m := range cases {
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		var got Money
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if got.Micros != m.Micros {
			t.Errorf("roundtrip: got %d, want %d", got.Micros, m.Micros)
		}
	}
}

func TestMoney_JSONMarshalFormat(t *testing.T) {
	m := Money{Micros: 1_500_000}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `"1.500000"`
	if string(b) != want {
		t.Errorf("Marshal: got %s, want %s", string(b), want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./pkg/pricing/...
```

Expected: FAIL (package `pricing` does not exist or `Money` is undefined).

- [ ] **Step 3: Create `pkg/pricing/money.go`**

```go
package pricing

import (
	"fmt"
	"strconv"
	"strings"
)

// Money represents a USD amount in micro-dollars (1 dollar = 1_000_000 micros).
// All cost arithmetic uses int64 to avoid floating-point precision loss.
type Money struct {
	Micros int64
}

// Add returns a + b.
func (m Money) Add(b Money) Money {
	return Money{Micros: m.Micros + b.Micros}
}

// Sub returns a - b.
func (m Money) Sub(b Money) Money {
	return Money{Micros: m.Micros - b.Micros}
}

// MulFloat returns m * q, where q is a float64 quantity (e.g. token count).
// The result is rounded to the nearest micro.
func (m Money) MulFloat(q float64) Money {
	return Money{Micros: int64(float64(m.Micros)*q + 0.5)}
}

// String returns the money as a decimal string with 6 decimal places.
func (m Money) String() string {
	whole := m.Micros / 1_000_000
	frac := m.Micros % 1_000_000
	if frac < 0 {
		frac = -frac
	}
	sign := ""
	if m.Micros < 0 {
		sign = "-"
	}
	return fmt.Sprintf("%s%d.%06d", sign, whole, frac)
}

// MarshalJSON encodes the money as a JSON string with 6 decimal places.
func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(`"` + m.String() + `"`), nil
}

// UnmarshalJSON parses a JSON string or number into Money.
func (m *Money) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		m.Micros = 0
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("invalid money %q: %w", s, err)
	}
	m.Micros = int64(f * 1_000_000)
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/pricing/... -v -run TestMoney
```

Expected: all 6 tests PASS.

- [ ] **Step 5: Run race detector**

```bash
go test ./pkg/pricing/... -race
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/pricing/money.go pkg/pricing/money_test.go
git commit -m "feat(pricing): add Money type with int64 micros and JSON support"
```

---

## Task 3: Pricing embed + Load

**Files:**
- Create: `pkg/pricing/embed.go`
- Create: `pkg/pricing/testdata/prices.json`
- Create: `pkg/pricing/prices.go`
- Create: `pkg/pricing/prices_test.go`

**Interfaces:**
- Consumes: existing `Money` type from Task 2
- Produces: `type Price struct { ... }`, `type PriceTable struct { ... }`, `func LoadPrices(r io.Reader) (*PriceTable, error)`. Plus a package-level `var EmbeddedPrices []byte` populated by `go:embed`.

- [ ] **Step 1: Create `pkg/pricing/testdata/prices.json`** (small fixture)

```json
{
  "claude-sonnet-4-5-20250929": {
    "input_cost_per_token": 0.000003,
    "output_cost_per_token": 0.000015,
    "cache_creation_input_token_cost": 0.00000375,
    "cache_read_input_token_cost": 0.0000003,
    "max_tokens": 8192
  },
  "claude-opus-4-20250514": {
    "input_cost_per_token": 0.000015,
    "output_cost_per_token": 0.000075,
    "cache_creation_input_token_cost": 0.00001875,
    "cache_read_input_token_cost": 0.0000015,
    "max_tokens": 8192
  },
  "kimi-for-coding": {
    "input_cost_per_token": 0.000001,
    "output_cost_per_token": 0.000003,
    "max_tokens": 8192
  }
}
```

- [ ] **Step 2: Write the failing test file `pkg/pricing/prices_test.go`**

```go
package pricing

import (
	"math"
	"strings"
	"testing"
)

func TestLoadPrices_Success(t *testing.T) {
	r := strings.NewReader(`{
		"claude-sonnet-4-5-20250929": {
			"input_cost_per_token": 0.000003,
			"output_cost_per_token": 0.000015
		}
	}`)
	pt, err := LoadPrices(r)
	if err != nil {
		t.Fatalf("LoadPrices: %v", err)
	}
	if got := pt.entries["claude-sonnet-4-5-20250929"]; got == nil {
		t.Fatal("expected entry for claude-sonnet-4-5-20250929")
	} else if math.Abs(got.InputCostPerToken-0.000003) > 1e-9 {
		t.Errorf("input cost: got %f, want 0.000003", got.InputCostPerToken)
	}
}

func TestLoadPrices_InvalidJSON(t *testing.T) {
	r := strings.NewReader(`{not valid json`)
	_, err := LoadPrices(r)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadPrices_EmptyReader(t *testing.T) {
	r := strings.NewReader("")
	pt, err := LoadPrices(r)
	if err != nil {
		t.Fatalf("LoadPrices: %v", err)
	}
	if len(pt.entries) != 0 {
		t.Errorf("expected empty table, got %d entries", len(pt.entries))
	}
}

func TestEmbeddedPrices_NonEmpty(t *testing.T) {
	if len(EmbeddedPrices) == 0 {
		t.Fatal("EmbeddedPrices is empty — go:embed failed")
	}
	if !strings.Contains(string(EmbeddedPrices), "claude-sonnet-4-5-20250929") {
		t.Error("EmbeddedPrices missing expected model entry")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./pkg/pricing/... -v -run TestLoadPrices
```

Expected: FAIL (LoadPrices undefined).

- [ ] **Step 4: Create `pkg/pricing/embed.go`**

```go
package pricing

import _ "embed"

// EmbeddedPrices contains the bundled model_prices_and_context_window.json.
// Source: litellm/model_prices_and_context_window.json (mirrors TS).
//
//go:embed testdata/prices.json
var EmbeddedPrices []byte
```

Note: in Plan 2 we'll add the real pricing JSON. For Plan 1, the testdata fixture is sufficient for verifying the embed mechanism.

- [ ] **Step 5: Create `pkg/pricing/prices.go`**

```go
package pricing

import (
	"encoding/json"
	"fmt"
	"io"
)

// Price holds the per-token pricing for one model, in dollars (not micros).
type Price struct {
	InputCostPerToken            float64 `json:"input_cost_per_token"`
	OutputCostPerToken           float64 `json:"output_cost_per_token"`
	CacheCreationInputTokenCost  float64 `json:"cache_creation_input_token_cost,omitempty"`
	CacheReadInputTokenCost      float64 `json:"cache_read_input_token_cost,omitempty"`
	MaxTokens                    int     `json:"max_tokens,omitempty"`
}

// rawPricesFile matches the upstream JSON shape: model name → Price.
type rawPricesFile map[string]Price

// PriceTable is an immutable lookup table of model prices.
type PriceTable struct {
	entries rawPricesFile
}

// LoadPrices parses a JSON pricing file from r.
func LoadPrices(r io.Reader) (*PriceTable, error) {
	if r == nil {
		return nil, fmt.Errorf("nil reader")
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading prices: %w", err)
	}
	if len(data) == 0 {
		return &PriceTable{entries: rawPricesFile{}}, nil
	}
	var raw rawPricesFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing prices: %w", err)
	}
	if raw == nil {
		raw = rawPricesFile{}
	}
	return &PriceTable{entries: raw}, nil
}

// rawPrices returns the underlying map for internal use (e.g. fuzzy scorer).
// Returns nil if pt is nil.
func (pt *PriceTable) rawPrices() rawPricesFile {
	if pt == nil {
		return nil
	}
	return pt.entries
}
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
go test ./pkg/pricing/... -v -run 'TestLoadPrices|TestEmbeddedPrices'
```

Expected: all 4 tests PASS.

- [ ] **Step 7: Commit**

```bash
git add pkg/pricing/embed.go pkg/pricing/prices.go pkg/pricing/prices_test.go pkg/pricing/testdata/prices.json
git commit -m "feat(pricing): add Price type, PriceTable, LoadPrices, and go:embed"
```

---

## Task 4: Lookup — exact match

**Files:**
- Create: `pkg/pricing/match.go`
- Create: `pkg/pricing/match_test.go`

**Interfaces:**
- Consumes: `PriceTable`, `Price` from Tasks 2-3
- Produces: `func (pt *PriceTable) LookupExact(model string) (Price, bool)`. Returns `(Price{}, false)` if no exact match. Used as tier 1 of the 3-tier resolution in Task 5.

- [ ] **Step 1: Write the failing test file `pkg/pricing/match_test.go`** (test for tier 1 only)

```go
package pricing

import (
	"strings"
	"testing"
)

func newTestTable(t *testing.T) *PriceTable {
	t.Helper()
	r := strings.NewReader(`{
		"claude-sonnet-4-5-20250929": {
			"input_cost_per_token": 0.000003,
			"output_cost_per_token": 0.000015
		},
		"moonshot/kimi-for-coding": {
			"input_cost_per_token": 0.000001,
			"output_cost_per_token": 0.000003
		}
	}`)
	pt, err := LoadPrices(r)
	if err != nil {
		t.Fatalf("LoadPrices: %v", err)
	}
	return pt
}

func TestLookupExact_Found(t *testing.T) {
	pt := newTestTable(t)
	got, ok := pt.LookupExact("claude-sonnet-4-5-20250929")
	if !ok {
		t.Fatal("expected match")
	}
	if got.InputCostPerToken != 0.000003 {
		t.Errorf("input cost: got %f, want 0.000003", got.InputCostPerToken)
	}
}

func TestLookupExact_NotFound(t *testing.T) {
	pt := newTestTable(t)
	_, ok := pt.LookupExact("nonexistent-model")
	if ok {
		t.Error("expected no match for nonexistent-model")
	}
}

func TestLookupExact_NilTable(t *testing.T) {
	var pt *PriceTable
	_, ok := pt.LookupExact("claude-sonnet-4-5-20250929")
	if ok {
		t.Error("expected no match for nil table")
	}
}

func TestLookupExact_CaseSensitive(t *testing.T) {
	pt := newTestTable(t)
	_, ok := pt.LookupExact("Claude-Sonnet-4-5-20250929")
	if ok {
		t.Error("expected no match — lookup must be case-sensitive (matches TS)")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./pkg/pricing/... -v -run TestLookupExact
```

Expected: FAIL (LookupExact undefined).

- [ ] **Step 3: Add `LookupExact` to `pkg/pricing/match.go`**

```go
package pricing

// LookupExact returns the price for model if an exact (case-sensitive) match
// exists in the table. Returns (Price{}, false) otherwise.
func (pt *PriceTable) LookupExact(model string) (Price, bool) {
	entries := pt.rawPrices()
	if entries == nil {
		return Price{}, false
	}
	p, ok := entries[model]
	return p, ok
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/pricing/... -v -run TestLookupExact
```

Expected: 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/pricing/match.go pkg/pricing/match_test.go
git commit -m "feat(pricing): add LookupExact (tier 1 of 3-tier resolution)"
```

---

## Task 5: Lookup — provider-prefix match

**Files:**
- Modify: `pkg/pricing/match.go` (append `LookupProviderPrefix` and a combined `Lookup` that orchestrates tier 1 then tier 2)
- Modify: `pkg/pricing/match_test.go` (append tier 2 tests)

**Interfaces:**
- Consumes: `LookupExact` from Task 4
- Produces: `func (pt *PriceTable) LookupProviderPrefix(model string) (Price, bool)` — splits on `/`, takes the suffix, looks up the suffix in the table. Tier 2 of resolution. Also: `func (pt *PriceTable) Lookup(model string) (Price, float64)` orchestrating tiers 1+2 (fuzzy tier added in Task 6).

- [ ] **Step 1: Append failing tests to `pkg/pricing/match_test.go`**

```go
func TestLookupProviderPrefix_Found(t *testing.T) {
	pt := newTestTable(t)
	// "moonshot/kimi-for-coding" → after stripping provider, key becomes "kimi-for-coding"
	// which exists in the table.
	got, ok := pt.LookupProviderPrefix("moonshot/kimi-for-coding")
	if !ok {
		t.Fatal("expected provider-prefix match")
	}
	if got.InputCostPerToken != 0.000001 {
		t.Errorf("input cost: got %f, want 0.000001", got.InputCostPerToken)
	}
}

func TestLookupProviderPrefix_NoSlash(t *testing.T) {
	pt := newTestTable(t)
	_, ok := pt.LookupProviderPrefix("claude-sonnet-4-5-20250929")
	if ok {
		t.Error("expected no provider-prefix match when no slash present")
	}
}

func TestLookupProviderPrefix_SuffixNotInTable(t *testing.T) {
	pt := newTestTable(t)
	_, ok := pt.LookupProviderPrefix("acme/unknown-model")
	if ok {
		t.Error("expected no provider-prefix match when suffix not in table")
	}
}

func TestLookup_Tier1Preferred(t *testing.T) {
	// "claude-sonnet-4-5-20250929" exists exactly. Even though a provider-prefix
	// form could exist too, tier 1 wins and returns confidence 1.0.
	pt := newTestTable(t)
	_, conf := pt.Lookup("claude-sonnet-4-5-20250929")
	if conf != 1.0 {
		t.Errorf("confidence: got %f, want 1.0", conf)
	}
}

func TestLookup_Tier2Confidence(t *testing.T) {
	// "moonshot/kimi-for-coding" has tier-2 match; confidence must be in (0.0, 1.0].
	pt := newTestTable(t)
	_, conf := pt.Lookup("moonshot/kimi-for-coding")
	if conf <= 0.0 || conf > 1.0 {
		t.Errorf("confidence: got %f, want (0.0, 1.0]", conf)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./pkg/pricing/... -v -run 'TestLookupProviderPrefix|TestLookup_Tier'
```

Expected: FAIL (LookupProviderPrefix, Lookup undefined).

- [ ] **Step 3: Append `LookupProviderPrefix` and `Lookup` (tiers 1+2) to `pkg/pricing/match.go`**

```go
package pricing

import "strings"

// LookupExact returns the price for model if an exact (case-sensitive) match
// exists in the table. Returns (Price{}, false) otherwise.
func (pt *PriceTable) LookupExact(model string) (Price, bool) {
	entries := pt.rawPrices()
	if entries == nil {
		return Price{}, false
	}
	p, ok := entries[model]
	return p, ok
}

// LookupProviderPrefix splits model on "/" and looks up the suffix in the
// table. Returns (Price{}, false) if no "/" is present or the suffix is not
// found. Matches TS behavior: the suffix after the last "/" is the lookup key.
func (pt *PriceTable) LookupProviderPrefix(model string) (Price, bool) {
	entries := pt.rawPrices()
	if entries == nil {
		return Price{}, false
	}
	idx := strings.LastIndex(model, "/")
	if idx < 0 || idx == len(model)-1 {
		return Price{}, false
	}
	suffix := model[idx+1:]
	p, ok := entries[suffix]
	return p, ok
}

// Lookup performs the tier-1 + tier-2 resolution. Tier-3 (fuzzy) is added in
// the next task. Returns (Price, confidence). Confidence is 1.0 for exact
// matches and 0.85 for provider-prefix matches (matches TS scoring band).
func (pt *PriceTable) Lookup(model string) (Price, float64) {
	if p, ok := pt.LookupExact(model); ok {
		return p, 1.0
	}
	if p, ok := pt.LookupProviderPrefix(model); ok {
		return p, 0.85
	}
	return Price{}, 0.0
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/pricing/... -v -run 'TestLookupProviderPrefix|TestLookup_Tier'
```

Expected: 5 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/pricing/match.go pkg/pricing/match_test.go
git commit -m "feat(pricing): add provider-prefix match (tier 2) and combined Lookup"
```

---

## Task 6: Lookup — fuzzy match

**Files:**
- Modify: `pkg/pricing/match.go` (add `LookupFuzzy` + extend `Lookup` to use it as tier 3)
- Modify: `pkg/pricing/match_test.go` (append tier 3 tests)

**Interfaces:**
- Consumes: `Lookup`, `LookupProviderPrefix`, `LookupExact` from Tasks 4-5
- Produces: `func (pt *PriceTable) LookupFuzzy(model string, threshold float64) (Price, float64)` — partial-match scorer; returns best match whose score ≥ threshold (default 0.6 per spec). Extended `Lookup` to fall through to fuzzy tier with confidence = score.

- [ ] **Step 1: Append failing tests to `pkg/pricing/match_test.go`**

```go
func TestLookupFuzzy_TypoMatch(t *testing.T) {
	// "kimi-for-codin" (missing 'g') is a fuzzy match for "kimi-for-coding"
	pt := newTestTable(t)
	p, score := pt.LookupFuzzy("kimi-for-codin", 0.6)
	if score < 0.6 {
		t.Fatalf("expected match above threshold, got score %f", score)
	}
	if p.InputCostPerToken != 0.000001 {
		t.Errorf("matched wrong price: got %f, want 0.000001", p.InputCostPerToken)
	}
}

func TestLookupFuzzy_BelowThreshold(t *testing.T) {
	pt := newTestTable(t)
	_, score := pt.LookupFuzzy("zzzzzzzzzzz", 0.6)
	if score >= 0.6 {
		t.Errorf("expected no match above threshold, got score %f", score)
	}
}

func TestLookup_Tier3Confidence(t *testing.T) {
	// Fuzzy match — confidence must equal the score returned by LookupFuzzy.
	pt := newTestTable(t)
	_, conf := pt.Lookup("kimi-for-codin")
	if conf < 0.6 || conf > 1.0 {
		t.Errorf("confidence: got %f, want [0.6, 1.0]", conf)
	}
}

func TestLookup_NoMatchReturnsZeroConfidence(t *testing.T) {
	pt := newTestTable(t)
	_, conf := pt.Lookup("totally-unrelated-model-name-zzz")
	if conf != 0.0 {
		t.Errorf("expected 0.0 confidence for no match, got %f", conf)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./pkg/pricing/... -v -run 'TestLookupFuzzy|TestLookup_NoMatch'
```

Expected: FAIL (LookupFuzzy undefined).

- [ ] **Step 3: Add `LookupFuzzy` and extend `Lookup` in `pkg/pricing/match.go`**

```go
package pricing

import "strings"

// LookupExact returns the price for model if an exact (case-sensitive) match
// exists in the table. Returns (Price{}, false) otherwise.
func (pt *PriceTable) LookupExact(model string) (Price, bool) {
	entries := pt.rawPrices()
	if entries == nil {
		return Price{}, false
	}
	p, ok := entries[model]
	return p, ok
}

// LookupProviderPrefix splits model on "/" and looks up the suffix in the
// table. Returns (Price{}, false) if no "/" is present or the suffix is not
// found. Matches TS behavior: the suffix after the last "/" is the lookup key.
func (pt *PriceTable) LookupProviderPrefix(model string) (Price, bool) {
	entries := pt.rawPrices()
	if entries == nil {
		return Price{}, false
	}
	idx := strings.LastIndex(model, "/")
	if idx < 0 || idx == len(model)-1 {
		return Price{}, false
	}
	suffix := model[idx+1:]
	p, ok := entries[suffix]
	return p, ok
}

// LookupFuzzy scores every key in the table against model using a simple
// partial-match score (longest common prefix length / max(len(model), len(key))).
// Returns the best match whose score >= threshold, or (Price{}, 0.0) if none.
// Threshold of 0.6 matches the spec; pass 0.0 to accept any non-zero match.
func (pt *PriceTable) LookupFuzzy(model string, threshold float64) (Price, float64) {
	entries := pt.rawPrices()
	if entries == nil || model == "" {
		return Price{}, 0.0
	}
	bestPrice := Price{}
	bestScore := 0.0
	for key, p := range entries {
		s := fuzzyScore(model, key)
		if s > bestScore {
			bestScore = s
			bestPrice = p
		}
	}
	if bestScore < threshold {
		return Price{}, 0.0
	}
	return bestPrice, bestScore
}

// Lookup performs the full 3-tier resolution: exact → provider-prefix → fuzzy.
// Returns (Price, confidence). Confidence is 1.0 for exact, 0.85 for
// provider-prefix, and the fuzzy score for tier 3 (>= 0.6 to be considered a
// match). Returns (Price{}, 0.0) when no tier produces a match.
func (pt *PriceTable) Lookup(model string) (Price, float64) {
	if p, ok := pt.LookupExact(model); ok {
		return p, 1.0
	}
	if p, ok := pt.LookupProviderPrefix(model); ok {
		return p, 0.85
	}
	return pt.LookupFuzzy(model, 0.6)
}

// fuzzyScore computes a simple similarity score between a and b in [0, 1].
// The score is the longest common prefix length divided by the longer string.
func fuzzyScore(a, b string) float64 {
	la := len(a)
	lb := len(b)
	maxLen := la
	if lb > maxLen {
		maxLen = lb
	}
	if maxLen == 0 {
		return 0.0
	}
	common := 0
	n := la
	if lb < n {
		n = lb
	}
	for i := 0; i < n; i++ {
		if a[i] == b[i] {
			common++
		} else {
			break
		}
	}
	return float64(common) / float64(maxLen)
}
```

Note: this fuzzy scorer is deliberately simple (longest common prefix). It matches the TS heuristic at the tier-3 fallback. A more sophisticated scorer (Levenshtein, Jaro-Winkler) can replace it later if precision is insufficient; the tier-3 interface remains stable.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/pricing/... -v -run 'TestLookupFuzzy|TestLookup_'
```

Expected: all tests PASS.

- [ ] **Step 5: Run race detector**

```bash
go test ./pkg/pricing/... -race
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/pricing/match.go pkg/pricing/match_test.go
git commit -m "feat(pricing): add fuzzy match (tier 3) and complete 3-tier Lookup"
```

---

## Task 7: Fetch with atomic swap

**Files:**
- Create: `pkg/pricing/fetcher.go`
- Create: `pkg/pricing/fetcher_test.go`

**Interfaces:**
- Consumes: `PriceTable`, `LoadPrices` from Task 3
- Produces: `type AtomicPriceTable struct { ... }` wrapping `*PriceTable` with thread-safe `Get()` and `Swap()` methods. Plus `func (a *AtomicPriceTable) Fetch(ctx, url, client) error` which fetches a new JSON, validates it, and atomically swaps. Mirrors TS `_pricing-fetcher.ts` behavior: 5s timeout, keep old table on error.

- [ ] **Step 1: Write the failing test file `pkg/pricing/fetcher_test.go`**

```go
package pricing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAtomicPriceTable_GetSet(t *testing.T) {
	a := NewAtomicPriceTable()
	if got := a.Get(); got != nil {
		t.Errorf("Get on empty: got %v, want nil", got)
	}
	pt, err := LoadPrices(strings.NewReader(`{"m1":{"input_cost_per_token":1}}`))
	if err != nil {
		t.Fatal(err)
	}
	a.Swap(pt)
	if got := a.Get(); got != pt {
		t.Errorf("Get after Swap: got %v, want %v", got, pt)
	}
}

func TestFetch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"claude-sonnet-4-5-20250929":{"input_cost_per_token":0.000003}}`))
	}))
	defer srv.Close()

	a := NewAtomicPriceTable()
	client := srv.Client()
	client.Timeout = 2 * time.Second
	if err := a.Fetch(context.Background(), srv.URL, client); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	got := a.Get()
	if got == nil {
		t.Fatal("expected table after successful fetch")
	}
	if _, ok := got.LookupExact("claude-sonnet-4-5-20250929"); !ok {
		t.Error("expected fetched model to be lookup-able")
	}
}

func TestFetch_InvalidJSONKeepsOldTable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer srv.Close()

	a := NewAtomicPriceTable()
	old, _ := LoadPrices(strings.NewReader(`{"old-model":{"input_cost_per_token":1}}`))
	a.Swap(old)

	client := srv.Client()
	client.Timeout = 2 * time.Second
	err := a.Fetch(context.Background(), srv.URL, client)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if got := a.Get(); got != old {
		t.Error("expected old table to remain after failed fetch")
	}
}

func TestFetch_TimeoutKeepsOldTable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	a := NewAtomicPriceTable()
	old, _ := LoadPrices(strings.NewReader(`{"old":{"input_cost_per_token":1}}`))
	a.Swap(old)

	client := srv.Client()
	client.Timeout = 50 * time.Millisecond
	err := a.Fetch(context.Background(), srv.URL, client)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if got := a.Get(); got != old {
		t.Error("expected old table to remain after timeout")
	}
}

func TestAtomicPriceTable_ConcurrentReads(t *testing.T) {
	a := NewAtomicPriceTable()
	pt, _ := LoadPrices(strings.NewReader(`{"m1":{"input_cost_per_token":1}}`))
	a.Swap(pt)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = a.Get()
		}()
	}
	wg.Wait()
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./pkg/pricing/... -v -run 'TestAtomicPriceTable|TestFetch'
```

Expected: FAIL (NewAtomicPriceTable undefined).

- [ ] **Step 3: Create `pkg/pricing/fetcher.go`**

```go
package pricing

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// AtomicPriceTable holds a PriceTable that can be swapped atomically.
// Readers see a consistent snapshot; Swap is atomic.
type AtomicPriceTable struct {
	v atomic.Pointer[PriceTable]
}

// NewAtomicPriceTable returns an empty AtomicPriceTable.
func NewAtomicPriceTable() *AtomicPriceTable {
	return &AtomicPriceTable{}
}

// Get returns the current PriceTable, or nil if none has been set.
func (a *AtomicPriceTable) Get() *PriceTable {
	if a == nil {
		return nil
	}
	return a.v.Load()
}

// Swap replaces the current table with pt.
func (a *AtomicPriceTable) Swap(pt *PriceTable) {
	if a == nil || pt == nil {
		return
	}
	a.v.Store(pt)
}

// Fetch retrieves pricing JSON from url, parses it, and atomically swaps it
// into the table. On any error (network, parse, validation), the existing
// table is left untouched. Uses client.Timeout for the per-request deadline.
func (a *AtomicPriceTable) Fetch(ctx context.Context, url string, client *http.Client) error {
	if url == "" {
		return fmt.Errorf("empty pricing URL")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("fetching %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MiB cap
	if err != nil {
		return fmt.Errorf("reading body: %w", err)
	}
	pt, err := LoadPrices(strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("parsing fetched prices: %w", err)
	}
	a.Swap(pt)
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/pricing/... -v -run 'TestAtomicPriceTable|TestFetch'
```

Expected: all tests PASS.

- [ ] **Step 5: Run race detector**

```bash
go test ./pkg/pricing/... -race
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/pricing/fetcher.go pkg/pricing/fetcher_test.go
git commit -m "feat(pricing): add AtomicPriceTable and Fetch with atomic swap"
```

---

## Task 8: Logger

**Files:**
- Create: `pkg/terminal/logger.go`
- Create: `pkg/terminal/logger_test.go`

**Interfaces:**
- Produces: `type Level int` with constants `Silent=0, Warn=1, Log=2, Info=3, Debug=4, Trace=5`. `type Logger struct { ... }` with methods `Warn(string, ...any)`, `Info(string, ...any)`, `Debug(string, ...any)`, `Trace(string, ...any)`, `Fatal(string, ...any)`. `func NewLogger(level int, w io.Writer) *Logger`. Output goes to `w` (typically `os.Stderr`); format `LEVEL[ts] msg` (matches TS consola level prefix).

- [ ] **Step 1: Write the failing test file `pkg/terminal/logger_test.go`**

```go
package terminal

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogger_RespectsLevel(t *testing.T) {
	cases := []struct {
		name      string
		level     int
		wantWarn  bool
		wantInfo  bool
		wantDebug bool
		wantTrace bool
	}{
		{"Silent_0_blocks_all", 0, false, false, false, false},
		{"Warn_1_allows_warn_only", 1, true, false, false, false},
		{"Log_2_allows_warn_log", 2, true, true, false, false},
		{"Info_3_allows_info", 3, true, true, true, false},
		{"Debug_4_allows_debug", 4, true, true, true, true},
		{"Trace_5_allows_all", 5, true, true, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := NewLogger(tc.level, &buf)
			l.Warn("w")
			l.Info("i")
			l.Debug("d")
			l.Trace("t")
			out := buf.String()
			if got := strings.Contains(out, "w"); got != tc.wantWarn {
				t.Errorf("warn: got %v, want %v", got, tc.wantWarn)
			}
			if got := strings.Contains(out, "i"); got != tc.wantInfo {
				t.Errorf("info: got %v, want %v", got, tc.wantInfo)
			}
			if got := strings.Contains(out, "d"); got != tc.wantDebug {
				t.Errorf("debug: got %v, want %v", got, tc.wantDebug)
			}
			if got := strings.Contains(out, "t"); got != tc.wantTrace {
				t.Errorf("trace: got %v, want %v", got, tc.wantTrace)
			}
		})
	}
}

func TestLogger_OutputFormat(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger(Log, &buf)
	l.Info("hello %s", "world")
	out := buf.String()
	if !strings.Contains(out, "hello world") {
		t.Errorf("expected formatted message, got %q", out)
	}
	// Format: "INFO  hello world\n" — level prefix + space + msg + newline
	if !strings.HasPrefix(out, "INFO ") {
		t.Errorf("expected INFO prefix, got %q", out)
	}
}

func TestLogger_NilWriterDoesNotPanic(t *testing.T) {
	l := NewLogger(Trace, nil)
	l.Info("noop")
	// Just verifying no panic
}

func TestNewLoggerFromEnv(t *testing.T) {
	cases := []struct {
		env  string
		want Level
	}{
		{"", Silent},
		{"0", Silent},
		{"1", Warn},
		{"3", Info},
		{"5", Trace},
		{"invalid", Silent}, // graceful fallback
	}
	for _, tc := range cases {
		t.Run("env="+tc.env, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", tc.env)
			if got := NewLoggerFromEnv(); got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./pkg/terminal/... -v
```

Expected: FAIL (package `terminal` does not exist).

- [ ] **Step 3: Create `pkg/terminal/logger.go`**

```go
// Package terminal provides shared CLI rendering primitives (tables, colors,
// loggers) for better-ccusage Go binaries. Mirrors the TS packages/terminal
// package and the cli-table3 + consola output.
package terminal

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"
)

type Level int

const (
	Silent Level = iota
	Warn
	Log
	Info
	Debug
	Trace
)

// Logger writes leveled messages to an io.Writer. Concurrent-safe.
type Logger struct {
	mu  sync.Mutex
	w   io.Writer
	lvl Level
}

// NewLogger returns a Logger that writes to w at the given level.
// Pass Silent (0) to disable all output.
func NewLogger(level int, w io.Writer) *Logger {
	if w == nil {
		w = io.Discard
	}
	return &Logger{w: w, lvl: Level(level)}
}

// NewLoggerFromEnv parses LOG_LEVEL and returns a Logger writing to stderr.
func NewLoggerFromEnv() *Logger {
	return NewLogger(parseLevel(os.Getenv("LOG_LEVEL")), os.Stderr)
}

func parseLevel(s string) int {
	if s == "" {
		return int(Silent)
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 || v > int(Trace) {
		return int(Silent)
	}
	return v
}

// logf writes a message at the given level if level <= l.lvl.
func (l *Logger) logf(level Level, tag, format string, args ...any) {
	if level > l.lvl || l.w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.w, "%s %s\n", tag, msg)
}

// Warn logs at Warn level.
func (l *Logger) Warn(format string, args ...any) { l.logf(Warn, "WARN ", format, args...) }

// Info logs at Info level.
func (l *Logger) Info(format string, args ...any) { l.logf(Info, "INFO ", format, args...) }

// Log logs at Log level (consola "log" mapping).
func (l *Logger) Log(format string, args ...any) { l.logf(Log, "LOG  ", format, args...) }

// Debug logs at Debug level.
func (l *Logger) Debug(format string, args ...any) { l.logf(Debug, "DEBUG", format, args...) }

// Trace logs at Trace level.
func (l *Logger) Trace(format string, args ...any) { l.logf(Trace, "TRACE", format, args...) }

// Fatal logs the message at Warn level and exits with code 1.
func (l *Logger) Fatal(format string, args ...any) {
	l.Warn(format, args...)
	os.Exit(1)
}

// WithTimestamp returns a copy of the logger that prefixes each message with
// an RFC3339 timestamp. Used by tests; not enabled by default.
func (l *Logger) WithTimestamp() *Logger {
	// Not used in production output — kept for future expansion.
	_ = time.RFC3339
	return l
}
```

Note: The `WithTimestamp` method is a placeholder to keep the package surface stable; it's not on the hot path. Safe to remove if it adds confusion.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/terminal/... -v -run TestLogger
```

Expected: all tests PASS.

- [ ] **Step 5: Run race detector**

```bash
go test ./pkg/terminal/... -race
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/terminal/logger.go pkg/terminal/logger_test.go
git commit -m "feat(terminal): add leveled Logger with LOG_LEVEL semantics"
```

---

## Task 9: Colors helpers

**Files:**
- Create: `pkg/terminal/colors.go`
- Create: `pkg/terminal/colors_test.go`
- Modify: `go.mod` (add lipgloss dep)

**Interfaces:**
- Produces: `type Style int` with constants `StyleHeader, StyleMuted, StyleCost, StyleWarning, StyleError, StyleSuccess`. `func StyleText(s Style, text string) string` returns the lipgloss-styled text. Constants are exposed so tests can assert on tag presence without depending on exact ANSI codes.

- [ ] **Step 1: Add lipgloss dependency**

```bash
cd /Volumes/Samsung970EVOPlus/dev-projects/better-ccusage
go get github.com/charmbracelet/lipgloss@latest
```

- [ ] **Step 2: Write the failing test file `pkg/terminal/colors_test.go`**

```go
package terminal

import (
	"strings"
	"testing"
)

func TestStyleText_NotEmpty(t *testing.T) {
	cases := []Style{StyleHeader, StyleMuted, StyleCost, StyleWarning, StyleError, StyleSuccess}
	for _, s := range cases {
		t.Run("style", func(t *testing.T) {
			got := StyleText(s, "hello")
			if got == "" {
				t.Errorf("StyleText(%d) returned empty", s)
			}
			if !strings.Contains(got, "hello") {
				t.Errorf("StyleText(%d) lost the input text", s)
			}
		})
	}
}

func TestStyleText_EmptyInput(t *testing.T) {
	got := StyleText(StyleHeader, "")
	if got != "" {
		t.Errorf("expected empty output for empty input, got %q", got)
	}
}

func TestStripANSI(t *testing.T) {
	in := "\x1b[31mred\x1b[0m plain \x1b[1;32mgreen\x1b[0m"
	want := "red plain green"
	if got := StripANSI(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./pkg/terminal/... -v -run 'TestStyleText|TestStripANSI'
```

Expected: FAIL (StyleText, StyleHeader, StripANSI undefined).

- [ ] **Step 4: Create `pkg/terminal/colors.go`**

```go
package terminal

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Style is a named text style matching the consola color tags used in TS.
type Style int

const (
	StyleHeader Style = iota
	StyleMuted
	StyleCost
	StyleWarning
	StyleError
	StyleSuccess
)

var styles = map[Style]lipgloss.Style{
	StyleHeader:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),  // bright blue
	StyleMuted:   lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("245")), // gray
	StyleCost:    lipgloss.NewStyle().Foreground(lipgloss.Color("10")),              // green
	StyleWarning: lipgloss.NewStyle().Foreground(lipgloss.Color("11")),              // yellow
	StyleError:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")),    // red
	StyleSuccess: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")),   // green
}

// StyleText returns the styled form of text. Returns "" for empty input so
// callers can chain without checking.
func StyleText(s Style, text string) string {
	if text == "" {
		return ""
	}
	return styles[s].Render(text)
}

// ansiEscape matches ANSI CSI sequences for stripping in tests and NormalizeForTest.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes all ANSI escape sequences from s.
func StripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

// TrimTrailingSpaces removes trailing spaces from each line of s. Used by
// normalize helpers to make golden files stable across terminal widths.
func TrimTrailingSpaces(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./pkg/terminal/... -v -run 'TestStyleText|TestStripANSI'
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/terminal/colors.go pkg/terminal/colors_test.go go.mod go.sum
git commit -m "feat(terminal): add StyleText and StripANSI helpers (lipgloss)"
```

---

## Task 10: Table renderer

**Files:**
- Create: `pkg/terminal/table.go`
- Create: `pkg/terminal/table_test.go`

**Interfaces:**
- Consumes: `Style` and `StripANSI` from Task 9
- Produces: `type Column struct { Header string; Width int; Align lipgloss.Position; Style Style }`. `type Table struct { ... }` with `NewTable(w io.Writer, cols ...Column) *Table`, `(*Table).SetRows(rows [][]string) *Table`, `(*Table).Render() error`. Output uses lipgloss borders + styled headers, matching the TS `cli-table3` look.

- [ ] **Step 1: Write the failing test file `pkg/terminal/table_test.go`**

```go
package terminal

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestTable_BasicRender(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable(&buf,
		Column{Header: "Model", Width: 20},
		Column{Header: "Cost", Width: 10, Align: lipgloss.Right},
	)
	tbl.SetRows([][]string{
		{"claude-sonnet-4-5", "0.50"},
		{"kimi-for-coding", "0.10"},
	})
	if err := tbl.Render(); err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Model") {
		t.Errorf("missing Model header in output:\n%s", out)
	}
	if !strings.Contains(out, "claude-sonnet-4-5") {
		t.Errorf("missing row 1 in output:\n%s", out)
	}
	if !strings.Contains(out, "kimi-for-coding") {
		t.Errorf("missing row 2 in output:\n%s", out)
	}
	if !strings.Contains(out, "0.50") {
		t.Errorf("missing cost value in output:\n%s", out)
	}
}

func TestTable_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable(&buf,
		Column{Header: "Model", Width: 10},
	)
	if err := tbl.Render(); err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Model") {
		t.Errorf("expected header even with no rows:\n%s", out)
	}
}

func TestTable_ColumnWidthRespected(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable(&buf,
		Column{Header: "X", Width: 10},
	)
	tbl.SetRows([][]string{{"short"}, {"a-much-longer-cell"}}) // longer than width
	if err := tbl.Render(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Even with overflow, the value should be present
	if !strings.Contains(out, "a-much-longer-cell") {
		t.Error("expected overflow cell to appear in output")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./pkg/terminal/... -v -run TestTable
```

Expected: FAIL (NewTable, Column, Render undefined).

- [ ] **Step 3: Create `pkg/terminal/table.go`**

```go
package terminal

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
)

// Column defines one column in a Table.
type Column struct {
	Header string
	Width  int
	Align  lipgloss.Position // lipgloss.Left, lipgloss.Center, lipgloss.Right
	Style  Style             // optional: applies a Style to the column cells
}

// Table renders rows of strings as a styled ASCII table using lipgloss.
type Table struct {
	w    io.Writer
	cols []Column
	rows [][]string
}

// NewTable constructs a Table that writes to w with the given columns.
func NewTable(w io.Writer, cols ...Column) *Table {
	if w == nil {
		w = io.Discard
	}
	// Apply default alignment (left) where unset.
	for i := range cols {
		if cols[i].Align == 0 {
			cols[i].Align = lipgloss.Left
		}
		if cols[i].Width == 0 {
			cols[i].Width = 12
		}
	}
	return &Table{w: w, cols: cols}
}

// SetRows replaces the table body.
func (t *Table) SetRows(rows [][]string) *Table {
	t.rows = rows
	return t
}

// Render writes the table to the configured writer.
func (t *Table) Render() error {
	if t.w == nil {
		return nil
	}
	headers := make([]string, len(t.cols))
	widths := make([]int, len(t.cols))
	for i, c := range t.cols {
		headers[i] = StyleText(StyleHeader, c.Header)
		widths[i] = c.Width
	}
	border := lipgloss.NormalBorder()
	headerStyle := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	cellStyles := make([]lipgloss.Style, len(t.cols))
	for i, c := range t.cols {
		s := lipgloss.NewStyle().Width(c.Width).Align(c.Align).Padding(0, 1)
		if c.Style != 0 {
			s = s.Inherit(styles[c.Style])
		}
		cellStyles[i] = s
	}
	tbl := lipgloss.NewTable().
		Border(border).
		BorderStyle(lipgloss.NewStyle()).
		Headers(headers...).
		Width(0) // auto
	for _, row := range t.rows {
		styled := make([]string, len(row))
		for i, cell := range row {
			idx := i
			if idx >= len(t.cols) {
				idx = len(t.cols) - 1
			}
			styled[i] = cellStyles[idx].Render(cell)
		}
		tbl.Row(styled...)
	}
	_, _ = fmt.Fprintln(t.w, tbl.String())
	_ = headerStyle // silence unused
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/terminal/... -v -run TestTable
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/terminal/table.go pkg/terminal/table_test.go
git commit -m "feat(terminal): add lipgloss-based Table renderer"
```

---

## Task 11: NormalizeForTest helper

**Files:**
- Create: `pkg/terminal/normalize.go`
- Create: `pkg/terminal/normalize_test.go`

**Interfaces:**
- Produces: `func NormalizeForTest(s string) string` — strips ANSI escapes, normalizes line endings to LF, trims trailing whitespace per line, and collapses runs of 3+ blank lines to 2. Used by golden-file tests in Plans 2-4.

- [ ] **Step 1: Write the failing test file `pkg/terminal/normalize_test.go`**

```go
package terminal

import (
	"strings"
	"testing"
)

func TestNormalizeForTest_StripsANSI(t *testing.T) {
	in := "\x1b[31mred\x1b[0m"
	want := "red"
	if got := NormalizeForTest(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeForTest_NormalizesLineEndings(t *testing.T) {
	in := "a\r\nb\rc\n"
	want := "a\nb\nc\n"
	if got := NormalizeForTest(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeForTest_TrimsTrailingWhitespace(t *testing.T) {
	in := "a   \nb\t\t\nc\n"
	want := "a\nb\nc\n"
	if got := NormalizeForTest(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeForTest_CollapsesBlankLines(t *testing.T) {
	in := "a\n\n\n\n\nb\n"
	want := "a\n\nb\n"
	if got := NormalizeForTest(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeForTest_Empty(t *testing.T) {
	if got := NormalizeForTest(""); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
	if got := NormalizeForTest(strings.Repeat("\n", 5)); got != "\n" {
		t.Errorf("expected single newline for blank-only input, got %q", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./pkg/terminal/... -v -run TestNormalizeForTest
```

Expected: FAIL (NormalizeForTest undefined).

- [ ] **Step 3: Create `pkg/terminal/normalize.go`**

```go
package terminal

import (
	"regexp"
	"strings"
)

// blankLineRun matches 3+ consecutive newlines for collapse.
var blankLineRun = regexp.MustCompile(`\n{3,}`)

// NormalizeForTest prepares a string for stable golden-file comparison:
//   1. Strips ANSI escape sequences
//   2. Normalizes CRLF and CR to LF
//   3. Trims trailing whitespace per line
//   4. Collapses runs of 3+ blank lines to 2 (one blank line)
func NormalizeForTest(s string) string {
	if s == "" {
		return ""
	}
	s = StripANSI(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = TrimTrailingSpaces(s)
	s = blankLineRun.ReplaceAllString(s, "\n\n")
	// Trim leading/trailing blank lines to a single trailing newline.
	s = strings.TrimLeft(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	return s
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/terminal/... -v -run TestNormalizeForTest
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/terminal/normalize.go pkg/terminal/normalize_test.go
git commit -m "feat(terminal): add NormalizeForTest helper for golden-file tests"
```

---

## Task 12: Foundation verification

**Files:** none (verification only)

- [ ] **Step 1: Run the full test suite with race detector**

```bash
cd /Volumes/Samsung970EVOPlus/dev-projects/better-ccusage
go test ./... -race -count=1
```

Expected: all tests PASS, no race warnings.

- [ ] **Step 2: Run `go vet`**

```bash
go vet ./...
```

Expected: no diagnostics.

- [ ] **Step 3: Verify module builds cleanly**

```bash
go build ./...
```

Expected: succeeds with no output.

- [ ] **Step 4: Check that all commit messages follow Conventional Commits**

```bash
git log --oneline -12
```

Expected: each commit starts with `feat:`, `fix:`, `chore:`, `docs:`, `test:`, or `refactor:` and includes a scope where applicable (`feat(pricing):`, `feat(terminal):`).

- [ ] **Step 5: Run the script that will be used in CI**

```bash
./scripts/test.sh
```

Expected: all tests pass with `-race` flag.

- [ ] **Step 6: Final commit (only if any fixup changes were needed)**

If Steps 1-5 all pass without modifications, skip this step. Otherwise:

```bash
git add -A
git commit -m "chore: foundation plan verification fixups"
```

- [ ] **Step 7: Report status to the user**

Tell the user:
- Plan 1 is complete.
- Total tasks completed: 12.
- Total commits: ~12.
- Test counts: `go test ./... -race` passes.
- Ready to begin Plan 2 (Core CLI) when approved.

---

## End of Plan 1

The next plan (Plan 2: Core CLI) depends on Plan 1's deliverables:
- `pkg/pricing.Money`, `PriceTable`, `Lookup`, `AtomicPriceTable`, `Fetch`
- `pkg/terminal.Logger`, `Table`, `StyleText`, `StripANSI`, `NormalizeForTest`
- Working `go.mod`, scripts, CI

Plan 2 introduces the `cobra` CLI framework and implements the JSONL data loader, cost aggregation, provider adapters, and the six commands (`daily`, `monthly`, `session`, `blocks`, `statusline`, `weekly`).
